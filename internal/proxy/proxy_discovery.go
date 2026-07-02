package proxy

import (
	"bytes"
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
	domainDiscovery     *proxydomain.DomainServiceDiscovery
	installTokenUsecase *launcherdomain.CleanServiceInstallTokenUsecase
	launcherManager     *launcherdomain.LauncherManager
	apiDomain           string
	internalApiAddress  string
	dataDir             string
}

func NewDynamicReverseProxyDiscovery(
	serviceDiscovery *proxydomain.ServiceDiscovery,
	domainDiscovery *proxydomain.DomainServiceDiscovery,
	installTokenUsecase *launcherdomain.CleanServiceInstallTokenUsecase,
	launcherManager *launcherdomain.LauncherManager,
	cfg configs.Config,
	pbConf *apis.ServeConfig) *DynamicReverseProxyDiscovery {

	launcherManager.SetOnServiceDeactivated(func(serviceID string) {
		svc, err := serviceDiscovery.FindServiceByIDOrName(context.Background(), serviceID)
		if err == nil {
			_ = serviceDiscovery.InvalidateServiceCache(svc.ID, svc.Name)
		} else {
			_ = serviceDiscovery.InvalidateServiceCache(serviceID, "")
		}
	})

	return &DynamicReverseProxyDiscovery{
		serviceDiscovery:    serviceDiscovery,
		domainDiscovery:     domainDiscovery,
		installTokenUsecase: installTokenUsecase,
		launcherManager:     launcherManager,
		apiDomain:           cfg.GetDomain(),
		internalApiAddress:  strings.Replace(pbConf.HttpAddr, "0.0.0.0:", "127.0.0.1:", 1),
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
	slog.Error("Reverse proxy error", "error", err, "url", r.URL.String(), "host", r.Host)

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

// gzDiskPath calcula la ruta en disco del .gz para un dataDir, serviceID y URL path dados.
// Retorna ("", false) si el path no es elegible para compresion (API, admin, extension no comprimible).
// Es una funcion de paquete (sin receptor) para poder reutilizarse desde buildReverseProxy y staticTransport.
func gzDiskPath(dataDir, serviceName, urlPath string) (diskPath string, gzUrlPath string, ok bool) {
	if serviceName == "" {
		return "", "", false
	}
	if len(urlPath) > 200 {
		return "", "", false
	}
	if strings.HasPrefix(urlPath, "/api/") || strings.HasPrefix(urlPath, "/_/") {
		return "", "", false
	}

	cleanPath := filepath.FromSlash(path.Clean("/" + urlPath))
	originalFile := filepath.Join(dataDir, serviceName, "pb_public", cleanPath)

	fi, err := os.Stat(originalFile)
	resolvedUrlPath := urlPath
	// Si el archivo no existe o es un directorio, asumimos fallback a index.html (SPA)
	if os.IsNotExist(err) || (err == nil && fi.IsDir()) {
		originalFile = filepath.Join(dataDir, serviceName, "pb_public", "index.html")
		if _, err := os.Stat(originalFile); err != nil {
			return "", "", false
		}
		resolvedUrlPath = "/index.html"
	}

	ext := filepath.Ext(originalFile)
	if !gzipCompressibleExts[strings.ToLower(ext)] {
		return "", "", false
	}

	diskPath = originalFile + ".gz"
	gzUrlPath = resolvedUrlPath + ".gz"
	return diskPath, gzUrlPath, true
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
		diskPath, gzUrl, ok := gzDiskPath(rp.dataDir, serviceID, req.URL.Path)
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
			req.URL.Path = gzUrl
			if req.URL.RawPath != "" {
				req.URL.RawPath = url.PathEscape(gzUrl)
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
		reqPath := r.Request.URL.Path
		origFilePath := strings.TrimSuffix(reqPath, ".gz")
		ext := path.Ext(origFilePath)
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
func (rp *DynamicReverseProxyDiscovery) ResolveTarget(ctx context.Context, host string, urlPath string) (*httputil.ReverseProxy, error) {
	if host == rp.apiDomain {
		return rp.buildReverseProxy(&url.URL{
			Scheme: "http",
			Host:   rp.internalApiAddress,
		}, ""), nil
	}

	serviceID, err := rp.extractID(host)
	if err != nil {
		return nil, err
	}

	var serviceName string
	isCustomDomain := false

	// Si no es un subdominio, buscamos por dominio personalizado
	if serviceID == "" {
		isCustomDomain = true
		target, err := rp.domainDiscovery.FindTargetByDomain(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("no target found for domain: %s", host)
		}
		if target.Service == "" {
			return nil, fmt.Errorf("no service associated with domain: %s", host)
		}
		serviceID = target.Service
		serviceName = target.ServiceName
	} else {
		// serviceID es el subdominio, que ahora corresponde al name
		serviceName = serviceID
	}

	// Si es una ruta de API o administración, ruteamos a PocketBase (y despertamos si es necesario)
	if strings.HasPrefix(urlPath, "/api/") || strings.HasPrefix(urlPath, "/_/") {
		// 1. Obtener la metadata del servicio (usando cache en memoria de 15 segundos)
		service, err := rp.serviceDiscovery.FindServiceByIDOrName(ctx, serviceID)
		if err != nil {
			if errors.Is(err, repositories.ErrNotFound) {
				return nil, fmt.Errorf("service not found for target: %s", serviceID)
			}
			return nil, fmt.Errorf("failed to fetch service metadata: %w", err)
		}

		// 2. Si es un subdominio, validar estrictamente que el nombre coincida (bloquea el ID técnico en subdominios)
		if !isCustomDomain && service.Name != serviceName {
			return nil, fmt.Errorf("service not found for subdomain: %s", serviceName)
		}

		// 3. Si está marcado como activo en BD y corre en memoria, ruteamos directamente
		if service.Status == "running" && rp.launcherManager.IsServiceRunning(service.ID) {
			rp.launcherManager.RecordActivity(service.ID)
			return rp.buildReverseProxy(&url.URL{
				Scheme: "http",
				Host:   net.JoinHostPort(service.IP, strconv.Itoa(service.Port)),
			}, serviceName), nil
		}

		// 4. Si no corre en memoria (durmiendo/offline), despertamos usando su ID real de base de datos
		ip, port, wakeupErr := rp.launcherManager.WakeupService(ctx, service.ID)
		if wakeupErr == nil {
			rp.launcherManager.RecordActivity(service.ID)
			_ = rp.serviceDiscovery.InvalidateServiceCache(service.ID, service.Name)
			return rp.buildReverseProxy(&url.URL{
				Scheme: "http",
				Host:   net.JoinHostPort(ip, strconv.Itoa(port)),
			}, serviceName), nil
		}
		return nil, fmt.Errorf("failed to wake up service %s: %w", service.ID, wakeupErr)
	}

	// Para cualquier otra ruta (estáticos), servimos desde disco directamente usando el nombre
	return rp.resolveStaticHandler(serviceName)
}

// resolveStaticHandler construye un *httputil.ReverseProxy que sirve los
// archivos estáticos de pb_public directamente desde disco, sin despertar
// la instancia de PocketBase.
func (rp *DynamicReverseProxyDiscovery) resolveStaticHandler(serviceName string) (*httputil.ReverseProxy, error) {
	staticDir := filepath.Join(rp.dataDir, serviceName)
	staticDir = filepath.Join(staticDir, "pb_public")
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		slog.Warn("serve_static: pb_public directory does not exist", "serviceName", serviceName, "path", staticDir)
		return nil, fmt.Errorf("static directory not found for service %s", serviceName)
	}
	return buildStaticProxy(newSpaFileServer(staticDir), rp.dataDir, serviceName), nil
}

// buildStaticProxy crea un httputil.ReverseProxy que sirve archivos estáticos
// directamente desde un http.Handler (FileServer) usando un Transport falso
// que intercepta las peticiones y las inyecta en el handler de forma in-process.
func buildStaticProxy(handler http.Handler, dataDir, serviceName string) *httputil.ReverseProxy {
	p := httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "http", Host: "localhost"})
	p.Transport = &staticTransport{handler: handler, dataDir: dataDir, serviceName: serviceName}
	p.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "static file error: "+err.Error(), http.StatusInternalServerError)
	}
	return p
}

