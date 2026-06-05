package proxy

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"pb_launcher/configs"
	launcherdomain "pb_launcher/internal/launcher/domain"
	proxydomain "pb_launcher/internal/proxy/domain"
	"pb_launcher/internal/proxy/domain/repositories"
	"pb_launcher/utils/networktools"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase/apis"
)

// spaFileServer es un http.Handler que sirve archivos estáticos desde staticDir.
// Si el archivo pedido no existe, entrega index.html para soportar SPAs con client-side routing.
type spaFileServer struct {
	staticDir string
	inner     http.Handler
}

func newSpaFileServer(staticDir string) http.Handler {
	return &spaFileServer{
		staticDir: staticDir,
		inner:     http.FileServer(http.Dir(staticDir)),
	}
}

func (s *spaFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cleanPath := filepath.Join(s.staticDir, filepath.FromSlash(path.Clean("/"+r.URL.Path)))
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		// Archivo no encontrado → entregar index.html (SPA fallback)
		http.ServeFile(w, r, filepath.Join(s.staticDir, "index.html"))
		return
	}
	s.inner.ServeHTTP(w, r)
}

type DynamicReverseProxyDiscovery struct {
	serviceDiscovery    *proxydomain.ServiceDiscovery
	proxyEntryDiscovery *proxydomain.ProxyEntryDiscovery
	domainDiscovery     *proxydomain.DomainServiceDiscovery
	installTokenUsecase *launcherdomain.CleanServiceInstallTokenUsecase
	launcherManager     *launcherdomain.LauncherManager
	apiDomain           string
	internalApiAddress  string
	dataDir             string
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
		dataDir:             cfg.GetDataDir(),
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

// Headers internos para coordinar Director <-> ModifyResponse.
// PocketBase los ignora al recibirlos como headers de request forwarded.
const (
	// gzHeaderOrigPath: ruta URL original antes de reescribir a .gz
	gzHeaderOrigPath = "X-Gz-Orig-Path"
	// gzHeaderPrecomp: el .gz ya estaba en disco — solo fijar headers de respuesta
	gzHeaderPrecomp = "X-Gz-Precomp"
	// gzHeaderCachePath: ruta en disco donde guardar el .gz generado on-the-fly
	gzHeaderCachePath = "X-Gz-Cache-Path"
	// gzHeaderAccept: el browser original aceptaba gzip (guardado antes de eliminar Accept-Encoding)
	gzHeaderAccept = "X-Gz-Accept"
)

// gzipCompressibleExts contiene extensiones que vale la pena comprimir.
var gzipCompressibleExts = map[string]bool{
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

// isCompressibleMime retorna true para Content-Types que valen la pena comprimir.
func isCompressibleMime(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.HasPrefix(ct, "text/") ||
		strings.HasPrefix(ct, "application/javascript") ||
		strings.HasPrefix(ct, "application/json") ||
		strings.HasPrefix(ct, "application/xml") ||
		strings.HasPrefix(ct, "application/wasm") ||
		strings.HasPrefix(ct, "image/svg")
}

// gzDiskPath calcula la ruta en disco del .gz para un serviceID y URL path dados.
// Retorna ("", false) si el path no es elegible para compresion (API, admin, extension no comprimible).
func (rp *DynamicReverseProxyDiscovery) gzDiskPath(serviceID, urlPath string) (string, bool) {
	if serviceID == "" {
		return "", false
	}
	if strings.HasPrefix(urlPath, "/api/") || strings.HasPrefix(urlPath, "/_/") {
		return "", false
	}
	ext := path.Ext(urlPath)
	if ext == "" || !gzipCompressibleExts[strings.ToLower(ext)] {
		return "", false
	}
	cleanPath := filepath.FromSlash(strings.TrimPrefix(urlPath, "/"))
	diskPath := filepath.Join(rp.dataDir, serviceID, "pb_public", cleanPath) + ".gz"
	return diskPath, true
}

// buildReverseProxy crea un proxy hacia target.
// serviceID habilita el patron lazy-compress + cache a disco:
//   - Si .gz existe en disco -> reescribe URL a .gz (0 CPU, maxima velocidad)
//   - Si .gz no existe -> comprime on-the-fly nivel 6 + guarda .gz a disco para proximos requests
//   - Si serviceID == "" -> proxy transparente sin compresion (entradas proxy, API domain)
func (rp *DynamicReverseProxyDiscovery) buildReverseProxy(target *url.URL, serviceID string) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		networktools.PrepareProxyHeaders(req, target)

		if !strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
			return
		}
		diskPath, ok := rp.gzDiskPath(serviceID, req.URL.Path)
		if !ok {
			return
		}

		origPath := req.URL.Path
		// Eliminar Accept-Encoding: nosotros manejamos la compresion, no PocketBase.
		req.Header.Del("Accept-Encoding")

		if _, statErr := os.Stat(diskPath); statErr == nil {
			// .gz ya existe en disco -> servir directamente, 0 CPU de compresion.
			req.Header.Set(gzHeaderOrigPath, origPath)
			req.Header.Set(gzHeaderPrecomp, "1")
			req.URL.Path = origPath + ".gz"
			if req.URL.RawPath != "" {
				req.URL.RawPath = req.URL.RawPath + ".gz"
			}
		} else {
			// .gz no existe -> comprimir on-the-fly y guardar en disco para proximos requests.
			req.Header.Set(gzHeaderAccept, "1")
			req.Header.Set(gzHeaderCachePath, diskPath)
		}
	}
	proxy.ModifyResponse = rp.proxyModifyResponse
	proxy.ErrorHandler = rp.proxyErrorHandler
	return proxy
}

