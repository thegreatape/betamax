package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func handleConfig(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if req.Method == "GET" && req.URL.Path == "/__betamax__/config" {
			fmt.Fprintf(resp, "{\"cassette\": null}")
		} else {
			handler.ServeHTTP(resp, req)
		}
	})
}

func Proxy(target *url.URL) http.Handler {
	return handleConfig(httputil.NewSingleHostReverseProxy(target))
}
