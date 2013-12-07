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
	var cassetteDir string
	var requestCount int

	proxyGet := func(path string) (*http.Response, error) {
		return http.Get(fmt.Sprintf("http://127.0.0.1:%s%s", proxyPort, path))
	}

	setCassette := func(cassetteName string) {
		jsonString := fmt.Sprintf("{\"cassette\": \"%v\"}", cassetteName)
		_, err := http.Post(fmt.Sprintf("http://127.0.0.1:%s/__betamax__/config", proxyPort),
			"text/json",
			bytes.NewBufferString(jsonString))
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
			} else {
				io.WriteString(writer, "hello, world")
			}
		}))

		targetUrl, _ := url.Parse(targetServer.URL)
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
			setCassette("test-cassette")

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
		It("replays GETs when a cassette is set", func() {
			setCassette("test-cassette")

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

		PIt("replays POSTs when a cassette is set", func() {})
		PIt("differentiates requests with different bodies", func() {})
		PIt("differentiates requests with different methods", func() {})
		PIt("differentiates requests with different headers", func() {})

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

		PIt("denies unrecorded responses when the option is set", func() {})

		It("write cassettes to disk", func() {
			setCassette("test-cassette")

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
			setCassette("first-cassette")

			resp, _ := proxyGet("/request-count")
			body, _ := ioutil.ReadAll(resp.Body)
			Expect(string(body)).To(Equal("1 requests so far"))

			setCassette("second-cassette")

			resp, _ = proxyGet("/request-count")
			body, _ = ioutil.ReadAll(resp.Body)
			Expect(string(body)).To(Equal("2 requests so far"))

			setCassette("first-cassette")

			resp, _ = proxyGet("/request-count")
			body, _ = ioutil.ReadAll(resp.Body)
			Expect(string(body)).To(Equal("1 requests so far"))
		})
	})

})
