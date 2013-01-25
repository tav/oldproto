// Public Domain (-) 2013 The Espra Authors.
// See the Espra UNLICENSE file for details.

package main

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/dchest/siphash"
	"github.com/tav/golly/aws"
	"github.com/tav/golly/runtime"
	"net"
	"time"
)

const (
	routeMask = 0xffff
	routeSize = 1 << 16
)

var (
	cmdDelete = []byte{'D'}
	cmdGet    = []byte{'G'}
	cmdPing   = []byte{'P'}
	cmdQuery  = []byte{'Q'}
	cmdRenew  = []byte{'R'}
	cmdSet    = []byte{'S'}
)

type Client struct {
	free *Client
}

type Node struct {
	Addr  string
	SeqID uint64
}

type MasterController interface {
	IsSelf() bool
	Run([]byte)
}

type Master struct {
	addr         string
	id           []byte
	isSelf       bool
	awsAccess    string
	awsSecret    string
	dynamoRegion *aws.Region
	dynamoTable  string
	httpClient   *http.Client
	mutex        sync.Mutex
	seen         map[string]int64
	selfID       []byte
	seqBytes     []byte
	seqID        uint64
	routes       [routeSize][]string
}

func (m *Master) IsDistributed() bool {
	return true
}

func (m *Master) IsSelf() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.isSelf
}

func (m *Master) Run(id []byte) {
	m.selfID = id
	for {

	}
}

type LocalMaster struct {
	addr     string
	mutex    sync.Mutex
	seqID    uint64
	seqBytes []byte
}

func (m *LocalMaster) GetSeq() []byte {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.seqBytes
}

func (m *LocalMaster) IsSelf() bool {
	return true
}

func (m *LocalMaster) Run(id []byte) {
}

func NewMaster(singleNode, awsAccess, awsSecret, dynamoRegion, dynamoTable string, masterTimeout time.Duration) MasterController {
	if awsAcess != "" || awsSecret != "" || dynamoTable != "" {
		if awsAcess == "" {
			runtime.Error("Distributed options set, but aws-access-key hasn't been specified")
		}
		if awsSecret == "" {
			runtime.Error("Distributed options set, but aws-secret-key hasn't been specified")
		}
		if dynamoTable == "" {
			runtime.Error("Distributed options set, but master-table hasn't been specified")
		}
		region, ok := aws.Regions[dynamoRegion]
		if !ok {
			runtime.Error("AWS Region %q not known", dynamoRegion)
		}
		return &Master{}
	}
	return &LocalMaster{addr: singleNode}
}

func NewClient(master MasterController) {
}

func encodeInt(v int64) {
	return
}

func getAddrFromID(id []byte) string {
	return fmt.Sprintf("%s:%d", net.IP(id[:4]), binary.LittleEndian.Uint16(id[4:6]))
}

func getNodeID(host string, port int) []byte {
	ip := net.ParseIP(host).To4()
	if ip == nil || len(ip) != 4 {
		runtime.Error("Could not parse host %q into a 32-bit IPv4 address", host)
	}
	id := make([]byte, 14)
	copy(id[:4], ip)
	binary.LittleEndian.PutUint16(id[4:6], uint16(port))
	binary.LittleEndian.PutUint64(id[6:], uint64(time.Now().UnixNano()<<1))
	return id
}

func getSlot(v []byte) uint64 {
	return siphash.Hash(hashKey0, hashKey1, v) & routeMask
}

func initHashKey(s string) {
	if len(s) != 32 {
		runtime.Error("The hash-key value %q is not a 32-byte string.", s)
	}
	k, err := hex.DecodeString(s)
	if err != nil {
		runtime.Error("The hash-key value %q is not a hex-encoded string: %s", s, err)
	}
	hashKey0 = binary.LittleEndian.Uint64(k[:8])
	hashKey1 = binary.LittleEndian.Uint64(k[8:])
}
