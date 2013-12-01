package proxy_test

import (
	"encoding/json"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/thegreatape/betamax/proxy"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
)

var _ = Describe("Proxy", func() {
	var proxy http.Handler
	var targetListener net.Listener
	var targetPort string
	var proxyListener net.Listener
	var proxyPort string

	BeforeEach(func() {
		targetListener, _ = net.Listen("tcp", "0.0.0.0:0")
		_, targetPort, _ = net.SplitHostPort(targetListener.Addr().String())
		proxyListener, _ = net.Listen("tcp", "0.0.0.0:0")
		_, proxyPort, _ = net.SplitHostPort(proxyListener.Addr().String())

		go http.Serve(targetListener, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			io.WriteString(writer, "hello, world")
		}))

		targetUrl, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%s/", targetPort))
		proxy = Proxy(targetUrl)
		go http.Serve(proxyListener, proxy)
	})

	AfterEach(func() {
		targetListener.Close()
		proxyListener.Close()
	})

	It("proxies without any configuration", func() {
		resp, _ := http.Get(fmt.Sprintf("http://127.0.0.1:%s", proxyPort))
		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)
		Expect(string(body)).To(Equal("hello, world"))
	})

	It("returns a 500 if the target server is down", func() {
		targetListener.Close()
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s", proxyPort))
		Expect(err).To(BeNil())
		Expect(resp.StatusCode).To(Equal(500))
	})

	Context("allows configuration over http", func() {
		It("returns the current configuration with a GET to /__betamax__/config", func() {
			resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/__betamax__/config", proxyPort))
			Expect(err).To(BeNil())

			var jsonResponse interface{}
			body, _ := ioutil.ReadAll(resp.Body)
			err = json.Unmarshal(body, &jsonResponse)
			Expect(err).To(BeNil())

			jsonData := jsonResponse.(map[string]interface{})
			Expect(jsonData["cassette"]).To(BeNil())
		})
	})
})
