// Public Domain (-) 2012-2013 The Espra Authors.
// See the Espra UNLICENSE file for details.

package main

import (
	"fmt"
	"github.com/tav/golly/log"
	"github.com/tav/golly/optparse"
	"github.com/tav/golly/runtime"
	"net"
	"net/rpc"
	"os"
	"sync"
	"time"
)

type DB struct {
	accessKey      string
	distributed    bool
	masterTable    string
	masterTimeout  time.Duration
	mutex          sync.Mutex
	routingTable   string
	routingTimeout time.Duration
	secretKey      string
}

func (db *DB) init() {
	for {

	}
}

func (db *DB) Ping(args *Ping, reply *Pong) error {
	reply.Value = "PONG"
	return nil
}

func main() {

	opts := optparse.Parser(
		"Usage: matchdb <config.yaml> [options]\n",
		"matchdb 0.0.1")

	host := opts.StringConfig("host", "",
		"the host to bind matchdb to")

	port := opts.IntConfig("port", 8090,
		"the port to bind matchdb to [8090]")

	awsAccessKey := opts.StringConfig("aws-access-key", "",
		"the AWS Access Key for DynamoDB")

	awsSecretKey := opts.StringConfig("aws-secret-key", "",
		"the AWS Secret Key for DynamoDB")

	masterTable := opts.StringConfig("master-table", "",
		"the DynamoDB table for the master lock")

	masterTimeout := opts.IntConfig("master-timeout", 6,
		"timeout in seconds for the master lock [6]")

	routingTable := opts.StringConfig("routing-table", "",
		"the DynamoDB table for the routing table")

	routingTimeout := opts.IntConfig("routing-timeout", 3,
		"timeout in seconds for routing entries [6]")

	os.Args[0] = "matchdb"
	runtime.DefaultOpts("matchdb", opts, os.Args)

	var distributed bool

	if *awsAccessKey != "" || *awsSecretKey != "" || *masterTable != "" || *routingTable != "" {
		if *awsAccessKey == "" {
			runtime.Error("Distributed options set, but aws-access-key hasn't been specified")
		}
		if *awsSecretKey == "" {
			runtime.Error("Distributed options set, but aws-secret-key hasn't been specified")
		}
		if *masterTable == "" {
			runtime.Error("Distributed options set, but master-table hasn't been specified")
		}
		if *routingTable == "" {
			runtime.Error("Distributed options set, but routing-table hasn't been specified")
		}
		distributed = true
	}

	addr := fmt.Sprintf("%s:%d", *host, *port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		runtime.Error("Cannot listen on %s: %v", addr, err)
	}

	db := &DB{}
	if distributed {
		db := &DB{
			accessKey:      *awsAccessKey,
			distributed:    distributed,
			secretKey:      *awsSecretKey,
			masterTable:    *masterTable,
			masterTimeout:  time.Duration(*masterTimeout) * time.Second,
			routingTable:   *routingTable,
			routingTimeout: time.Duration(*routingTimeout) * time.Second,
		}
		go db.init()
	}

	log.Info("MatchDB running on %s", addr)
	rpc.RegisterName("db", db)
	rpc.Accept(listener)

}
