package proxy

import (
	"net/http"
	"net/url"
)

type Cassette struct {
	Name     string
	Episodes []Episode
}

type Episode struct {
	Request  RecordedRequest
	Response RecordedResponse
}

type RecordedRequest struct {
	Method string
	URL    *url.URL
	Header http.Header
	Body   []byte
	Form   map[string][]string
}

type RecordedResponse struct {
	StatusCode int
	Body       []byte
	Header     http.Header
}
