package ram

import (
	"time"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/ratnet/api"
)

type outboxMsg struct {
	channel   string
	msg       string
	timeStamp int64
}

// Node : defines an instance of the API with a ql-DB backed Node
type Node struct {
	contentKey bc.KeyPair
	routingKey bc.KeyPair

	recentPageIdx int
	recentPage1   map[string]byte
	recentPage2   map[string]byte

	policies  []api.Policy
	router    api.Router
	firstRun  bool
	isRunning bool

	debugMode bool

	// external data members
	in  chan api.Msg
	out chan api.Msg
	err chan api.Msg

	// db -> ram replacements
	channels map[string]*api.ChannelPriv
	config   map[string]string
	contacts map[string]*api.Contact
	outbox   []*outboxMsg
	peers    map[string]*api.Peer
	profiles map[string]*api.ProfilePriv
}

// New : creates a new instance of API
func New(contentKey, routingKey bc.KeyPair) *Node {
	// create node
	node := new(Node)

	// init page maps
	node.recentPage1 = make(map[string]byte)
	node.recentPage2 = make(map[string]byte)

	// init assorted other
	node.channels = make(map[string]*api.ChannelPriv)
	node.config = make(map[string]string)
	node.contacts = make(map[string]*api.Contact)
	node.peers = make(map[string]*api.Peer)
	node.profiles = make(map[string]*api.ProfilePriv)

	// set crypto modes
	node.contentKey = contentKey
	node.routingKey = routingKey

	// setup chans
	node.in = make(chan api.Msg)
	node.out = make(chan api.Msg)
	node.err = make(chan api.Msg)

	// setup default router
	node.router = new(api.DefaultRouter)

	return node
}

// SetPolicy : set the array of Policy objects for this Node
func (node *Node) SetPolicy(policies ...api.Policy) {
	node.policies = policies
}

// SetRouter : set the Router object for this Node
func (node *Node) SetRouter(router api.Router) {
	node.router = router
}

// FlushOutbox : Deletes outbound messages older than maxAgeSeconds seconds
func (node *Node) FlushOutbox(maxAgeSeconds int64) {
	c := (time.Now().UnixNano()) - (maxAgeSeconds * 1000000000)
	for index, mail := range node.outbox {
		if mail.timeStamp < c {
			node.outbox = append(node.outbox[:index], node.outbox[index+1:]...)
		}
	}
}

// Channels

// In : Returns the In channel of this node
func (node *Node) In() chan api.Msg {
	return node.in
}

// Out : Returns the In channel of this node
func (node *Node) Out() chan api.Msg {
	return node.out
}

// Err : Returns the In channel of this node
func (node *Node) Err() chan api.Msg {
	return node.err
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
