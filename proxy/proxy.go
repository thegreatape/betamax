package proxy

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func handleConfigRequest(resp http.ResponseWriter, req *http.Request, config *Config) {
	if req.Method == "GET" {
		json.NewEncoder(resp).Encode(config)
	} else if req.Method == "POST" {
		json.NewDecoder(req.Body).Decode(config)
		config.Load()
	}
}

func configHandler(handler http.Handler, config *Config) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/__betamax__/config" {
			handleConfigRequest(resp, req, config)
		} else {
			handler.ServeHTTP(resp, req)
		}
	})
}

func cassetteHandler(handler http.Handler, config *Config) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if config.Cassette == "" {
			handler.ServeHTTP(resp, req)
			return
		}

		if episode := findEpisode(req, config); episode != nil {
			serveEpisode(episode, resp)
		} else {
			serveAndRecord(resp, req, handler, config)
		}
	})
}

// reads all bytes of the request into memory
// returns the read bytes and replaces the request's Reader
// with a refilled reader. seems like there should be a better
// way to do this.
func peekBytes(req *http.Request) (body []byte, err error) {
	body, err = ioutil.ReadAll(req.Body)
	req.Body = ioutil.NopCloser(bytes.NewReader(body))
	return
}

func sameURL(a *url.URL, b *url.URL) bool {
	return a.Path == b.Path && a.RawQuery == b.RawQuery && a.Fragment == b.Fragment
}

func sameRequest(a *RecordedRequest, b *http.Request) bool {
	if a.Method != b.Method {
		return false
	}

	if !sameURL(a.URL, b.URL) {
		return false
	}

	body, _ := peekBytes(b)
	if bytes.Compare(a.Body, body) != 0 {
		return false
	}

	return true
}

func serveAndRecord(resp http.ResponseWriter, req *http.Request, handler http.Handler, config *Config) {
	proxyWriter := ProxyResponseWriter{Writer: resp}
	recordedRequest := recordRequest(req)

	handler.ServeHTTP(&proxyWriter, req)
	writeEpisode(Episode{Request: recordedRequest, Response: proxyWriter.Response}, config)
}

func recordRequest(req *http.Request) RecordedRequest {
	body, _ := peekBytes(req)
	return RecordedRequest{
		URL:    req.URL,
		Header: req.Header,
		Method: req.Method,
		Body:   body,
	}
}

func writeEpisode(episode Episode, config *Config) {
	config.Episodes = append(config.Episodes, episode)
	config.Save()
}

func findEpisode(req *http.Request, config *Config) *Episode {
	for _, episode := range config.Episodes {
		if sameRequest(&episode.Request, req) {
			return &episode
		}
	}
	return nil
}

func serveEpisode(episode *Episode, resp http.ResponseWriter) {
	for k, values := range episode.Response.Header {
		for _, value := range values {
			resp.Header().Add(k, value)
		}
	}
	resp.WriteHeader(episode.Response.StatusCode)
	resp.Write(episode.Response.Body)
}

func Proxy(target *url.URL, cassetteDir string) http.Handler {
	config := &Config{CassetteDir: cassetteDir}
	cassetteHandler := cassetteHandler(httputil.NewSingleHostReverseProxy(target), config)
	return configHandler(cassetteHandler, config)
}
