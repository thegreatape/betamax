package main

import (
  "net"
  "net/http"
  "io"
)

func main() {
  listener, _ := net.Listen("tcp", "0.0.0.0:8080")
  http.Serve(listener, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request){
    io.WriteString(writer, "goodbye, world!\n")
    listener.Close()
  }))
}
