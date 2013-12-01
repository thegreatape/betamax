package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Config struct {
	Cassette string `json:"cassette"`
}

func handleConfigRequest(resp http.ResponseWriter, req *http.Request, config *Config) {
	if req.Method == "GET" {
		json.NewEncoder(resp).Encode(config)
	} else if req.Method == "POST" {
		json.NewDecoder(req.Body).Decode(config)
	}
}

func handleConfig(handler http.Handler) http.Handler {
	config := new(Config)
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/__betamax__/config" {
			handleConfigRequest(resp, req, config)
		} else {
			handler.ServeHTTP(resp, req)
		}
	})
}

func Proxy(target *url.URL) http.Handler {
	return handleConfig(httputil.NewSingleHostReverseProxy(target))
}
