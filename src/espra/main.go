// Public Domain (-) 2011-2013 The Espra Authors.
// See the Espra UNLICENSE file for details.

package espra

import (
	"appengine"
	"espra/backend"
	"espra/rpc"
	"net/http"
)

var (
	devServer    bool
	html404      = []byte(htmlErr404Str)
	htmlHome     = []byte(htmlHomeStr)
	htmlRedirect = []byte(htmlRedirectStr)
)

func handle(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Host
	path := r.URL.Path
	if devServer {
		query := r.URL.Query()
		if query.Get("__host__") != "" {
			host = query.Get("__host__")
		}
	} else if r.URL.Scheme != "https" && host != redirectHost {
		r.URL.Scheme = "https"
		http.Redirect(w, r, r.URL.String(), 301)
		return
	}
	switch host {
	case officialHost, "":
		// Fast path the root request.
		if path == "/" {
			renderIndex(w, r)
			return
		}
		if len(path) >= 2 && path[:2] == "/_" {
			switch path {
			case "/_api":
				rpc.Handle(path, w, r)
			case "/_ah/start":
				backend.Start(w, r)
			case "/_ah/stop":
				backend.Stop(w, r)
			default:
				w.WriteHeader(404)
				w.Write(html404)
			}
		} else {
			renderIndex(w, r)
		}
	case redirectHost:
		render(htmlRedirect, w)
	default:
		// TODO(tav): Extend this to support scripted interfaces on custom
		// domains.
		r.URL.Host = officialHost
		http.Redirect(w, r, r.URL.String(), 301)
		return
	}
}

func render(c []byte, w http.ResponseWriter) {
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("X-XSS-Protection", "0")
	w.Write(c)
}

func renderIndex(w http.ResponseWriter, r *http.Request) {
	auth, err := r.Cookie("auth")
	if err == nil && auth.Value == "1" {
		render(htmlHome, w)
		return
	}
	render(htmlHome, w)
	// render(htmlAnonHome, w)
}

func init() {
	if appengine.IsDevAppServer() {
		devServer = true
	}
	http.DefaultServeMux.Handle("/", http.HandlerFunc(handle))
}
