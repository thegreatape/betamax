package main

import (
	"flag"
	"fmt"

	"github.com/thegreatape/betamax/proxy"

	"net"
	"net/http"
	"net/url"
	"os"
)

func main() {
	cassetteDirectory := flag.String("cassete-directory", "./cassettes", "directory when recorded interactions are written")
	port := flag.Int("port", 8080, "port for proxy to listen on")
	target := flag.String("target-url", "", "remote target url to proxy requests to")

	flag.Parse()

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", *port))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if *target == "" {
		fmt.Println("No target url given.")
		flag.Usage()
		os.Exit(1)
	}

	targetUrl, err := url.Parse(*target)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	proxy := proxy.Proxy(targetUrl, *cassetteDirectory)

	fmt.Printf("betamax server proxy to %s listening on 0.0.0.0:%d\n", targetUrl, *port)
	http.Serve(listener, proxy)
}
