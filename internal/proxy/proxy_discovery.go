package proxy

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"pb_launcher/configs"
	launcherdomain "pb_launcher/internal/launcher/domain"
	proxydomain "pb_launcher/internal/proxy/domain"
	"pb_launcher/internal/proxy/domain/repositories"
	"pb_launcher/utils/networktools"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase/apis"
)

type DynamicReverseProxyDiscovery struct {
	serviceDiscovery    *proxydomain.ServiceDiscovery
	proxyEntryDiscovery *proxydomain.ProxyEntryDiscovery
	domainDiscovery     *proxydomain.DomainServiceDiscovery
	installTokenUsecase *launcherdomain.CleanServiceInstallTokenUsecase
	launcherManager     *launcherdomain.LauncherManager
	apiDomain           string
	internalApiAddress  string
}

func NewDynamicReverseProxyDiscovery(
	serviceDiscovery *proxydomain.ServiceDiscovery,
	proxyEntryDiscovery *proxydomain.ProxyEntryDiscovery,
	domainDiscovery *proxydomain.DomainServiceDiscovery,
	installTokenUsecase *launcherdomain.CleanServiceInstallTokenUsecase,
	launcherManager *launcherdomain.LauncherManager,
	cfg configs.Config,
	pbConf *apis.ServeConfig) *DynamicReverseProxyDiscovery {
	return &DynamicReverseProxyDiscovery{
		serviceDiscovery:    serviceDiscovery,
		proxyEntryDiscovery: proxyEntryDiscovery,
		domainDiscovery:     domainDiscovery,
		installTokenUsecase: installTokenUsecase,
		launcherManager:     launcherManager,
		apiDomain:           cfg.GetDomain(),
		internalApiAddress:  pbConf.HttpAddr,
	}

}

func (rp *DynamicReverseProxyDiscovery) extractID(host string) (string, error) {
	if host == rp.apiDomain {
		return "", fmt.Errorf("invalid ID: host is the base domain")
	}
	suffix := "." + rp.apiDomain
	if !strings.HasSuffix(host, suffix) {
		return "", nil
	}
	id := strings.TrimSuffix(host, suffix)
	if id == "" {
		return "", fmt.Errorf("invalid ID: prefix is empty")
	}
	if strings.Contains(id, ".") {
		return "", fmt.Errorf("invalid ID: prefix contains invalid character '.'")
	}
	return id, nil
}

func (rp *DynamicReverseProxyDiscovery) proxyErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("proxy error", "error", err)
	http.Error(w, "upstream error", http.StatusBadGateway)
}

const superusersEndpoint = "/api/collections/_superusers/records"

func (rp *DynamicReverseProxyDiscovery) proxyModifyResponse(r *http.Response) error {
	if r.Request.Method == http.MethodPost &&
		strings.HasPrefix(r.Request.URL.Path, superusersEndpoint) &&
		r.StatusCode == 200 {
		authorization := r.Request.Header.Get("Authorization")
		rp.installTokenUsecase.CleanInstallToken(r.Request.Context(), authorization)
	}
	return nil
}

func (rp *DynamicReverseProxyDiscovery) buildReverseProxy(target *url.URL) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		networktools.PrepareProxyHeaders(req, target)
	}
	proxy.ModifyResponse = rp.proxyModifyResponse
	proxy.ErrorHandler = rp.proxyErrorHandler
	return proxy
}

func (rp *DynamicReverseProxyDiscovery) ResolveTarget(ctx context.Context, host string) (*httputil.ReverseProxy, error) {
	if host == rp.apiDomain {
		return rp.buildReverseProxy(&url.URL{
			Scheme: "http",
			Host:   rp.internalApiAddress,
		}), nil
	}

	id, err := rp.extractID(host)
	if err != nil {
		return nil, err
	}

	if id == "" {
		target, err := rp.domainDiscovery.FindTargetByDomain(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("no target found for domain: %s", host)
		}
		if target.Service != nil {
			service, err := rp.serviceDiscovery.FindRunningServiceByID(ctx, *target.Service)
			if err != nil {
				if errors.Is(err, repositories.ErrNotFound) {
					ip, port, wakeupErr := rp.launcherManager.WakeupService(ctx, *target.Service)
					if wakeupErr == nil {
						rp.launcherManager.RecordActivity(*target.Service)
						return rp.buildReverseProxy(&url.URL{
							Scheme: "http",
							Host:   net.JoinHostPort(ip, strconv.Itoa(port)),
						}), nil
					}
					return nil, fmt.Errorf("failed to wake up service %s: %w", *target.Service, wakeupErr)
				}
				return nil, fmt.Errorf("service not found for id: %s", *target.Service)
			}
			rp.launcherManager.RecordActivity(*target.Service)
			return rp.buildReverseProxy(&url.URL{
				Scheme: "http",
				Host:   net.JoinHostPort(service.IP, strconv.Itoa(service.Port)),
			}), nil
		}
		if target.ProxyEntry != nil {
			entry, err := rp.proxyEntryDiscovery.FindEnabledProxyEntryByID(ctx, *target.ProxyEntry)
			if err != nil {
				return nil, fmt.Errorf("proxy entry not found for id: %s", *target.ProxyEntry)
			}
			targetURL, err := url.Parse(entry.TargetUrl)
			if err != nil {
				return nil, fmt.Errorf("failed to parse target URL: %s", entry.TargetUrl)
			}
			return rp.buildReverseProxy(targetURL), nil
		}
		return nil, fmt.Errorf("no target found for domain: %s", host)
	}

	service, err := rp.serviceDiscovery.FindRunningServiceByID(ctx, id)
	if err == nil {
		rp.launcherManager.RecordActivity(id)
		return rp.buildReverseProxy(&url.URL{
			Scheme: "http",
			Host:   net.JoinHostPort(service.IP, strconv.Itoa(service.Port)),
		}), nil
	}
	if errors.Is(err, repositories.ErrNotFound) {
		ip, port, wakeupErr := rp.launcherManager.WakeupService(ctx, id)
		if wakeupErr == nil {
			rp.launcherManager.RecordActivity(id)
			return rp.buildReverseProxy(&url.URL{
				Scheme: "http",
				Host:   net.JoinHostPort(ip, strconv.Itoa(port)),
			}), nil
		}
	} else {
		return nil, fmt.Errorf("failed to resolve service by id: %s", id)
	}

	entry, err := rp.proxyEntryDiscovery.FindEnabledProxyEntryByID(ctx, id)
	if err == nil {
		targetURL, err := url.Parse(entry.TargetUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to parse target URL: %s", entry.TargetUrl)
		}
		return rp.buildReverseProxy(targetURL), nil
	}
	if !errors.Is(err, repositories.ErrNotFound) {
		return nil, fmt.Errorf("failed to resolve proxy entry by id: %s", id)
	}
	return nil, fmt.Errorf("no target found for host: %s with id: %s", host, id)
}
