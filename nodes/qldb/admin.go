package qldb

import (
	"errors"
	"log"
	"time"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/ratnet/api"
)

// CID : Return content key
func (node *Node) CID() (bc.PubKey, error) {
	return node.contentKey.GetPubKey(), nil
}

// GetContact : Return a list of  keys
func (node *Node) GetContact(name string) (*api.Contact, error) {
	pubs, err := node.qlGetContactPubKey(name)
	if err != nil {
		return nil, err
	} else if pubs == "" {
		return nil, nil
	}
	contact := new(api.Contact)
	contact.Name = name
	contact.Pubkey = pubs
	return contact, nil
}

// GetContacts : Return a list of  keys
func (node *Node) GetContacts() ([]api.Contact, error) {
	return node.qlGetContacts()
}

// AddContact : Add or Update a contact key to this node's database
func (node *Node) AddContact(name string, key string) error {
	return node.qlAddContact(name, key)
}

// DeleteContact : Remove a contact from this node's database
func (node *Node) DeleteContact(name string) error {
	node.qlDeleteContact(name)
	return nil
}

// GetChannel : Return a channel by name
func (node *Node) GetChannel(name string) (*api.Channel, error) {
	privkey, err := node.qlGetChannelPrivKey(name)
	if err != nil {
		return nil, err
	}
	prv := node.contentKey.Clone()
	if err := prv.FromB64(privkey); err != nil {
		return nil, err
	}
	channel := new(api.Channel)
	channel.Name = name
	channel.Pubkey = prv.GetPubKey().ToB64()
	return channel, nil
}

// GetChannels : Return list of channels known to this node
func (node *Node) GetChannels() ([]api.Channel, error) {
	return node.qlGetChannels()
}

// AddChannel : Add a channel to this node's database
func (node *Node) AddChannel(name string, privkey string) error {
	if err := node.qlAddChannel(name, privkey); err != nil {
		return err
	}
	node.refreshChannels()
	return nil
}

// DeleteChannel : Remove a channel from this node's database
func (node *Node) DeleteChannel(name string) error {
	node.qlDeleteChannel(name)
	return nil
}

// GetProfile : Retrieve a profiles
func (node *Node) GetProfile(name string) (*api.Profile, error) {
	return node.qlGetProfile(name)
}

// GetProfiles : Retrieve the list of profiles for this node
func (node *Node) GetProfiles() ([]api.Profile, error) {
	return node.qlGetProfiles()
}

// AddProfile : Add or Update a profile to this node's database
func (node *Node) AddProfile(name string, enabled bool) error {
	return node.qlAddProfile(name, enabled)
}

// DeleteProfile : Remove a profile from this node's database
func (node *Node) DeleteProfile(name string) error {
	node.qlDeleteProfile(name)
	return nil
}

// LoadProfile : Load a profile key from the database as the content key
func (node *Node) LoadProfile(name string) (bc.PubKey, error) {
	pk := node.qlGetProfilePrivateKey(name)
	profileKey := node.contentKey.Clone()
	if err := profileKey.FromB64(pk); err != nil {
		node.errMsg(err, false)
		return nil, err
	}
	node.contentKey = profileKey
	node.debugMsg("Profile Loaded: " + profileKey.GetPubKey().ToB64())
	return profileKey.GetPubKey(), nil
}

// GetPeer : Retrieve a peer by name
func (node *Node) GetPeer(name string) (*api.Peer, error) {
	return node.qlGetPeer(name)
}

// GetPeers : Retrieve a list of peers in this node's database
func (node *Node) GetPeers() ([]api.Peer, error) {
	return node.qlGetPeers()
}

// AddPeer : Add or Update a peer configuration
func (node *Node) AddPeer(name string, enabled bool, uri string) error {
	return node.qlAddPeer(name, enabled, uri)
}

// DeletePeer : Remove a peer from this node's database
func (node *Node) DeletePeer(name string) error {
	node.qlDeletePeer(name)
	return nil
}

// Send : Transmit a message to a single key
func (node *Node) Send(contactName string, data []byte, pubkey ...bc.PubKey) error {
	var destkey bc.PubKey
	if pubkey != nil && len(pubkey) > 0 && pubkey[0] != nil { // third argument is optional pubkey override
		destkey = pubkey[0]
	} else {
		s, err := node.qlGetContactPubKey(contactName)
		if err != nil {
			return err
		} else if s == "" {
			return errors.New("Unknown Contact")
		}
		destkey = node.contentKey.GetPubKey().Clone()
		if err := destkey.FromB64(s); err != nil {
			return err
		}
	}

	return node.send("", destkey, data)
}

