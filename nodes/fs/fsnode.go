package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/debouncer"
	"github.com/awgh/ratnet/api"
	"github.com/awgh/ratnet/api/events"
	"github.com/awgh/ratnet/nodes"
	"github.com/awgh/ratnet/router"
)

// OutBufferSize - size of the buffer in messages for the Out() channel
var OutBufferSize = 128

type outboxMsg struct {
	channel   string
	msg       []byte
	timeStamp int64
}

// Node : defines an instance of the API with a ql-DB backed Node
type Node struct {
	contentKey bc.KeyPair
	routingKey bc.KeyPair

	policies  []api.Policy
	router    api.Router
	isRunning uint32

	// external data members
	in     chan api.Msg
	out    chan api.Msg
	events chan api.Event

	// db -> ram replacements
	channels map[string]*api.ChannelPriv
	config   map[string]string
	contacts map[string]*api.Contact
	peers    map[string]*api.Peer
	profiles map[string]*api.ProfilePriv
	streams  map[uint32]*api.StreamHeader
	chunks   map[uint32]map[uint32]*api.Chunk

	basePath string

	mutex         sync.RWMutex
	trigggerMutex sync.Mutex
	debouncer     *debouncer.Debouncer
}

// New : creates a new instance of API
func New(contentKey, routingKey bc.KeyPair, basePath string) *Node {
	// create node
	node := new(Node)

	// init assorted other
	node.channels = make(map[string]*api.ChannelPriv)
	node.config = make(map[string]string)
	node.contacts = make(map[string]*api.Contact)
	node.peers = make(map[string]*api.Peer)
	node.profiles = make(map[string]*api.ProfilePriv)
	node.streams = make(map[uint32]*api.StreamHeader)
	node.chunks = make(map[uint32]map[uint32]*api.Chunk)

	// set crypto modes
	node.contentKey = contentKey
	node.routingKey = routingKey

	// setup chans
	node.in = make(chan api.Msg)
	node.out = make(chan api.Msg, OutBufferSize)
	node.events = make(chan api.Event, OutBufferSize)

	// setup default router
	node.router = router.NewDefaultRouter()

	node.basePath = basePath
	os.Mkdir(basePath, 0700)

	return node
}

func hex32(n uint32) string {
	return fmt.Sprintf("%08x", n)
}

func hex64(n int64) string {
	return fmt.Sprintf("%016x", n)
}

// IsRunning - returns true if this node is running
func (node *Node) IsRunning() bool {
	return atomic.LoadUint32(&node.isRunning) == 1
}

func (node *Node) setIsRunning(b bool) {
	var running uint32 = 0
	if b {
		running = 1
	}
	atomic.StoreUint32(&node.isRunning, running)
}

// GetPolicies : returns the array of Policy objects for this Node
func (node *Node) GetPolicies() []api.Policy {
	node.mutex.Lock()
	defer node.mutex.Unlock()
	return node.policies
}

// SetPolicy : set the array of Policy objects for this Node
func (node *Node) SetPolicy(policies ...api.Policy) {
	node.mutex.Lock()
	defer node.mutex.Unlock()
	node.policies = policies
}

// Router : get the Router object for this Node
func (node *Node) Router() api.Router {
	return node.router
}

// SetRouter : set the Router object for this Node
func (node *Node) SetRouter(router api.Router) {
	node.router = router
}

// FlushOutbox : Deletes outbound messages older than maxAgeSeconds seconds
func (node *Node) FlushOutbox(maxAgeSeconds int64) {
	now := time.Now()
	_ = filepath.Walk(node.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			events.Warning(node, "FlushOutbox failure accessing a path:", path, err.Error())
			return err
		}
		if !info.IsDir() {
			if diff := now.Sub(info.ModTime()); diff > time.Duration(maxAgeSeconds)*time.Second {
				events.Debug(node, "Deleting file:", filepath.Join(node.basePath, info.Name()), diff)
				if err = os.Remove(path); err != nil {
					events.Error(node, "error deleting file: "+err.Error())
				}
			}
		}
		return nil
	})
}

// Channels

// In : Returns the In channel of this node
func (node *Node) In() chan api.Msg {
	return node.in
}

// Out : Returns the Out channel of this node
func (node *Node) Out() chan api.Msg {
	return node.out
}

// Events : Returns the Events channel of this node
func (node *Node) Events() chan api.Event {
	return node.events
}

// RPC set to default handlers

// AdminRPC :
func (node *Node) AdminRPC(transport api.Transport, call api.RemoteCall) (interface{}, error) {
	return nodes.AdminRPC(transport, node, call)
}

// PublicRPC :
func (node *Node) PublicRPC(transport api.Transport, call api.RemoteCall) (interface{}, error) {
	return nodes.PublicRPC(transport, node, call)
}