func (rp *DynamicReverseProxyDiscovery) proxyModifyResponse(r *http.Response) error {
	origin := r.Request.Header.Get("Origin")
	if origin != "" {
		r.Header.Set("Access-Control-Allow-Origin", origin)
		r.Header.Set("Access-Control-Allow-Credentials", "true")
	}

	// Caso 1: .gz existia en disco — PocketBase lo sirvio, solo corregir headers.
	// Guard SPA: si PocketBase hizo fallback a index.html (text/html) para un .gz
	// inexistente, no aplicar gzip headers para evitar ERR_CONTENT_DECODING_FAILED.
	if r.Request.Header.Get(gzHeaderPrecomp) == "1" && r.StatusCode == http.StatusOK {
		origPath := r.Request.Header.Get(gzHeaderOrigPath)
		ext := path.Ext(origPath)
		mimeType := mime.TypeByExtension(ext)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		respCT := r.Header.Get("Content-Type")
		isSpaFallback := strings.HasPrefix(respCT, "text/html") && !strings.HasPrefix(mimeType, "text/html")
		if !isSpaFallback {
			r.Header.Set("Content-Encoding", "gzip")
			r.Header.Set("Content-Type", mimeType)
			r.Header.Set("Vary", "Accept-Encoding")
		}
	}

	// Caso 2: .gz no existia — comprimir on-the-fly (nivel 6, rapido) y cachear a disco.
	// io.MultiWriter hace tee: los bytes comprimidos van al browser Y al archivo .gz simultaneamente.
	// Proximos requests encontraran el .gz en disco y usaran el Caso 1 (0 CPU).
	if r.Request.Header.Get(gzHeaderAccept) == "1" && r.StatusCode == http.StatusOK {
		if isCompressibleMime(r.Header.Get("Content-Type")) {
			diskPath := r.Request.Header.Get(gzHeaderCachePath)
			if mkErr := os.MkdirAll(filepath.Dir(diskPath), 0755); mkErr == nil {
				if gzFile, createErr := os.Create(diskPath); createErr == nil {
					pr, pw := io.Pipe()
					// BestCompression = nivel 9: maximo ratio posible.
					// Justificado porque solo se comprime UNA VEZ (primer request) y se cachea a disco.
					// El costo extra de CPU es ~5ms por archivo — insignificante frente al beneficio permanente.
					// MultiWriter en la SALIDA del gzip: bytes comprimidos van a pw (browser) y gzFile (disco).
					gz, _ := gzip.NewWriterLevel(io.MultiWriter(pw, gzFile), gzip.BestCompression)
					origBody := r.Body
					go func() {
						if _, copyErr := io.Copy(gz, origBody); copyErr != nil {
							slog.Warn("proxy: error comprimiendo respuesta on-the-fly", "error", copyErr)
						}
						gz.Close()    // escribe footer gzip — obligatorio
						gzFile.Close()
						pw.Close()
						origBody.Close()
					}()
					r.Body = pr
					r.Header.Set("Content-Encoding", "gzip")
					r.Header.Set("Vary", "Accept-Encoding")
					r.Header.Del("Content-Length") // tamano varia al comprimir
					r.ContentLength = -1
				}
			}
		}
	}

	if r.Request.Method == http.MethodPost &&
		strings.HasPrefix(r.Request.URL.Path, superusersEndpoint) &&
		r.StatusCode == 200 {
		authorization := r.Request.Header.Get("Authorization")
		rp.installTokenUsecase.CleanInstallToken(r.Request.Context(), authorization)
	}
	return nil
}

