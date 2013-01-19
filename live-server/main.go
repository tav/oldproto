// Public Domain (-) 2010-2013 The Espra Authors.
// See the Espra UNLICENSE file for details.

package main

import (
	"bufio"
	"bytes"
	"github.com/garyburd/go-websocket/websocket"
	// "compress/gzip"
	// "crypto/tls"
	"crypto/subtle"
	// "encoding/json"
	"encoding/gob"
	"fmt"
	"github.com/tav/golly/log"
	"github.com/tav/golly/optparse"
	"github.com/tav/golly/runtime"
	"github.com/tav/golly/tlsconf"
	"io"
	// "io/ioutil"
	// "net"
	"net/http"
	"os"
	"strconv"
	"time"
	// "strings"
	"sync"
)

const (
	logPrefix      = "ls"
	maxPublishSize = 1 << 22
	nodeIDLength   = 32
)

var (
	debugMode = false
	respOK    = []byte("OK")
)

type LiveServer struct {
	hsts             string
	hstsEnabled      bool
	httpClient       *http.Client
	inMaintenance    bool
	mutex            sync.RWMutex
	publishAck       []byte
	publishAckURL    string
	publishKey       []byte
	publishKeyLength int
	upstreamAddr     string
	upstreamHost     string
	upstreamPort     int
	upstreamTLS      bool
}

type Item struct {
	ID string
}

func (s *LiveServer) HandlePublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		serve404(w, r)
		return
	}
	defer r.Body.Close()
	cl, err := strconv.ParseUint(r.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		serve400(w, r)
		return
	}
	if cl > maxPublishSize {
		serve400(w, r)
		return
	}
	body := bufio.NewReader(r.Body)
	key, err := body.ReadBytes('\n')
	if err != nil {
		serve400(w, r)
		return
	}
	if subtle.ConstantTimeCompare(key, s.publishKey) != 1 {
		serve401(w, r)
		return
	}
	dec := gob.NewDecoder(body)
	item := &Item{}
	err = dec.Decode(item)
	if err != nil || item == nil {
		serve400(w, r)
		return
	}
	go s.Publish(item)
	w.Write(respOK)
	logRequest(HTTP_OK, http.StatusOK, r)
	return
}

func (s *LiveServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	fmt.Println("BOOM")
	conn, err := websocket.Upgrade(w, r.Header, "", 1024, 1024)
	if err != nil {
		log.StandardError(err)
		serve400(w, r)
		return
	}
	defer conn.Close()
	for {
		op, r, err := conn.NextReader()
		if err != nil {
			return
		}
		if op != websocket.OpBinary && op != websocket.OpText {
			continue
		}
		w, err := conn.NextWriter(op)
		if err != nil {
			return
		}
		io.Copy(w, r)
		w.Close()
	}
}

func (s *LiveServer) AckPublish(id string, tries time.Duration) {
	resp, err := s.httpClient.Post(s.publishAckURL+id, "text/plain", bytes.NewBuffer(s.publishAck))
	if err != nil {
		time.Sleep(tries * 100 * time.Millisecond)
		if tries == 10 {
			return
		}
		s.AckPublish(id, tries+1)
		return
	}
	resp.Body.Close()
}

func (s *LiveServer) Publish(item *Item) {
	s.AckPublish(item.ID, 1)
}

func (s *LiveServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Set default headers.
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if s.hstsEnabled {
		w.Header().Set("Strict-Transport-Security", s.hsts)
	}

	// Return the HTTP 503 error page if we're in maintenance mode.
	s.mutex.RLock()
	if s.inMaintenance {
		s.mutex.RUnlock()
		serve503(w, r)
		return
	}
	s.mutex.RUnlock()

	switch r.URL.Path {
	case "/":
		serveIndex(w, r)
	case "/connect":
		s.HandleWebSocket(w, r)
	case "/ping":
		servePong(w, r)
	case "/publish":
		s.HandlePublish(w, r)
	default:
		serve404(w, r)
	}

}

func (s *LiveServer) SetMaintenance(status bool) {
	s.mutex.Lock()
	s.inMaintenance = status
	s.mutex.Unlock()
}

