package proxy

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
)

type Config struct {
	CassetteDir string
	Cassette    string `json:"cassette"`
	Episodes    []Episode
}

func (c *Config) Save() error {
	jsonData, err := json.Marshal(&c.Episodes)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path.Join(c.CassetteDir, c.Cassette+".json"), jsonData, 0700)
}

type Cassette struct {
	Name     string
	Episodes []Episode
}

type Episode struct {
	Request  http.Request
	Response RecordedResponse
}

type RecordedResponse struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

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
	p.Response.Headers = p.Writer.Header()
	p.Response.StatusCode = statusCode
	p.Writer.WriteHeader(statusCode)
}

func handleConfigRequest(resp http.ResponseWriter, req *http.Request, config *Config) {
	if req.Method == "GET" {
		json.NewEncoder(resp).Encode(config)
	} else if req.Method == "POST" {
		json.NewDecoder(req.Body).Decode(config)
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

func sameURL(a *url.URL, b *url.URL) bool {
	return a.Path == b.Path && a.RawQuery == b.RawQuery && a.Fragment == b.Fragment
}

func sameRequest(a *http.Request, b *http.Request) bool {
	if a.Method != b.Method {
		return false
	}

	if !sameURL(a.URL, b.URL) {
		return false
	}

	//if a.Body != b.Body {
	//return false
	//}

	return true
}

func serveAndRecord(resp http.ResponseWriter, req *http.Request, handler http.Handler, config *Config) {
	proxyWriter := ProxyResponseWriter{Writer: resp}
	handler.ServeHTTP(&proxyWriter, req)
	writeEpisode(Episode{Request: *req, Response: proxyWriter.Response}, config)
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

	for k, values := range episode.Response.Headers {
		for _, value := range values {
			resp.Header().Add(k, value)
		}
	}
	resp.WriteHeader(episode.Response.StatusCode)
	resp.Write(episode.Response.Body)
}

func Proxy(target *url.URL, cassetteDir string) http.Handler {
	config := &Config{CassetteDir: cassetteDir}
	os.MkdirAll(cassetteDir, 0700)

	configHandler := configHandler(httputil.NewSingleHostReverseProxy(target), config)
	return cassetteHandler(configHandler, config)
}
