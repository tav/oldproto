// Public Domain (-) 2013 The Espra Authors.
// See the Espra UNLICENSE file for details.

package main

import (
	"fmt"
)

const htmlTop = `<!doctype html>
<meta charset=utf-8>
<title>`

const htmlBottom = `</title>
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
	htmlIndex       = []byte(htmlTop + "Files Endpoint" + htmlBottom + "Files Endpoint")
	htmlIndexLength = fmt.Sprintf("%d", len(htmlIndex))
	html400         = []byte(htmlTop + "400 Bad Request" + htmlBottom + "400 Bad Request")
	html400Length   = fmt.Sprintf("%d", len(html400))
	html401         = []byte(htmlTop + "401 Unauthorized" + htmlBottom + "401 Unauthorized")
	html401Length   = fmt.Sprintf("%d", len(html401))
	html404         = []byte(htmlTop + "404 Not Found" + htmlBottom + "404 Not Found")
	html404Length   = fmt.Sprintf("%d", len(html404))
	html503         = []byte(htmlTop + "503 Service Unavailable" + htmlBottom + "503 Service Unavailable")
	html503Length   = fmt.Sprintf("%d", len(html503))
)
