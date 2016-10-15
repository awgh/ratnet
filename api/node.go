package api

import "github.com/awgh/bencrypt/bc"

// Node : abstract base type for RatNet implementations
type Node interface {

	// Local Access Only (Not Exposed Through RPC API)
	Start() error
	Stop()
	SetPolicy(policies ...Policy)
	SetRouter(router Router)
	GetChannelPrivKey(name string) (string, error)
	Handle(channelName string, message []byte) error
	Forward(channelName string, message []byte) error

	// FlushOutbox : Empties the outbox of messages older than maxAgeSeconds
	FlushOutbox(maxAgeSeconds int64)

	// RPC Entrypoints

	// AdminRPC :
	AdminRPC(method string, args ...string) (string, error)

	// PublicRPC :
	PublicRPC(method string, args ...string) (string, error)

	// PUBLIC API
	// Functions that are safe for non-authenticated calls / open Internet

	// ID : get the routing public key
	ID() (bc.PubKey, error)

	// Dropoff : Deliver a batch of messages to this node
	Dropoff(bundle Bundle) error

	// Pickup : Get outgoing messages from this node
	Pickup(routingPub bc.PubKey, lastTime int64, channelNames ...string) (Bundle, error)

	//

	// Admin API Functions
	// Functions that are NOT SAFE for non-authenticated access from the Internet

	// CID : Return content key
	CID() (bc.PubKey, error)

	// GetContact : Return a contact by name
	GetContact(name string) (*Contact, error)
	// GetContacts : Return a list of contacts
	GetContacts() ([]Contact, error)
	// AddContact : Add or Update a contact key
	AddContact(name string, key string) error
	// DeleteContact : Remove a contact
	DeleteContact(name string) error

	// GetChannel : Return a channel by name
	GetChannel(name string) (*Channel, error)
	// GetChannels : Return list of channels known to this node
	GetChannels() ([]Channel, error)
	// AddChannel : Add a channel to this node's database
	AddChannel(name string, privkey string) error
	// DeleteChannel : Remove a channel from this node's database
	DeleteChannel(name string) error

	// GetProfile : Retrieve a Profile by name
	GetProfile(name string) (*Profile, error)
	// GetProfiles : Retrieve the list of profiles for this node
	GetProfiles() ([]Profile, error)
	// AddProfile : Add or Update a profile to this node's database
	AddProfile(name string, enabled bool) error
	// DeleteProfile : Remove a profile from this node's database
	DeleteProfile(name string) error
	// LoadProfile : Load a profile key from the database as the content key
	LoadProfile(name string) (bc.PubKey, error)

	// GetPeer : Retrieve a peer by name
	GetPeer(name string) (*Peer, error)
	// GetPeers : Retrieve this node's list of peers
	GetPeers() ([]Peer, error)
	// AddPeer : Add or Update a peer configuration
	AddPeer(name string, enabled bool, uri string) error
	// DeletePeer : Remove a peer from this node's database
	DeletePeer(name string) error

	// Send : Transmit a message to a single key
	Send(contactName string, data []byte, pubkey ...bc.PubKey) error
	// SendChannel : Transmit a message to a channel
	SendChannel(channelName string, data []byte, pubkey ...bc.PubKey) error

	//  End of Admin API Functions

	//

	// Channels
	// In : Returns the In channel of this node
	In() chan Msg
	// Out : Returns the In channel of this node
	Out() chan Msg
	// Err : Returns the In channel of this node
	Err() chan Msg

	// Debug
	GetDebug() bool
	SetDebug(mode bool)
}

// Contact : object that describes a contact (named public key)
type Contact struct {
	Name   string
	Pubkey string
}

// Channel : object that describes a channel (named public key)
type Channel struct {
	Name   string
	Pubkey string
}

// ChannelPriv : object that describes a channel (including private key)
type ChannelPriv struct {
	Name    string
	Pubkey  string
	Privkey bc.KeyPair
}

// Profile : object that describes a profile
type Profile struct {
	Name    string
	Enabled bool
	Pubkey  string
}

// ProfilePriv : object that describes a profile (including private key)
type ProfilePriv struct {
	Name    string
	Enabled bool
	Privkey bc.KeyPair
}

// Peer : object that describes a peer (transport connection instructions)
type Peer struct {
	Name    string
	URI     string
	Enabled bool
}

// Bundle : mostly-opaque data blob returned by Pickup and passed into Dropoff
type Bundle struct {
	Data []byte
	Time int64
}
