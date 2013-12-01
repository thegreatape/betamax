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
)

var _ = Describe("Proxy", func() {
	var proxy http.Handler
	var targetServer *httptest.Server

	var proxyListener net.Listener
	var proxyPort string

	BeforeEach(func() {
		proxyListener, _ = net.Listen("tcp", "0.0.0.0:0")
		_, proxyPort, _ = net.SplitHostPort(proxyListener.Addr().String())

		targetServer = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			io.WriteString(writer, "hello, world")
		}))

		targetUrl, _ := url.Parse(targetServer.URL)
		proxy = Proxy(targetUrl)
		go http.Serve(proxyListener, proxy)
	})

	AfterEach(func() {
		targetServer.Close()
		proxyListener.Close()
	})

	It("proxies without any configuration", func() {
		resp, _ := http.Get(fmt.Sprintf("http://127.0.0.1:%s", proxyPort))
		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)
		Expect(string(body)).To(Equal("hello, world"))
	})

	It("returns a 500 if the target server is down", func() {
		targetServer.Close()
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s", proxyPort))
		Expect(err).To(BeNil())
		Expect(resp.StatusCode).To(Equal(500))
	})

	Context("allows configuration over http", func() {
		It("returns the current configuration with a GET to /__betamax__/config", func() {
			resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/__betamax__/config", proxyPort))
			Expect(err).To(BeNil())

			var jsonResponse map[string]interface{}
			body, _ := ioutil.ReadAll(resp.Body)
			err = json.Unmarshal(body, &jsonResponse)
			Expect(err).To(BeNil())

			Expect(jsonResponse["cassette"]).To(Equal(""))
		})

		It("allows setting the current cassette via POST", func() {
			resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%s/__betamax__/config", proxyPort), "text/json", bytes.NewBufferString("{\"cassette\": \"test-cassette\"}"))
			Expect(err).To(BeNil())

			resp, err = http.Get(fmt.Sprintf("http://127.0.0.1:%s/__betamax__/config", proxyPort))
			Expect(err).To(BeNil())

			var jsonResponse map[string]interface{}
			body, _ := ioutil.ReadAll(resp.Body)
			err = json.Unmarshal(body, &jsonResponse)
			Expect(err).To(BeNil())

			Expect(jsonResponse["cassette"]).To(Equal("test-cassette"))
		})
	})

	Context("records and plays back proxied responses", func() {
		It("replays responses when a cassette is set", func() {
			resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%s/__betamax__/config", proxyPort), "text/json", bytes.NewBufferString("{\"cassette\": \"test-cassette\"}"))
			Expect(err).To(BeNil())

			resp, err = http.Get(fmt.Sprintf("http://127.0.0.1:%s", proxyPort))
			body, _ := ioutil.ReadAll(resp.Body)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(200))
			Expect(string(body)).To(Equal("hello, world"))

			targetServer.Close()

			resp, err = http.Get(fmt.Sprintf("http://127.0.0.1:%s", proxyPort))
			body, _ = ioutil.ReadAll(resp.Body)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(200))
			Expect(string(body)).To(Equal("hello, world"))

		})
		PIt("records nothing without a current cassette", func() {})
		PIt("denies unrecorded responses when the option is set", func() {})
	})

})
