package proxy

import (
	"net/http/httputil"
	"net/url"
)

func Proxy(target *url.URL) *httputil.ReverseProxy {
	return httputil.NewSingleHostReverseProxy(target)
}
