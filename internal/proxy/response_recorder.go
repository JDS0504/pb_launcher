package proxy

import (
	"bytes"
	"io"
	"net/http"
)

// responseRecorder captura la respuesta de un http.Handler en memoria
// para luego convertirla en un *http.Response compatible con http.RoundTripper.
type responseRecorder struct {
	header http.Header
	body   *bytes.Buffer
	code   int
}

func newResponseRecorder() *responseRecorder {
	return &responseRecorder{
		header: make(http.Header),
		body:   &bytes.Buffer{},
		code:   http.StatusOK,
	}
}

func (r *responseRecorder) Header() http.Header {
	return r.header
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.code = statusCode
}

func (r *responseRecorder) toResponse(req *http.Request) *http.Response {
	return &http.Response{
		StatusCode: r.code,
		Header:     r.header,
		Body:       io.NopCloser(bytes.NewReader(r.body.Bytes())),
		Request:    req,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}
}
