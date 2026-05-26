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
	
	launcherManager.SetOnServiceDeactivated(func(serviceID string) {
		_ = serviceDiscovery.InvalidateServiceCacheByID(serviceID)
	})

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
	slog.Error("proxy error", "error", err, "host", r.Host, "path", r.URL.Path)

	statusCode := http.StatusBadGateway
	message := "upstream error: the service is temporarily unavailable"

	var netErr *net.OpError
	if errors.As(err, &netErr) {
		if netErr.Op == "dial" {
			statusCode = http.StatusServiceUnavailable
			message = "service unavailable: the instance is not running or is starting up"
		} else if netErr.Timeout() {
			statusCode = http.StatusGatewayTimeout
			message = "gateway timeout: the service took too long to respond"
		}
	}

	http.Error(w, message, statusCode)
}

const superusersEndpoint = "/api/collections/_superusers/records"

func (rp *DynamicReverseProxyDiscovery) proxyModifyResponse(r *http.Response) error {
	origin := r.Request.Header.Get("Origin")
	if origin != "" {
		r.Header.Set("Access-Control-Allow-Origin", origin)
		r.Header.Set("Access-Control-Allow-Credentials", "true")
	}

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
			serviceID := *target.Service
			service, err := rp.serviceDiscovery.FindRunningServiceByID(ctx, serviceID)
			
			// Si no hay error en BD y el servicio realmente corre en memoria
			if err == nil && rp.launcherManager.IsServiceRunning(serviceID) {
				rp.launcherManager.RecordActivity(serviceID)
				return rp.buildReverseProxy(&url.URL{
					Scheme: "http",
					Host:   net.JoinHostPort(service.IP, strconv.Itoa(service.Port)),
				}), nil
			}

			// Si no corre en memoria, o no está marcado en ejecución en BD, lo despertamos
			if errors.Is(err, repositories.ErrNotFound) || (err == nil && !rp.launcherManager.IsServiceRunning(serviceID)) {
				ip, port, wakeupErr := rp.launcherManager.WakeupService(ctx, serviceID)
				if wakeupErr == nil {
					rp.launcherManager.RecordActivity(serviceID)
					_ = rp.serviceDiscovery.InvalidateServiceCacheByID(serviceID)
					return rp.buildReverseProxy(&url.URL{
						Scheme: "http",
						Host:   net.JoinHostPort(ip, strconv.Itoa(port)),
					}), nil
				}
				return nil, fmt.Errorf("failed to wake up service %s: %w", serviceID, wakeupErr)
			}
			if err != nil {
				return nil, fmt.Errorf("service not found for id: %s", serviceID)
			}
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
	if err == nil && rp.launcherManager.IsServiceRunning(id) {
		rp.launcherManager.RecordActivity(id)
		return rp.buildReverseProxy(&url.URL{
			Scheme: "http",
			Host:   net.JoinHostPort(service.IP, strconv.Itoa(service.Port)),
		}), nil
	}

	// Si no corre en memoria, o no está marcado en ejecución en BD
	if errors.Is(err, repositories.ErrNotFound) || (err == nil && !rp.launcherManager.IsServiceRunning(id)) {
		ip, port, wakeupErr := rp.launcherManager.WakeupService(ctx, id)
		if wakeupErr == nil {
			rp.launcherManager.RecordActivity(id)
			_ = rp.serviceDiscovery.InvalidateServiceCacheByID(id)
			return rp.buildReverseProxy(&url.URL{
				Scheme: "http",
				Host:   net.JoinHostPort(ip, strconv.Itoa(port)),
			}), nil
		}
	} else if err != nil {
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
