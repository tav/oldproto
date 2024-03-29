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
	"syscall"
)

var (
	behindProxy  bool
	pongLength   = "0"
	pongResponse = []byte(`handlePing("`)
)

// Log event types.
const (
	HTTP_BAD_REQUEST = iota
	HTTP_INTERNAL_ERROR
	HTTP_MAINTENANCE
	HTTP_NOT_FOUND
	HTTP_OK
	HTTP_PING
	HTTP_REDIRECT
	HTTP_STATIC
	HTTP_UNAUTHORIZED
	HTTP_UPSTREAM_ERROR
	HTTP_WEBSOCKET
)

type Maintainable interface {
	SetMaintenance(bool)
}

func handleMaintenance(frontends []Maintainable, initState bool) {
	if initState {
		for _, f := range frontends {
			f.SetMaintenance(true)
		}
	}
	ch := make(chan bool, 1)
	go func() {
		for {
			enable := <-ch
			for _, f := range frontends {
				if enable {
					f.SetMaintenance(true)
				} else {
					f.SetMaintenance(false)
				}
			}
		}
	}()
	runtime.SignalHandlers[syscall.SIGUSR1] = func() {
		ch <- true
	}
	runtime.SignalHandlers[syscall.SIGUSR2] = func() {
		ch <- false
	}
}

type RedirectServer struct {
	hsts       string
	html       []byte
	htmlLength string
	url        string
}

func (s *RedirectServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/ping" {
		servePong(w, r)
		return
	}
	if s.hsts != "" {
		w.Header().Set("Strict-Transport-Security", s.hsts)
	}
	w.Header().Set("Location", s.url)
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Content-Length", s.htmlLength)
	w.WriteHeader(http.StatusMovedPermanently)
	w.Write(s.html)
	logRequest(HTTP_REDIRECT, http.StatusMovedPermanently, r)
}

func isGlobalIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if !ip.IsGlobalUnicast() {
		return false
	}
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 10 {
			return false
		}
		if ip4[0] == 192 && ip4[1] == 168 {
			return false
		}
		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			return false
		}
	}
	// TODO(tav): Exclude unique local addresses within the block fc00::/7 for
	// IPv6 addresses.
	return true
}

func logRequest(proto, status int, r *http.Request) {
	var ip string
	if behindProxy {
		if xfwd := r.Header.Get("X-Forwarded-For"); xfwd != "" {
			if strings.Contains(xfwd, ",") {
				for _, addr := range strings.Split(xfwd, ",") {
					if p := net.ParseIP(strings.TrimSpace(addr)); isGlobalIP(p) {
						ip = p.String()
						break
					}
				}
			} else if p := net.ParseIP(xfwd); isGlobalIP(p) {
				ip = p.String()
			}
		}
	} else {
		splitPoint := strings.LastIndex(r.RemoteAddr, ":")
		if splitPoint == -1 {
			ip = r.RemoteAddr
		} else {
			ip = r.RemoteAddr[0:splitPoint]
		}
	}
	log.InfoData(logPrefix, proto, status, r.Method, r.Host, r.URL,
		ip, r.UserAgent(), r.Referer())
}

func runRedirector(host string, port int, url string, hsts int) {
	if url == "" {
		return
	}
	s := &RedirectServer{}
	if hsts != 0 {
		s.hsts = fmt.Sprintf("max-age=%d", hsts)
	}
	s.html = []byte(fmt.Sprintf(
		`Please <a href="%s">click here if your browser doesn't redirect</a> automatically.`,
		url))
	s.htmlLength = fmt.Sprintf("%d", len(s.html))
	s.url = url
	runHTTP("HTTP Redirector", host, port, s, " -> "+url)
}

func runHTTP(name string, host string, port int, handler http.Handler, suffix string) {
	addr := fmt.Sprintf("%s:%d", host, port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		runtime.Error("Cannot listen on %s: %v", addr, err)
	}
	go serveHTTP(name, listener, handler)
	if host == "" {
		host = "localhost"
	}
	log.Info("%s running on http://%s:%d%s", name, host, port, suffix)
}

func serveHTTP(name string, listener net.Listener, handler http.Handler) {
	err := http.Serve(listener, handler)
	if err != nil {
		runtime.Error("Couldn't serve %s: %s", name, err)
	}
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", htmlIndexLength)
	w.WriteHeader(http.StatusOK)
	w.Write(htmlIndex)
	logRequest(HTTP_OK, http.StatusOK, r)
}

func servePong(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Content-Length", pongLength)
	w.WriteHeader(http.StatusOK)
	w.Write(pongResponse)
	logRequest(HTTP_PING, http.StatusOK, r)
}

func serve400(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", html400Length)
	w.WriteHeader(http.StatusBadRequest)
	w.Write(html400)
	logRequest(HTTP_BAD_REQUEST, http.StatusNotFound, r)
}

func serve401(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", html401Length)
	w.WriteHeader(http.StatusUnauthorized)
	w.Write(html401)
	logRequest(HTTP_UNAUTHORIZED, http.StatusUnauthorized, r)
}

func serve404(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", html404Length)
	w.WriteHeader(http.StatusNotFound)
	w.Write(html404)
	logRequest(HTTP_NOT_FOUND, http.StatusNotFound, r)
}

func serve503(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", html503Length)
	w.WriteHeader(http.StatusServiceUnavailable)
	w.Write(html503)
	logRequest(HTTP_MAINTENANCE, http.StatusServiceUnavailable, r)
}

func setupPong(typ, id string) {
	typ += `", "` + id + `");`
	pongResponse = append(pongResponse, typ...)
	pongLength = fmt.Sprintf("%d", len(pongResponse))
}
