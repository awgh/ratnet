package qldb

// To install ql:
//force github.com/cznic/zappy to purego mode
//go get -tags purego github.com/cznic/ql  (or ql+cgo seems to work on arm now, too)

import (
	"database/sql"
	"sync"
	"sync/atomic"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/ratnet/api"
	"github.com/awgh/ratnet/nodes"
	"github.com/awgh/ratnet/router"

	_ "modernc.org/ql/driver" // load the QL database driver
)

var OutBufferSize = 128

// Node : defines an instance of the API with a ql-DB backed Node
type Node struct {
	contentKey  bc.KeyPair
	routingKey  bc.KeyPair
	channelKeys map[string]bc.KeyPair

	policies []api.Policy
	router   api.Router
	db       func() *sql.DB
	mutex    *sync.Mutex

	isRunning uint32

	debugMode bool

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
	node.contentKey = contentKey
	node.routingKey = routingKey

	// setup chans
	node.in = make(chan api.Msg)
	node.out = make(chan api.Msg, OutBufferSize)
	node.events = make(chan api.Event)

	// setup default router
	node.router = router.NewDefaultRouter()

	return node
}

func (node *Node) getIsRunning() bool {
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

// Debug

// GetDebug : Returns the debug mode status of this node
func (node *Node) GetDebug() bool {
	return node.debugMode
}

// SetDebug : Sets the debug mode status of this node
func (node *Node) SetDebug(mode bool) {
	node.debugMode = mode
}
