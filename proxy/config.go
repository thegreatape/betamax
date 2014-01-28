package proxy

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
)

type Config struct {
	TargetHost             string
	CassetteDir            string
	Episodes               []Episode
	Cassette               string   `json:"cassette"`
	RecordNewEpisodes      bool     `json:"record_new_episodes"`
	DenyUnrecordedRequests bool     `json:"deny_unrecorded_requests"`
	RewriteHostHeader      bool     `json:"rewrite_host_header"`
	MatchHeaders           []string `json:"match_headers"`
}

type WriteableEpisode struct {
	Request  WriteableRecordedRequest
	Response WriteableRecordedResponse
}

// proxy structs with interface{} instead of []byte
// for bodies so we can write plain text as human-readable
// strings but still store binary
type WriteableRecordedRequest struct {
	Method string
	URL    *url.URL
	Header http.Header
	Body   interface{}
	Form   map[string][]string
}

type WriteableRecordedResponse struct {
	StatusCode int
	Body       interface{}
	Header     http.Header
}

func IsText(headers http.Header) bool {
	contentType := headers["Content-Type"]
	if contentType == nil {
		return false
	}
	matched, _ := regexp.Match("^(text/)|(json)", []byte(contentType[0]))
	return matched
}

func writableBodyForContentType(body []byte, headers http.Header) interface{} {
	if IsText(headers) {
		return string(body)
	} else {
		return body
	}
}

func bodyForContentType(body interface{}, headers http.Header) []byte {
	if IsText(headers) {
		return []byte(body.(string))
	} else {
		str, _ := base64.StdEncoding.DecodeString(body.(string))
		return []byte(str)
	}
}

func writeableEpisodes(episodes []Episode) []WriteableEpisode {
	writeables := make([]WriteableEpisode, len(episodes))
	for i, episode := range episodes {
		request := WriteableRecordedRequest{
			Method: episode.Request.Method,
			URL:    episode.Request.URL,
			Header: episode.Request.Header,
			Body:   writableBodyForContentType(episode.Request.Body, episode.Request.Header),
			Form:   episode.Request.Form,
		}

		response := WriteableRecordedResponse{
			StatusCode: episode.Response.StatusCode,
			Header:     episode.Response.Header,
			Body:       writableBodyForContentType(episode.Response.Body, episode.Response.Header),
		}

		writeable := WriteableEpisode{
			Request:  request,
			Response: response,
		}

		writeables[i] = writeable
	}
	return writeables
}

func episodes(writeableEpisodes []WriteableEpisode) []Episode {
	episodes := make([]Episode, len(writeableEpisodes))
	for i, writeableEpisode := range writeableEpisodes {
		request := RecordedRequest{
			Method: writeableEpisode.Request.Method,
			URL:    writeableEpisode.Request.URL,
			Header: writeableEpisode.Request.Header,
			Body:   bodyForContentType(writeableEpisode.Request.Body, writeableEpisode.Request.Header),
			Form:   writeableEpisode.Request.Form,
		}

		response := RecordedResponse{
			StatusCode: writeableEpisode.Response.StatusCode,
			Header:     writeableEpisode.Response.Header,
			Body:       bodyForContentType(writeableEpisode.Response.Body, writeableEpisode.Response.Header),
		}

		episode := Episode{
			Request:  request,
			Response: response,
		}

		episodes[i] = episode
	}
	return episodes
}

func (c *Config) Save() error {
	episodes := writeableEpisodes(c.Episodes)

	jsonData, err := json.MarshalIndent(&episodes, "", "  ")
	if err != nil {
		return err
	}
	os.MkdirAll(c.CassetteDir, 0700)
	return ioutil.WriteFile(path.Join(c.CassetteDir, c.Cassette+".json"), jsonData, 0700)
}

func (c *Config) Load() error {

	cassetteData, err := ioutil.ReadFile(path.Join(c.CassetteDir, c.Cassette+".json"))
	if err != nil {
		c.Episodes = []Episode{}
		return err
	}
	writableEpisodes := []WriteableEpisode{}
	err = json.Unmarshal(cassetteData, &writableEpisodes)
	c.Episodes = episodes(writableEpisodes)
	return err
}
