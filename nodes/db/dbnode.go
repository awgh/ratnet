package db

// To install upper db:
// go get -v -u github.com/upper/db
// FOR POSTGRES DRIVER: go get -v -u github.com/lib/pq

import (
	"sync"
	"sync/atomic"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/bencrypt/ecc"
	"github.com/awgh/debouncer"
	"github.com/awgh/ratnet/api"
	"github.com/awgh/ratnet/nodes"
	"github.com/awgh/ratnet/router"
	"github.com/upper/db/v4"
)

// OutBufferSize - Out() output go channel buffer size
var OutBufferSize = 128

// Node : defines an instance of the API with a ql-DB backed Node
type Node struct {
	contentKey  bc.KeyPair
	routingKey  bc.KeyPair
	channelKeys map[string]bc.KeyPair

	policies []api.Policy
	router   api.Router

	db db.Session

	mutex         *sync.Mutex
	trigggerMutex sync.Mutex
	debouncer     *debouncer.Debouncer

	isRunning uint32

	// external data members
	in     chan api.Msg
	out    chan api.Msg
	events chan api.Event
}

// New : creates a new instance of API
func New(contentKey, routingKey bc.KeyPair) *Node {
	// create node
	node := new(Node)
	node.mutex = &sync.Mutex{}

	// init channel key map
	node.channelKeys = make(map[string]bc.KeyPair)

	// set crypto modes
	if contentKey == nil {
		contentKey = new(ecc.KeyPair)
	}
	if contentKey.GetPubKey() == contentKey.GetPubKey().Nil() {
		contentKey.GenerateKey()
	}
	if routingKey == nil {
		routingKey = new(ecc.KeyPair)
	}
	if routingKey.GetPubKey() == routingKey.GetPubKey().Nil() {
		routingKey.GenerateKey()
	}
	node.contentKey = contentKey
	node.routingKey = routingKey

	// setup chans
	node.in = make(chan api.Msg)
	node.out = make(chan api.Msg, OutBufferSize)
	node.events = make(chan api.Event, OutBufferSize)

	// setup default router
	node.router = router.NewDefaultRouter()

	return node
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
	return node.policies
}

// SetPolicy : set the array of Policy objects for this Node
func (node *Node) SetPolicy(policies ...api.Policy) {
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
