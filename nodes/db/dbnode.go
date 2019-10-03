package db

// To install upper db:
// go get -v -u upper.io/db.v3
// FOR POSTGRES DRIVER: go get -v -u github.com/lib/pq

import (
	"sync"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/ratnet/api"
	"github.com/awgh/ratnet/nodes"
	"github.com/awgh/ratnet/router"
	"upper.io/db.v3/lib/sqlbuilder"
)

// Node : defines an instance of the API with a ql-DB backed Node
type Node struct {
	contentKey  bc.KeyPair
	routingKey  bc.KeyPair
	channelKeys map[string]bc.KeyPair

	policies []api.Policy
	router   api.Router

	db sqlbuilder.Database

	mutex *sync.Mutex

	isRunning bool

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
	node.out = make(chan api.Msg)
	node.events = make(chan api.Event)

	// setup default router
	node.router = router.NewDefaultRouter()

	return node
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
