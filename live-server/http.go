// Public Domain (-) 2010-2013 The Espra Authors.
// See the Espra UNLICENSE file for details.

package main

import (
	"fmt"
	"github.com/tav/golly/log"
	"github.com/tav/golly/runtime"
	"net"
	"net/http"
	"strings"
)

var (
	behindProxy bool
	logPrefix   string
)

// Log event types.
const (
	HTTP_PING = iota
	HTTP_REDIRECT
	HTTPS_INTERNAL_ERROR
	HTTPS_MAINTENANCE
	HTTPS_REDIRECT
	HTTPS_STATIC
	HTTPS_UPSTREAM
	HTTPS_UPSTREAM_ERROR
	HTTPS_WEBSOCKET
)

type Redirector struct {
	hsts       string
	html       []byte
	htmlLength string
	pong       []byte
	url        string
}

func (redirector *Redirector) ServeHTTP(conn http.ResponseWriter, req *http.Request) {

	if req.URL.Path == "/ping" {
		headers := conn.Header()
		headers.Set("Content-Type", "text/plain")
		headers.Set("Content-Length", "4")
		conn.WriteHeader(http.StatusOK)
		conn.Write(redirector.pong)
		logRequest(HTTP_PING, http.StatusOK, req.Host, req)
		return
	}

	if redirector.hsts != "" {
		conn.Header().Set("Strict-Transport-Security", redirector.hsts)
	}

	conn.Header().Set("Location", redirector.url)
	conn.Header().Set("Content-Type", "text/html")
	conn.Header().Set("Content-Length", redirector.htmlLength)
	conn.WriteHeader(http.StatusMovedPermanently)
	conn.Write(redirector.html)
	logRequest(HTTP_REDIRECT, http.StatusMovedPermanently, req.Host, req)

}

func logRequest(proto, status int, host string, request *http.Request) {
	var ip string
	splitPoint := strings.LastIndex(request.RemoteAddr, ":")
	if splitPoint == -1 {
		ip = request.RemoteAddr
	} else {
		ip = request.RemoteAddr[0:splitPoint]
	}
	log.InfoData(logPrefix, proto, status, request.Method, host, request.URL,
		ip, request.UserAgent(), request.Referer())
}

func serveRedirector(listener net.Listener, redirector *Redirector) {
	err := http.Serve(listener, redirector)
	if err != nil {
		runtime.Error("Couldn't serve HTTP Redirector: %s", err)
	}
}

func RunRedirector(host string, port int, url string, hsts int) {
	if url == "" {
		return
	}
	r := &Redirector{}
	if hsts != 0 {
		r.hsts = fmt.Sprintf("max-age=%d", hsts)
	}
	r.html = []byte(fmt.Sprintf(
		`Please <a href="%s">click here if your browser doesn't redirect</a> automatically.`,
		url))
	r.htmlLength = fmt.Sprintf("%d", len(r.html))
	r.pong = []byte("PONG")
	r.url = url
	addr := fmt.Sprintf("%s:%d", host, port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		runtime.Error("Cannot listen on %s: %v", addr, err)
	}
	go serveRedirector(listener, r)
	if host == "" {
		host = "localhost"
	}
	log.Info("HTTP Redirector running on http://%s:%d -> %s", host, port, url)
}
