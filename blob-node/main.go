// Public Domain (-) 2012 The Espra Authors.
// See the Espra UNLICENSE file for details.

package main

import (
	"crypto/sha1"
	"fmt"
	"github.com/tav/golly/crypto"
	"net/http"
	"strings"
)

const tmplHead = `<!DOCTYPE html>
<meta charset=utf-8>
<title>`

const tmplBody = `</title>
<link href='//fonts.googleapis.com/css?family=Droid+Sans' rel=stylesheet>
<style>
body {
  font-family: 'Droid Sans', Verdana, sans-serif;
  font-size: 40px;
  padding: 10px 7px;
}
</style>
<body>`

var (
	html401   = []byte(tmplHead + "401 Not Authorized" + tmplBody + "401 Not Authorized")
	html404   = []byte(tmplHead + "404 Not Found" + tmplBody + "404 Not Found")
	htmlIndex = []byte(tmplHead + "Files Endpoint" + tmplBody + "Blob Node Endpoint")
)

var validTypes = map[string]string{
	"application/x-midi":   "audio/midi",
	"audio/3gpp":           "audio/3gpp",
	"audio/3gpp2":          "audio/3gpp2",
	"audio/aac":            "audio/aac",
	"audio/ac3":            "audio/ac3",
	"audio/basic":          "audio/basic",
	"audio/mid":            "audio/midi",
	"audio/midi":           "audio/midi",
	"audio/mp3":            "audio/mp3",
	"audio/mp4":            "audio/mp4",
	"audio/mpeg":           "audio/mpeg",
	"audio/mpeg3":          "audio/mp3",
	"audio/x-aac":          "audio/aac",
	"audio/x-ac3":          "audio/ac3",
	"audio/x-aiff":         "audio/aiff",
	"audio/x-m4a":          "audio/x-m4a",
	"audio/x-mid":          "audio/midi",
	"audio/x-midi":         "audio/midi",
	"audio/x-mp3":          "audio/mp3",
	"audio/x-mpeg3":        "audio/mpeg3",
	"audio/x-wav":          "audio/wav",
	"audio/wav":            "audio/wav",
	"image/bmp":            "image/bmp",
	"image/gif":            "image/gif",
	"image/jpeg":           "image/jpeg",
	"image/jpeg2000-image": "image/jpeg2000-image",
	"image/pjpeg":          "image/jpeg",
	"image/png":            "image/png",
	"image/tiff":           "image/tiff",
	"image/x-bmp":          "image/bmp",
	"image/x-png":          "image/png",
	"image/x-tiff":         "image/tiff",
	"image/x-windows-bmp":  "image/bmp",
	"video/3gpp":           "video/3gpp",
	"video/3gpp2":          "video/3gpp2",
	"video/avi":            "video/avi",
	"video/mp4":            "video/mp4",
	"video/mpeg":           "video/mpeg",
	"video/msvideo":        "video/avi",
	"video/quicktime":      "video/quicktime",
	"video/x-mpeg":         "video/mpeg",
	"video/x-msvideo":      "video/avi",
}

func isInvalidURL(url string) bool {
	url = strings.ToLower(url)
	if strings.HasSuffix(url, "http://") || strings.HasSuffix(url, "https://") {
		return false
	}
	return true
}

func handle(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path != "/" {
		w.WriteHeader(404)
		w.Write(html404)
		return
	}

	if r.URL.RawQuery == "" {
		w.Write(htmlIndex)
		return
	}

	params := r.URL.Query()
	key := params.Get("key")
	url := params.Get("url")

	if key == "" || url == "" || isInvalidURL(url) {
		w.WriteHeader(404)
		w.Write(html404)
		return
	}

	user, ok := crypto.GetIronValue("files", key, userKey, true)
	if !ok {
		w.WriteHeader(401)
		w.Write(html401)
		return
	}

	hash := sha1.New()
	hash.Write([]byte(url))
	shasum := hash.Sum(nil)

	// 	w.Header().Set("Strict-Transport-Security", "max-age=31536000")
	// 	w.Header().Set("X-Frame-Options", "DENY")

	fmt.Fprintf(w, "user: %q\n", digest)
	fmt.Fprintf(w, "user: %v\n", user)

}

func main() {
	http.DefaultServeMux.Handle("/", http.HandlerFunc(handle))
}
