package ram

import (
	"sync/atomic"

	"github.com/awgh/bencrypt/ecc"
	"github.com/awgh/debouncer"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/ratnet/api"
	"github.com/awgh/ratnet/nodes"
	"github.com/awgh/ratnet/router"
)

// OutBufferSize - size of the buffer in messages for the Out() channel
var OutBufferSize = 128

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
	outbox   outboxQueue
	peers    map[string]*api.Peer
	profiles map[string]*api.ProfilePriv
	streams  map[uint32]*api.StreamHeader
	chunks   map[uint32]map[uint32]*api.Chunk

	debouncer *debouncer.Debouncer
}

// New : creates a new instance of API
func New(contentKey, routingKey bc.KeyPair) *Node {
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
	node.events = make(chan api.Event)

	// setup default router
	node.router = router.NewDefaultRouter()
	node.outbox.node = node

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

// FlushOutbox : Deletes outbound messages older than maxAgeSeconds seconds
func (node *Node) FlushOutbox(maxAgeSeconds int64) {
	node.outbox.Flush(maxAgeSeconds)
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
