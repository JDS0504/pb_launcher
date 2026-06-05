package proxy

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
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

// gzipCompressible contiene las extensiones para las que se sirve el .gz pre-comprimido.
// Debe coincidir con compressibleForGzip en internal/filemanager/manager.go (SSOT en docs).
var gzipCompressible = map[string]bool{
	".html": true,
	".js":   true,
	".css":  true,
	".json": true,
	".svg":  true,
	".xml":  true,
	".txt":  true,
	".wasm": true,
	".map":  true,
}

// gzipOriginalMimeHeader es el header interno usado para pasar el MIME type original
// desde el Director hasta ModifyResponse. PocketBase lo ignora.
const gzipOriginalMimeHeader = "X-Gz-Orig-Mime"

// rewriteRequestForGzip reescribe el path a <path>.gz cuando:
//  1. El browser acepta gzip (Accept-Encoding: gzip)
//  2. La extensión del archivo es comprimible
//  3. No es una ruta de API o panel admin de PocketBase
//
// PocketBase sirve el .gz como archivo estático. ModifyResponse
// corrige el Content-Type y añade Content-Encoding: gzip.
func rewriteRequestForGzip(req *http.Request) {
	if !strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
		return
	}
	urlPath := req.URL.Path
	// No tocar rutas de API ni panel admin de PocketBase.
	if strings.HasPrefix(urlPath, "/api/") || strings.HasPrefix(urlPath, "/_/") {
		return
	}
	ext := path.Ext(urlPath)
	if ext == "" || !gzipCompressible[strings.ToLower(ext)] {
		return
	}

	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Guardar el MIME original para restaurarlo en ModifyResponse.
	req.Header.Set(gzipOriginalMimeHeader, mimeType)
	// PocketBase servirá el .gz como archivo estático.
	req.URL.Path = urlPath + ".gz"
	if req.URL.RawPath != "" {
		req.URL.RawPath = req.URL.RawPath + ".gz"
	}
	// Eliminar Accept-Encoding para que PocketBase no intente comprimir el .gz ya comprimido.
	req.Header.Del("Accept-Encoding")
}

func (rp *DynamicReverseProxyDiscovery) proxyModifyResponse(r *http.Response) error {
	origin := r.Request.Header.Get("Origin")
	if origin != "" {
		r.Header.Set("Access-Control-Allow-Origin", origin)
		r.Header.Set("Access-Control-Allow-Credentials", "true")
	}

	// Si reescribimos la URL a .gz y PocketBase respondió con éxito:
	// corregir Content-Type al tipo original del archivo y añadir Content-Encoding.
	// Content-Length se mantiene: es el tamaño del .gz, que es lo que el browser recibe.
	if origMime := r.Request.Header.Get(gzipOriginalMimeHeader); origMime != "" && r.StatusCode == http.StatusOK {
		r.Header.Set("Content-Encoding", "gzip")
		r.Header.Set("Content-Type", origMime)
		// Vary indica a proxies/CDN que la respuesta varía según el encoding soportado.
		r.Header.Set("Vary", "Accept-Encoding")
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
		// Reescribir a .gz si el browser acepta gzip y el archivo es comprimible.
		rewriteRequestForGzip(req)
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