func setAckInfo(id int, s *LiveServer) {
	if id == 0 {
		runtime.Error("The publish-cluster-id cannot be 0")
	}
	if (id & (id - 1)) != 0 {
		runtime.Error("The publish-cluster-id is not a power of 2")
	}
	s.publishAck = []byte(fmt.Sprintf("%s,%d,%d", string(s.publishKey), id, id))
	if s.upstreamTLS && s.upstreamPort == 443 {
		s.publishAckURL = fmt.Sprintf("https://%s/_ack_publish/", s.upstreamHost)
	} else {
		s.publishAckURL = fmt.Sprintf("https://%s/_ack_publish/", s.upstreamAddr)
	}
}

func main() {

	// Define the options for the command line and config file options parser.
	opts := optparse.Parser(
		"Usage: live-server <config.yaml> [options]\n",
		"live-server 0.0.1")

	host := opts.StringConfig("host", "",
		"the host to bind the live-server to")

	port := opts.IntConfig("port", 9040,
		"the port to bind the live-server to [9040]")

	redirectURL := opts.StringConfig("redirect-url", "",
		"the URL that the HTTP Redirector redirects to")

	redirectorHost := opts.StringConfig("redirector-host", "",
		"the host to bind the HTTP Redirector to")

	redirectorPort := opts.IntConfig("redirector-port", 9080,
		"the port to bind the HTTP Redirector to [9080]")

	hstsMaxAge := opts.IntConfig("hsts-max-age", 31536000,
		"max-age value of HSTS in number of seconds [0 (disabled)]")

	// awsAccessKey := opts.StringConfig("aws-access-key", "",
	// 	"the AWS Access Key for DynamoDB")

	// awsSecretKey := opts.StringConfig("aws-secret-key", "",
	// 	"the AWS Secret Key for DynamoDB")

	awsRegion := opts.StringConfig("aws-region", "us-east-1",
		"the AWS Region for DynamoDB [us-east-1]")

	// masterTable := opts.StringConfig("master-table", "",
	// 	"the DynamoDB table for the master lock")

	// masterTimeout := opts.IntConfig("master-timeout", 6000,
	// 	"timeout in milliseconds for the master lock [6000]")

	// routingTimeout := opts.IntConfig("routing-timeout", 3000,
	// 	"timeout in milliseconds for routing entries [3000]")

	publishKey := opts.StringConfig("publish-key", "",
		"the shared secret for publishing new items")

	publishClusterID := opts.IntConfig("publish-cluster-id", 0,
		"the cluster id to use when acknowledging publish requests")

	upstreamHost := opts.StringConfig("upstream-host", "localhost",
		"the upstream host to connect to [localhost]")

	upstreamPort := opts.IntConfig("upstream-port", 8080,
		"the upstream port to connect to [8080]")

	upstreamTLS := opts.BoolConfig("upstream-tls", false,
		"use TLS when connecting to upstream [false]")

	maintenanceMode := opts.BoolConfig("maintenance", false,
		"start up in maintenance mode [false]")

	// Setup the console log filter.
	log.ConsoleFilters[logPrefix] = func(items []interface{}) (bool, []interface{}) {
		return true, items[2 : len(items)-2]
	}

	// Parse the command line options.
	os.Args[0] = "live-server"
	debugMode, _, _, _ = runtime.DefaultOpts("live-server", opts, os.Args)

	// Initialise the TLS config if using TLS to communicate upstream.
	if *upstreamTLS {
		tlsconf.Init()
	}

	// Initialise ping/pong variables.
	setupPong(*awsRegion)

	// Ensure required config values.
	if *publishKey == "" {
		runtime.Error("The publish-key cannot be empty")
	}

	server := &LiveServer{
		publishKey:       []byte(*publishKey),
		publishKeyLength: len(*publishKey),
		upstreamAddr:     fmt.Sprintf("%s:%d", *upstreamHost, *upstreamPort),
		upstreamHost:     *upstreamHost,
		upstreamPort:     *upstreamPort,
		upstreamTLS:      *upstreamTLS,
	}

	if *hstsMaxAge != 0 {
		server.hstsEnabled = true
		server.hsts = fmt.Sprintf("max-age=%d", *hstsMaxAge)
	}

	// Set response payload for acknowledging successful publish calls.
	setAckInfo(*publishClusterID, server)

	// Enable maintenance handling.
	frontends := []Maintainable{server}
	handleMaintenance(frontends, *maintenanceMode)

	// Setup the HTTP Redirector.
	runRedirector(*redirectorHost, *redirectorPort, *redirectURL, *hstsMaxAge)

	// Run the Live Server.
	runHTTP("Live Server", *host, *port, server, "")

	// Enter the wait loop for the process to be killed.
	loopForever := make(chan bool, 1)
	<-loopForever

}