// staticTransport implementa http.RoundTripper redirigiendo la petición al FileServer
// en memoria, sin realizar ninguna conexión de red real.
// Aplica el mismo patrón lazy-compress + cache a disco que buildReverseProxy:
//   - Primer visitante: comprime on-the-fly (nivel 9) y guarda .gz a disco.
//   - Visitantes siguientes: sirve el .gz precomprimido (0 CPU).
type staticTransport struct {
	handler     http.Handler
	dataDir     string
	serviceName string
}

func (t *staticTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Solo aplicar compresión si el cliente acepta gzip y la ruta es elegible.
	acceptsGzip := strings.Contains(req.Header.Get("Accept-Encoding"), "gzip")
	if acceptsGzip {
		diskPath, gzUrl, ok := gzDiskPath(t.dataDir, t.serviceName, req.URL.Path)
		if ok {
			origFilePath := strings.TrimSuffix(gzUrl, ".gz")
			ext := path.Ext(origFilePath)

			if _, statErr := os.Stat(diskPath); statErr == nil {
				// Caso 1: .gz precomprimido existe en disco → servir directamente (0 CPU).
				gzData, readErr := os.ReadFile(diskPath)
				if readErr == nil {
					mimeType := mime.TypeByExtension(ext)
					if mimeType == "" {
						mimeType = "application/octet-stream"
					}
					h := make(http.Header)
					h.Set("Content-Type", mimeType)
					h.Set("Content-Encoding", "gzip")
					h.Set("Vary", "Accept-Encoding")
					return &http.Response{
						StatusCode:    http.StatusOK,
						Header:        h,
						Body:          io.NopCloser(bytes.NewReader(gzData)),
						ContentLength: int64(len(gzData)),
						Request:       req,
						Proto:         "HTTP/1.1",
						ProtoMajor:    1,
						ProtoMinor:    1,
					}, nil
				}
			}

			// Caso 2: .gz no existe → servir el archivo original, comprimir on-the-fly
			// y guardar .gz a disco para los próximos requests.
			rw := newResponseRecorder()
			t.handler.ServeHTTP(rw, req)
			resp := rw.toResponse(req)

			if resp.StatusCode == http.StatusOK && isCompressibleMime(resp.Header.Get("Content-Type")) {
				if mkErr := os.MkdirAll(filepath.Dir(diskPath), 0755); mkErr == nil {
					if gzFile, createErr := os.Create(diskPath); createErr == nil {
						pr, pw := io.Pipe()
						// BestCompression = nivel 9: se paga solo una vez, luego siempre se sirve el .gz.
						// MultiWriter: bytes comprimidos van a pw (browser) y gzFile (disco) simultáneamente.
						gz, _ := gzip.NewWriterLevel(io.MultiWriter(pw, gzFile), gzip.BestCompression)
						origBody := resp.Body
						go func() {
							if _, copyErr := io.Copy(gz, origBody); copyErr != nil {
								slog.Warn("static: error comprimiendo on-the-fly", "error", copyErr, "path", req.URL.Path)
							}
							gz.Close()     // escribe footer gzip — obligatorio
							gzFile.Close()
							pw.Close()
							origBody.Close()
						}()
						resp.Body = pr
						resp.Header.Set("Content-Encoding", "gzip")
						resp.Header.Set("Vary", "Accept-Encoding")
						resp.Header.Del("Content-Length")
						resp.ContentLength = -1
					}
				}
			}
			return resp, nil
		}
	}

	// Sin compresión: proxy transparente (rutas no elegibles, cliente sin gzip, etc.).
	rw := newResponseRecorder()
	t.handler.ServeHTTP(rw, req)
	return rw.toResponse(req), nil
}
