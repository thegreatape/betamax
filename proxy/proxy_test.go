package proxy_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/thegreatape/betamax/proxy"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
)

var _ = Describe("Proxy", func() {
	var proxy http.Handler
	var targetServer *httptest.Server

	var proxyListener net.Listener
	var proxyPort string
	var targetUrl *url.URL
	var cassetteDir string
	var requestCount int

	proxyGetWithHeaders := func(path string, headers map[string]string) (*http.Response, error) {
		client := new(http.Client)
		req, _ := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%s%s", proxyPort, path), nil)
		for key, value := range headers {
			req.Header.Add(key, value)
		}

		return client.Do(req)
	}

	proxyGet := func(path string) (*http.Response, error) {
		return http.Get(fmt.Sprintf("http://127.0.0.1:%s%s", proxyPort, path))
	}

	proxyPost := func(path string, form url.Values) (*http.Response, error) {
		return http.PostForm(fmt.Sprintf("http://127.0.0.1:%s%s", proxyPort, path), form)
	}

	configureProxy := func(options map[string]interface{}) {
		jsonBytes, err := json.Marshal(options)
		Expect(err).To(BeNil())

		_, err = http.Post(fmt.Sprintf("http://127.0.0.1:%s/__betamax__/config", proxyPort),
			"text/json",
			bytes.NewBuffer(jsonBytes))
		Expect(err).To(BeNil())
	}

	BeforeEach(func() {
		requestCount = 0
		proxyListener, _ = net.Listen("tcp", "0.0.0.0:0")
		_, proxyPort, _ = net.SplitHostPort(proxyListener.Addr().String())

		targetServer = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			requestCount++
			if request.URL.Path == "/request-count" {
				io.WriteString(writer, fmt.Sprintf("%d requests so far", requestCount))
			} else if request.URL.Path == "/echo-host" {
				io.WriteString(writer, request.Host)
			} else {
				io.WriteString(writer, "hello, world")
			}
		}))

		targetUrl, _ = url.Parse(targetServer.URL)
		cassetteDir = path.Join(os.TempDir(), "cassettes")
		os.RemoveAll(cassetteDir)
		proxy = Proxy(targetUrl, cassetteDir)
		go http.Serve(proxyListener, proxy)
	})

	AfterEach(func() {
		targetServer.Close()
		proxyListener.Close()
	})

	It("proxies without any configuration", func() {
		resp, _ := proxyGet("/")
		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)
		Expect(string(body)).To(Equal("hello, world"))
	})

	It("returns a 500 if the target server is down", func() {
		targetServer.Close()
		resp, err := proxyGet("/")
		Expect(err).To(BeNil())
		Expect(resp.StatusCode).To(Equal(500))
	})

	Context("allows configuration over http", func() {
		It("returns the current configuration with a GET to /__betamax__/config", func() {
			resp, err := proxyGet("/__betamax__/config")
			Expect(err).To(BeNil())

			var jsonResponse map[string]interface{}
			body, _ := ioutil.ReadAll(resp.Body)
			err = json.Unmarshal(body, &jsonResponse)
			Expect(err).To(BeNil())

			Expect(jsonResponse["cassette"]).To(Equal(""))
		})

		It("allows setting the current cassette via POST", func() {
			configureProxy(map[string]interface{}{"cassette": "test-cassette"})

			resp, err := proxyGet("/__betamax__/config")
			Expect(err).To(BeNil())

			var jsonResponse map[string]interface{}
			body, _ := ioutil.ReadAll(resp.Body)
			err = json.Unmarshal(body, &jsonResponse)
			Expect(err).To(BeNil())

			Expect(jsonResponse["cassette"]).To(Equal("test-cassette"))
		})
	})

	Context("records and plays back proxied responses", func() {
		It("replays requests when a cassette is set", func() {
			configureProxy(map[string]interface{}{"cassette": "test-cassette"})

			resp, err := proxyGet("/")
			body, _ := ioutil.ReadAll(resp.Body)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(200))
			Expect(string(body)).To(Equal("hello, world"))

			targetServer.Close()

			resp, err = proxyGet("/")
			body, _ = ioutil.ReadAll(resp.Body)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(200))
			Expect(string(body)).To(Equal("hello, world"))
		})

		It("differentiates requests with different bodies", func() {
			configureProxy(map[string]interface{}{"cassette": "test-cassette"})

			formOne := url.Values{}
			formOne.Add("Foo", "Bar")

			formTwo := url.Values{}
			formTwo.Add("Foo", "Bar")
			formTwo.Add("Baz", "Quux")

			resp, _ := proxyPost("/request-count", formOne)
			body, _ := ioutil.ReadAll(resp.Body)
			Expect(string(body)).To(Equal("1 requests so far"))

			resp, _ = proxyPost("/request-count", formTwo)
			body, _ = ioutil.ReadAll(resp.Body)
			Expect(string(body)).To(Equal("2 requests so far"))

			resp, _ = proxyPost("/request-count", formOne)
			body, _ = ioutil.ReadAll(resp.Body)
			Expect(string(body)).To(Equal("1 requests so far"))
		})

		It("differentiates requests with different methods", func() {
			configureProxy(map[string]interface{}{"cassette": "test-cassette"})

			resp, _ := proxyPost("/request-count", url.Values{})
			body, _ := ioutil.ReadAll(resp.Body)
			Expect(string(body)).To(Equal("1 requests so far"))

			resp, _ = proxyGet("/request-count")
			body, _ = ioutil.ReadAll(resp.Body)
			Expect(string(body)).To(Equal("2 requests so far"))

			resp, _ = proxyPost("/request-count", url.Values{})
			body, _ = ioutil.ReadAll(resp.Body)
			Expect(string(body)).To(Equal("1 requests so far"))
		})

		It("differentiates requests with different headers when configured to match them", func() {
			configureProxy(map[string]interface{}{"cassette": "test-cassette"})

			resp, _ := proxyGetWithHeaders("/request-count", map[string]string{"Content-Type": "text/json"})
			body, _ := ioutil.ReadAll(resp.Body)
			Expect(string(body)).To(Equal("1 requests so far"))

			resp, _ = proxyGetWithHeaders("/request-count", map[string]string{"Content-Type": "text/html"})
			body, _ = ioutil.ReadAll(resp.Body)
			Expect(string(body)).To(Equal("1 requests so far"))

			configureProxy(map[string]interface{}{"match_headers": []string{"Content-Type"}})

			resp, _ = proxyGetWithHeaders("/request-count", map[string]string{"Content-Type": "text/sharks"})
			body, _ = ioutil.ReadAll(resp.Body)
			Expect(string(body)).To(Equal("2 requests so far"))
		})

		It("differentiates requests with different query string", func() {
			configureProxy(map[string]interface{}{"cassette": "test-cassette"})

			resp, _ := proxyGet("/request-count?foo=bar")
			body, _ := ioutil.ReadAll(resp.Body)
			Expect(string(body)).To(Equal("1 requests so far"))

			resp, _ = proxyGet("/request-count?foo=quux")
			body, _ = ioutil.ReadAll(resp.Body)
			Expect(string(body)).To(Equal("2 requests so far"))

			resp, _ = proxyGet("/request-count?foo=bar")
			body, _ = ioutil.ReadAll(resp.Body)
			Expect(string(body)).To(Equal("1 requests so far"))
		})

		It("records nothing without a current cassette", func() {
			resp, err := proxyGet("/")
			body, _ := ioutil.ReadAll(resp.Body)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(200))
			Expect(string(body)).To(Equal("hello, world"))

			targetServer.Close()

			resp, err = proxyGet("/")
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(500))
		})

		It("denies unrecorded responses when the option is set", func() {
			configureProxy(map[string]interface{}{"cassette": "test-cassette", "deny_unrecorded_requests": true})

			resp, _ := proxyGet("/request-count")
			Expect(resp.StatusCode).To(Equal(403))

			body, _ := ioutil.ReadAll(resp.Body)
			Expect(string(body)).To(Equal(""))
		})

		It("does not record new episodes when the option is unset", func() {
			configureProxy(map[string]interface{}{"cassette": "test-cassette", "record_new_episodes": false})

			resp, _ := proxyGet("/request-count")
			body, _ := ioutil.ReadAll(resp.Body)
			Expect(string(body)).To(Equal("1 requests so far"))

			resp, _ = proxyGet("/request-count")
			body, _ = ioutil.ReadAll(resp.Body)
			Expect(string(body)).To(Equal("2 requests so far"))
		})

		It("write cassettes to disk", func() {
			configureProxy(map[string]interface{}{"cassette": "test-cassette"})

			proxyGet("/")

			cassetteData, err := ioutil.ReadFile(path.Join(cassetteDir, "test-cassette.json"))
			Expect(err).To(BeNil())
			Expect(cassetteData).ToNot(BeEmpty())

			var cassetteJson []map[string]interface{}
			err = json.Unmarshal(cassetteData, &cassetteJson)
			Expect(err).To(BeNil())
			Expect(cassetteJson).ToNot(BeEmpty())

			episode := cassetteJson[0]
			Expect(episode["Request"]).ToNot(BeEmpty())
			Expect(episode["Response"]).ToNot(BeEmpty())
		})

		It("switches cassettes on demand", func() {
			configureProxy(map[string]interface{}{"cassette": "first-cassette"})

			resp, _ := proxyGet("/request-count")
			body, _ := ioutil.ReadAll(resp.Body)
			Expect(string(body)).To(Equal("1 requests so far"))

			configureProxy(map[string]interface{}{"cassette": "second-cassette"})

			resp, _ = proxyGet("/request-count")
			body, _ = ioutil.ReadAll(resp.Body)
			Expect(string(body)).To(Equal("2 requests so far"))

			configureProxy(map[string]interface{}{"cassette": "first-cassette"})

			resp, _ = proxyGet("/request-count")
			body, _ = ioutil.ReadAll(resp.Body)
			Expect(string(body)).To(Equal("1 requests so far"))
		})

		It("rewrites the Host header by default to the target's host", func() {
			resp, _ := proxyGet("/echo-host")
			body, _ := ioutil.ReadAll(resp.Body)
			Expect(string(body)).To(Equal(targetUrl.Host))

			configureProxy(map[string]interface{}{"rewrite_host_header": false})

			resp, _ = proxyGet("/echo-host")
			body, _ = ioutil.ReadAll(resp.Body)
			Expect(string(body)).To(Equal(fmt.Sprintf("127.0.0.1:%s", proxyPort)))
		})
	})

})
