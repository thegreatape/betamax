package proxy_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/thegreatape/betamax/proxy"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
)

var _ = Describe("Proxy", func() {
	var proxy *httputil.ReverseProxy
	var targetListener net.Listener
	var proxyListener net.Listener

	BeforeEach(func() {
		targetListener, _ = net.Listen("tcp", "0.0.0.0:8081")
		proxyListener, _ = net.Listen("tcp", "0.0.0.0:8080")

		go http.Serve(targetListener, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			io.WriteString(writer, "hello, world")
		}))

		targetUrl, _ := url.Parse("http://127.0.0.1:8081/")
		proxy = Proxy(targetUrl)
	})

	AfterEach(func() {
		targetListener.Close()
		proxyListener.Close()
	})

	It("proxies without any configuration", func() {
		go http.Serve(proxyListener, proxy)

		resp, _ := http.Get("http://127.0.0.1:8080")
		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)
		Expect(string(body)).To(Equal("hello, world"))
	})
})
