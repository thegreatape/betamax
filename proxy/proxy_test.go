package proxy_test

import (
	"fmt"
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
	var targetPort string
	var proxyListener net.Listener
	var proxyPort string

	BeforeEach(func() {
		targetListener, _ = net.Listen("tcp", "0.0.0.0:0")
		_, targetPort, _ = net.SplitHostPort(targetListener.Addr().String())
		proxyListener, _ = net.Listen("tcp", "0.0.0.0:0")
		_, proxyPort, _ = net.SplitHostPort(targetListener.Addr().String())

		go http.Serve(targetListener, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			io.WriteString(writer, "hello, world")
		}))

		targetUrl, _ := url.Parse("http://127.0.0.1:8081/")
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

	It("refuses conneciton if the target server is down", func() {
		targetListener.Close()
		_, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s", proxyPort))
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("connection refused"))
	})
})