// SendChannel : Transmit a message to a channel
func (node *Node) SendChannel(channelName string, data []byte, pubkey ...bc.PubKey) error {
	var destkey bc.PubKey

	if pubkey != nil && len(pubkey) > 0 && pubkey[0] != nil { // third argument is optional PubKey override
		destkey = pubkey[0]
	} else {
		key, ok := node.channelKeys[channelName]
		if !ok {
			return errors.New("No public key for Channel")
		}
		destkey = key.GetPubKey()
	}

	if destkey == nil {
		log.Fatal("nil DestKey in SendChannel")
	}

	return node.send(channelName, destkey, data)
}

func (node *Node) send(channelName string, destkey bc.PubKey, msg []byte) error {

	data, err := node.contentKey.EncryptMessage(msg, destkey)
	if err != nil {
		return err
	}

	// prepend a uint16 of channel name length, little-endian
	t := uint16(len(channelName))
	rxsum := []byte{byte(t >> 8), byte(t & 0xFF)}
	rxsum = append(rxsum, []byte(channelName)...)
	data = append(rxsum, data...)

	ts := time.Now().UnixNano()
	err = node.qlOutboxEnqueue(channelName, data, ts, false) // todo: not checking if exists here?
	if err != nil {
		return err
	}
	return nil
}

// SendBulk : Transmit messages to a single key
func (node *Node) SendBulk(contactName string, data [][]byte, pubkey ...bc.PubKey) error {
	var destkey bc.PubKey

	if pubkey != nil && len(pubkey) > 0 && pubkey[0] != nil { // third argument is optional pubkey override
		destkey = pubkey[0]
	} else {
		s, err := node.qlGetContactPubKey(contactName)
		if err != nil {
			return err
		}
		if s == "" {
			return errors.New("Unknown Contact")
		}
		destkey = node.contentKey.GetPubKey().Clone()
		if err := destkey.FromB64(s); err != nil {
			return err
		}
	}

	return node.sendBulk("", destkey, data)
}

// SendChannelBulk : Transmit messages to a channel
func (node *Node) SendChannelBulk(channelName string, data [][]byte, pubkey ...bc.PubKey) error {
	var destkey bc.PubKey

	if pubkey != nil && len(pubkey) > 0 && pubkey[0] != nil { // third argument is optional PubKey override
		destkey = pubkey[0]
	} else {
		key, ok := node.channelKeys[channelName]
		if !ok {
			return errors.New("No public key for Channel")
		}
		destkey = key.GetPubKey()
	}

	if destkey == nil {
		log.Fatal("nil DestKey in SendChannel")
	}

	return node.sendBulk(channelName, destkey, data)
}

func (node *Node) sendBulk(channelName string, destkey bc.PubKey, msg [][]byte) error {

	// prepend a uint16 of channel name length, little-endian
	t := uint16(len(channelName))
	rxsum := []byte{byte(t >> 8), byte(t & 0xFF)}
	rxsum = append(rxsum, []byte(channelName)...)

	//todo: is this passing msg by reference or not???
	data := make([][]byte, len(msg))
	for i := range msg {
		var err error
		data[i], err = node.contentKey.EncryptMessage(msg[i], destkey)
		if err != nil {
			return err
		}
		data[i] = append(rxsum, data[i]...)
	}
	ts := time.Now().UnixNano()
	node.outboxBulkInsert(channelName, ts, data)
	return nil
}

// Start : starts the Connection Policy threads
func (node *Node) Start() error {
	// do not start again if the node is already running
	if node.isRunning {
		return nil
	}
	node.isRunning = true

	// start the signal monitor
	node.signalMonitor()

	// start the policies
	if node.policies != nil {
		for i := 0; i < len(node.policies); i++ {
			if err := node.policies[i].RunPolicy(); err != nil {
				return err
			}
		}
	}

	// input loop
	go func() {
		for {
			// check if we should stop running
			if !node.isRunning {
				break
			}

			// read message off the input channel
			message := <-node.In()

			switch message.IsChan {
			case true:
				if err := node.SendChannel(message.Name, message.Content.Bytes(), message.PubKey); err != nil {
					log.Fatal(err)
				}

			case false:
				if err := node.Send(message.Name, message.Content.Bytes(), message.PubKey); err != nil {
					log.Fatal(err)
				}
			}
		}
	}()

	return nil
}

// Stop : sets the isRunning flag to false, indicating that all go routines should end
func (node *Node) Stop() {
	node.isRunning = false
	for _, policy := range node.policies {
		policy.Stop()
	}
}