// ResolveTarget determina el destino al que hay que redirigir la petición
// para el host dado. Retorna un *httputil.ReverseProxy o un error si no
// se pudo resolver el destino.
//
// Nota: la interfaz retorna *httputil.ReverseProxy por compatibilidad con el
// caller existente. Para el modo serve_static devolvemos nil y el caller
// detecta ese caso usando resolveHandler, que acepta http.Handler genérico.
func (rp *DynamicReverseProxyDiscovery) ResolveTarget(ctx context.Context, host string) (*httputil.ReverseProxy, error) {
	if host == rp.apiDomain {
		return rp.buildReverseProxy(&url.URL{
			Scheme: "http",
			Host:   rp.internalApiAddress,
		}, ""), nil
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

			// Modo estático: servir pb_public directamente desde disco, sin despertar PocketBase.
			if target.ServeStatic {
				return rp.resolveStaticHandler(serviceID)
			}

			service, err := rp.serviceDiscovery.FindRunningServiceByID(ctx, serviceID)

			// Si no hay error en BD y el servicio realmente corre en memoria
			if err == nil && rp.launcherManager.IsServiceRunning(serviceID) {
				rp.launcherManager.RecordActivity(serviceID)
				return rp.buildReverseProxy(&url.URL{
					Scheme: "http",
					Host:   net.JoinHostPort(service.IP, strconv.Itoa(service.Port)),
				}, serviceID), nil
			}

			// Si no corre en memoria, o no esta marcado en ejecucion en BD, lo despertamos
			if errors.Is(err, repositories.ErrNotFound) || (err == nil && !rp.launcherManager.IsServiceRunning(serviceID)) {
				ip, port, wakeupErr := rp.launcherManager.WakeupService(ctx, serviceID)
				if wakeupErr == nil {
					rp.launcherManager.RecordActivity(serviceID)
					_ = rp.serviceDiscovery.InvalidateServiceCacheByID(serviceID)
					return rp.buildReverseProxy(&url.URL{
						Scheme: "http",
						Host:   net.JoinHostPort(ip, strconv.Itoa(port)),
					}, serviceID), nil
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
			return rp.buildReverseProxy(targetURL, ""), nil
		}
		return nil, fmt.Errorf("no target found for domain: %s", host)
	}

	service, err := rp.serviceDiscovery.FindRunningServiceByID(ctx, id)
	if err == nil && rp.launcherManager.IsServiceRunning(id) {
		rp.launcherManager.RecordActivity(id)
		return rp.buildReverseProxy(&url.URL{
			Scheme: "http",
			Host:   net.JoinHostPort(service.IP, strconv.Itoa(service.Port)),
		}, id), nil
	}

	// Si no corre en memoria, o no esta marcado en ejecucion en BD
	if errors.Is(err, repositories.ErrNotFound) || (err == nil && !rp.launcherManager.IsServiceRunning(id)) {
		ip, port, wakeupErr := rp.launcherManager.WakeupService(ctx, id)
		if wakeupErr == nil {
			rp.launcherManager.RecordActivity(id)
			_ = rp.serviceDiscovery.InvalidateServiceCacheByID(id)
			return rp.buildReverseProxy(&url.URL{
				Scheme: "http",
				Host:   net.JoinHostPort(ip, strconv.Itoa(port)),
			}, id), nil
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
		return rp.buildReverseProxy(targetURL, ""), nil
	}
	if !errors.Is(err, repositories.ErrNotFound) {
		return nil, fmt.Errorf("failed to resolve proxy entry by id: %s", id)
	}
	return nil, fmt.Errorf("no target found for host: %s with id: %s", host, id)
}

// resolveStaticHandler construye un *httputil.ReverseProxy que sirve los
// archivos estáticos de pb_public directamente desde disco, sin despertar
// la instancia de PocketBase.
func (rp *DynamicReverseProxyDiscovery) resolveStaticHandler(serviceID string) (*httputil.ReverseProxy, error) {
	staticDir := filepath.Join(rp.dataDir, serviceID, "pb_public")
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		slog.Warn("serve_static: pb_public directory does not exist", "serviceID", serviceID, "path", staticDir)
		return nil, fmt.Errorf("static directory not found for service %s", serviceID)
	}
	return buildStaticProxy(newSpaFileServer(staticDir)), nil
}

// buildStaticProxy crea un httputil.ReverseProxy que sirve archivos estáticos
// directamente desde un http.Handler (FileServer) usando un Transport falso
// que intercepta las peticiones y las inyecta en el handler de forma in-process.
func buildStaticProxy(handler http.Handler) *httputil.ReverseProxy {
	dummyTarget, _ := url.Parse("http://localhost")
	p := httputil.NewSingleHostReverseProxy(dummyTarget)
	p.Transport = &staticTransport{handler: handler}
	p.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "static file error: "+err.Error(), http.StatusInternalServerError)
	}
	return p
}

// staticTransport implementa http.RoundTripper redirigiendo la petición al FileServer
// en memoria, sin realizar ninguna conexión de red real.
type staticTransport struct {
	handler http.Handler
}

func (t *staticTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rw := newResponseRecorder()
	t.handler.ServeHTTP(rw, req)
	return rw.toResponse(req), nil
}
