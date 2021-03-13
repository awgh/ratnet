package api

import (
	"github.com/awgh/bencrypt/bc"
)

// Node : abstract base type for RatNet implementations
type Node interface {

	// Local Access Only (Not Exposed Through RPC API)
	Start() error
	Stop()
	GetPolicies() []Policy
	SetPolicy(policies ...Policy)
	Router() Router
	SetRouter(router Router)
	GetChannelPrivKey(name string) (string, error)
	Handle(msg Msg) (bool, error)
	Forward(msg Msg) error
	IsRunning() bool

	// Chunking
	// AddStream - inform node of receipt of a stream header
	AddStream(streamID uint32, totalChunks uint32, channelName string) error
	// AddChunk - inform node of receipt of a chunk
	AddChunk(streamID uint32, chunkNum uint32, data []byte) error

	// FlushOutbox : Empties the outbox of messages older than maxAgeSeconds
	FlushOutbox(maxAgeSeconds int64)

	// RPC Entrypoints

	// AdminRPC :
	AdminRPC(transport Transport, call RemoteCall) (interface{}, error)

	// PublicRPC :
	PublicRPC(transport Transport, call RemoteCall) (interface{}, error)

	// PUBLIC API
	// Functions that are safe for non-authenticated calls / open Internet

	// ID : get the routing public key (1)
	ID() (bc.PubKey, error)

	// Dropoff : Deliver a batch of messages to this node (2)
	Dropoff(bundle Bundle) error

	// Pickup : Get outgoing messages from this node (3)
	Pickup(routingPub bc.PubKey, lastTime int64, maxBytes int64, channelNames ...string) (Bundle, error)

	//

	// Admin API Functions
	// Functions that are NOT SAFE for non-authenticated access from the Internet

	// CID : Return content key (16)
	CID() (bc.PubKey, error)

	// GetContact : Return a contact by name (17)
	GetContact(name string) (*Contact, error)
	// GetContacts : Return a list of contacts (18)
	GetContacts() ([]Contact, error)
	// AddContact : Add or Update a contact key (19)
	AddContact(name string, key string) error
	// DeleteContact : Remove a contact (20)
	DeleteContact(name string) error

	// GetChannel : Return a channel by name (21)
	GetChannel(name string) (*Channel, error)
	// GetChannels : Return list of channels known to this node (22)
	GetChannels() ([]Channel, error)
	// AddChannel : Add a channel to this node's database (23)
	AddChannel(name string, privkey string) error
	// DeleteChannel : Remove a channel from this node's database (24)
	DeleteChannel(name string) error

	// GetProfile : Retrieve a Profile by name (25)
	GetProfile(name string) (*Profile, error)
	// GetProfiles : Retrieve the list of profiles for this node (26)
	GetProfiles() ([]Profile, error)
	// AddProfile : Add or Update a profile to this node's database (27)
	AddProfile(name string, enabled bool) error
	// DeleteProfile : Remove a profile from this node's database (28)
	DeleteProfile(name string) error
	// LoadProfile : Load a profile key from the database as the content key (29)
	LoadProfile(name string) (bc.PubKey, error)

	// GetPeer : Retrieve a peer by name (30)
	GetPeer(name string) (*Peer, error)
	// GetPeers : Retrieve this node's list of peers (31)
	GetPeers(group ...string) ([]Peer, error)
	// AddPeer : Add or Update a peer configuration (32)
	AddPeer(name string, enabled bool, uri string, group ...string) error
	// DeletePeer : Remove a peer from this node's database (33)
	DeletePeer(name string) error

	// Send : Transmit a message to a single key (34) <deprecated>
	Send(contactName string, data []byte, pubkey ...bc.PubKey) error
	// SendChannel : Transmit a message to a channel (35) <deprecated>
	SendChannel(channelName string, data []byte, pubkey ...bc.PubKey) error

	// SendMsg : Transmit a message object (36)
	SendMsg(msg Msg) error

	//  End of Admin API Functions

	//

	// Channels
	// In : Returns the In channel of this node
	In() chan Msg
	// Out : Returns the Out channel of this node
	Out() chan Msg
	// Events : Returns the Err channel of this node
	Events() chan Event

	ImportExport
}

// Contact : object that describes a contact (named public key)
type Contact struct {
	Name   string `db:"name"`
	Pubkey string `db:"pubkey"`
}

// Channel : object that describes a channel (named public key)
type Channel struct {
	Name   string `db:"name"`
	Pubkey string `db:"pubkey"`
}

// ChannelPriv : object that describes a channel (including private key)
type ChannelPriv struct {
	Name    string
	Pubkey  string
	Privkey bc.KeyPair
}

// ChannelPrivB64 : object that describes a channel, database version (including private key)
type ChannelPrivB64 struct {
	Name    string `db:"name"`
	Privkey string `db:"privkey"`
}

// Profile : object that describes a profile
type Profile struct {
	Name    string `db:"name"`
	Enabled bool   `db:"enabled"`
	Pubkey  string `db:"pubkey"`
}

// ProfilePriv : object that describes a profile (including private key)
type ProfilePriv struct {
	Name    string
	Enabled bool
	Privkey bc.KeyPair
}

// ProfilePrivB64 : object that describes a profile, database version (including private key)
type ProfilePrivB64 struct {
	Name    string `db:"name"`
	Enabled bool   `db:"enabled"`
	Privkey string `db:"privkey"`
}

// Peer : object that describes a peer (transport connection instructions)
type Peer struct {
	Name    string `db:"name"`
	Enabled bool   `db:"enabled"`
	URI     string `db:"uri"`
	Group   string `db:"peergroup"`
}

// Bundle : mostly-opaque data blob returned by Pickup and passed into Dropoff
type Bundle struct {
	Data []byte
	Time int64
}

// OutboxMsg : object that describes an outbox message
type OutboxMsg struct {
	Channel   string `db:"channel"`
	Msg       []byte `db:"msg"`
	Timestamp int64  `db:"timestamp"`
}

// ConfigValue - Name/Value pairs of configuration strings
type ConfigValue struct {
	Name  string `db:"name"`
	Value string `db:"value"`
}
