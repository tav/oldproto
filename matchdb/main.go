// Public Domain (-) 2012-2013 The Espra Authors.
// See the Espra UNLICENSE file for details.

package main

import (
	"bufio"
	"github.com/tav/golly/log"
	"github.com/tav/golly/optparse"
	"github.com/tav/golly/runtime"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

type Entry struct {
	id      string
	listing []byte
}

type DB struct {
	alloc       int64
	limit       int64
	limitNotify int64
	id          []byte
	master      MasterController
	mutex       sync.Mutex
	timeout     time.Duration
	seen        map[string]int64
	table       map[string]Entry
}

func (db *DB) Get() {
	return
}

func (db *DB) Ping() {
	return
}

func (db *DB) Set() {
	return
}

func (db *DB) Handle(conn net.Conn) {

	id := make([]byte, 14)

	n, err := conn.Read(id)
	if err != nil || n != 14 {
		conn.Close()
		return
	}

	addr := getAddrFromID(id)
	r := bufio.NewReaderSize(conn, 1024)
	w := bufio.NewWriterSize(conn, 1024)

	defer conn.Close()

	for {

	}

}

func (db *DB) Run(host string, port int, limit int64, master MasterController) error {

	if host == "" {
		host = runtime.GetIP()
	}

	db.id = getNodeID(host, port)
	db.limit = limit << 10
	db.limitNotify = (db.limit * 2) / 3

	addr, listener := runtime.GetAddrListener(host, port)

	defer listener.Close()
	go master.Run(db.id)

	log.Info("MatchDB running on %s", addr)

	delay := 1 * time.Millisecond
	maxDelay := 1 * time.Second

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if delay == 0 {
					delay = 5 * time.Millisecond
				} else {
					delay *= 2
				}
				if delay > maxDelay {
					delay = maxDelay
				}
				log.Error("Accept error: %v; retrying in %v", err, delay)
				time.Sleep(delay)
				continue
			}
			log.Error("Accept error: %v", err)
			return err
		}
		delay = 0
		go db.Handle(conn)
	}

}

func main() {

	opts := optparse.Parser(
		"Usage: matchdb <config.yaml> [options]\n",
		"matchdb 0.0.1")

	host := opts.StringConfig("host", "",
		"the host to bind matchdb to")

	port := opts.IntConfig("port", 8090,
		"the port to bind matchdb to [8090]")

	allocLimit := opts.IntConfig("alloc-limit", 5000000,
		"maximum allocation limit in kilobytes [5000000]")

	hashKey := opts.StringConfig("hash-key", "",
		"16-byte hash key encoded as a 32-byte hex string")

	awsAccessKey := opts.StringConfig("aws-access-key", "",
		"the AWS Access Key for DynamoDB")

	awsSecretKey := opts.StringConfig("aws-secret-key", "",
		"the AWS Secret Key for DynamoDB")

	awsRegion := opts.StringConfig("aws-region", "us-east-1",
		"the AWS Region for DynamoDB [us-east-1]")

	masterTable := opts.StringConfig("master-table", "",
		"the DynamoDB table for the master lock")

	masterTimeout := opts.IntConfig("master-timeout", 6000,
		"timeout in milliseconds for the master lock [6000]")

	os.Args[0] = "matchdb"
	runtime.DefaultOpts("matchdb", opts, os.Args)

	initHashKey(*hashKey)
	master := NewMaster("", *awsAccessKey, *awsSecretKey, *awsRegion, *masterTable, time.Duration(*masterTimeout)*time.Millisecond)

	db := &DB{}
	db.Run(*host, *port, master, int64(*allocLimit))

}
