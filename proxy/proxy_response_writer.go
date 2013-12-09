package proxy

import "net/http"

type ProxyResponseWriter struct {
	Writer   http.ResponseWriter
	Response RecordedResponse
}

func (p *ProxyResponseWriter) Header() http.Header {
	return p.Writer.Header()
}

func (p *ProxyResponseWriter) Write(bytes []byte) (int, error) {
	p.Response.Body = append(p.Response.Body, bytes...)
	return p.Writer.Write(bytes)
}

func (p *ProxyResponseWriter) WriteHeader(statusCode int) {
	// according to docs, once WriteHeader is called, further modifications to Header have
	// no effect; hence, we can copy it here.
	p.Response.Header = p.Writer.Header()
	p.Response.StatusCode = statusCode
	p.Writer.WriteHeader(statusCode)
}
