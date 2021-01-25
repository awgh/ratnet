package fs

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/ratnet/api"
	"github.com/awgh/ratnet/api/chunking"
	"github.com/awgh/ratnet/api/events"
)

// CID : Return content key
func (node *Node) CID() (bc.PubKey, error) {
	return node.contentKey.GetPubKey(), nil
}

// GetContact : Return a Contact by name
func (node *Node) GetContact(name string) (*api.Contact, error) {
	node.mutex.RLock()
	defer node.mutex.RUnlock()
	c, ok := node.contacts[name]
	if !ok {
		return nil, errors.New("Contact not found")
	}
	r := new(api.Contact)
	r.Name = name
	r.Pubkey = c.Pubkey
	return r, nil
}

// GetContacts : Return a list of Contacts
func (node *Node) GetContacts() ([]api.Contact, error) {
	node.mutex.RLock()
	defer node.mutex.RUnlock()

	var contacts []api.Contact
	for _, v := range node.contacts {
		contacts = append(contacts, api.Contact{Name: v.Name, Pubkey: v.Pubkey})
	}
	return contacts, nil
}

// AddContact : Add or Update a contact key to this node's database
func (node *Node) AddContact(name string, key string) error {
	node.mutex.Lock()
	defer node.mutex.Unlock()
	if !node.contentKey.ValidatePubKey(key) {
		return errors.New("Invalid Public Key in AddContact")
	}
	c := new(api.Contact)
	c.Name = name
	c.Pubkey = key
	node.contacts[name] = c
	return nil
}

// DeleteContact : Remove a contact from this node's database
func (node *Node) DeleteContact(name string) error {
	node.mutex.Lock()
	defer node.mutex.Unlock()
	if _, ok := node.contacts[name]; !ok {
		return errors.New("Contact not found")
	}
	delete(node.contacts, name)
	return nil
}

// GetChannel : Return a channel by name
func (node *Node) GetChannel(name string) (*api.Channel, error) {
	node.mutex.RLock()
	defer node.mutex.RUnlock()
	channel, ok := node.channels[name]
	if !ok {
		return nil, errors.New("Channel not found")
	}
	c := new(api.Channel)
	c.Name = name
	c.Pubkey = channel.Pubkey
	return c, nil
}

// GetChannels : Return list of channels known to this node
func (node *Node) GetChannels() ([]api.Channel, error) {
	node.mutex.RLock()
	defer node.mutex.RUnlock()
	var channels []api.Channel
	for _, v := range node.channels {
		channels = append(channels, api.Channel{
			Name: v.Name, Pubkey: v.Pubkey,
		})
	}
	return channels, nil
}

// AddChannel : Add a channel to this node's database
func (node *Node) AddChannel(name string, privkey string) error {
	node.mutex.Lock()
	defer node.mutex.Unlock()
	pk := node.contentKey.Clone()
	if err := pk.FromB64(privkey); err != nil {
		return errors.New("Invalid channel key")
	}
	c := new(api.ChannelPriv)
	c.Name = name
	c.Pubkey = pk.GetPubKey().ToB64()
	c.Privkey = pk
	node.channels[name] = c
	return nil
}

// DeleteChannel : Remove a channel from this node's database
func (node *Node) DeleteChannel(name string) error {
	node.mutex.Lock()
	defer node.mutex.Unlock()
	if _, ok := node.channels[name]; !ok {
		return errors.New("Channel not found")
	}
	delete(node.channels, name)
	return nil
}

// GetProfile : Retrieve a profile by name
func (node *Node) GetProfile(name string) (*api.Profile, error) {
	node.mutex.RLock()
	defer node.mutex.RUnlock()
	p, ok := node.profiles[name]
	if !ok {
		return nil, errors.New("Profile not found")
	}
	pub := new(api.Profile)
	pub.Name = name
	pub.Enabled = p.Enabled
	kp := p.Privkey
	pk := kp.GetPubKey()
	pub.Pubkey = pk.ToB64()
	return pub, nil
}

// GetProfiles : Retrieve the list of profiles for this node
func (node *Node) GetProfiles() ([]api.Profile, error) {
	node.mutex.RLock()
	defer node.mutex.RUnlock()
	var profiles []api.Profile
	for name, profile := range node.profiles {
		profiles = append(profiles, api.Profile{
			Name:    name,
			Enabled: profile.Enabled,
			Pubkey:  profile.Privkey.GetPubKey().ToB64(),
		})
	}
	return profiles, nil
}

// AddProfile : Add or Update a profile to this node's database
func (node *Node) AddProfile(name string, enabled bool) error {
	node.mutex.Lock()
	defer node.mutex.Unlock()
	// generate new profile keypair
	profileKey := node.contentKey.Clone()
	if _, ok := node.profiles[name]; !ok {
		profileKey.GenerateKey()
	}
	// insert new profile, or stomp old one by name
	mp := new(api.ProfilePriv)
	mp.Enabled = enabled
	mp.Name = name
	mp.Privkey = profileKey
	node.profiles[name] = mp
	return nil
}

// DeleteProfile : Remove a profile from this node's database
func (node *Node) DeleteProfile(name string) error {
	node.mutex.Lock()
	defer node.mutex.Unlock()
	if _, ok := node.profiles[name]; !ok {
		return errors.New("Profile not found")
	}
	delete(node.profiles, name)
	return nil
}

// LoadProfile : Load a profile key from the database as the content key
func (node *Node) LoadProfile(name string) (bc.PubKey, error) {
	node.mutex.Lock()
	defer node.mutex.Unlock()
	if _, ok := node.channels[name]; !ok {
		return nil, errors.New("Profile not found")
	}
	node.contentKey = node.profiles[name].Privkey
	events.Debug(node, "Profile Loaded: "+node.contentKey.GetPubKey().ToB64())
	return node.contentKey.GetPubKey(), nil
}

// GetPeer : Retrieve a peer from this node's database
func (node *Node) GetPeer(name string) (*api.Peer, error) {
	node.mutex.RLock()
	defer node.mutex.RUnlock()
	peer, ok := node.peers[name]
	if !ok {
		return nil, errors.New("Peer not found")
	}
	p := new(api.Peer)
	p.Name = name
	p.Enabled = peer.Enabled
	p.URI = peer.URI
	p.Group = peer.Group
	return p, nil
}

// GetPeers : Retrieve a list of peers in this node's database
func (node *Node) GetPeers(group ...string) ([]api.Peer, error) {
	node.mutex.RLock()
	defer node.mutex.RUnlock()
	// if we don't have a specified group, it's ""
	groupName := ""
	if len(group) > 0 {
		groupName = group[0]
	}
	var peers []api.Peer
	for _, v := range node.peers {
		if v.Group == groupName {
			peers = append(peers, api.Peer{Name: v.Name, Enabled: v.Enabled, URI: v.URI, Group: groupName})
		}
	}
	return peers, nil
}

// AddPeer : Add or Update a peer configuration
func (node *Node) AddPeer(name string, enabled bool, uri string, group ...string) error {
	node.mutex.Lock()
	defer node.mutex.Unlock()
	// if we don't have a specified group, it's ""
	groupName := ""
	if len(group) > 0 {
		groupName = group[0]
	}
	peer := new(api.Peer)
	peer.Name = name
	peer.Enabled = enabled
	peer.URI = uri
	peer.Group = groupName
	node.peers[name] = peer
	return nil
}

// DeletePeer : Remove a peer from this node's database
func (node *Node) DeletePeer(name string) error {
	node.mutex.Lock()
	defer node.mutex.Unlock()
	if _, ok := node.peers[name]; !ok {
		return errors.New("Peer not found")
	}
	delete(node.peers, name)
	return nil
}

// Send : Transmit a message to a single key
func (node *Node) Send(contactName string, data []byte, pubkey ...bc.PubKey) error {
	var destkey bc.PubKey
	if pubkey != nil && len(pubkey) > 0 && pubkey[0] != nil { // third argument is optional pubkey override
		destkey = pubkey[0]
	} else {
		node.mutex.RLock()
		if _, ok := node.contacts[contactName]; !ok {
			node.mutex.RUnlock()
			return errors.New("Contact not found")
		}
		destkey = node.contentKey.GetPubKey().Clone()
		if err := destkey.FromB64(node.contacts[contactName].Pubkey); err != nil {
			node.mutex.RUnlock()
			return err
		}
		node.mutex.RUnlock()
	}

	return node.SendMsg(api.Msg{Name: contactName, Content: bytes.NewBuffer(data), IsChan: false, PubKey: destkey, Chunked: false})
}

// SendChannelBulk : Transmit messages to a channel
func (node *Node) SendChannelBulk(channelName string, data [][]byte, pubkey ...bc.PubKey) error {
	for i := range data {
		time.Sleep(10 * time.Millisecond)
		if err := node.SendChannel(channelName, data[i], pubkey...); err != nil {
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

// SendChannel : Transmit a message to a channel
func (node *Node) SendChannel(channelName string, data []byte, pubkey ...bc.PubKey) error {
	var destkey bc.PubKey

	if pubkey != nil && len(pubkey) > 0 && pubkey[0] != nil { // third argument is optional PubKey override
		destkey = pubkey[0]
	} else {
		node.mutex.RLock()
		c, ok := node.channels[channelName]
		node.mutex.RUnlock()
		if !ok {
			return errors.New("No public key for Channel")
		}
		destkey = c.Privkey.GetPubKey()
	}

	return node.SendMsg(api.Msg{Name: channelName, Content: bytes.NewBuffer(data), IsChan: true, PubKey: destkey, Chunked: false})
}

// SendMsg : Transmits a message
func (node *Node) SendMsg(msg api.Msg) error {
	// determine if we need to chunk
	chunkSize := chunking.ChunkSize(node)                               // finds the minimum transport byte limit
	if msg.Content.Len() > 0 && uint32(msg.Content.Len()) > chunkSize { // we need to chunk
		if msg.Chunked { // we're already chunked, freak out!
			return errors.New("Chunked message needs to be chunked, bailing out")
		}
		return chunking.SendChunked(node, chunkSize, msg)
	}

	data, err := node.contentKey.EncryptMessage(msg.Content.Bytes(), msg.PubKey)
	if err != nil {
		return err
	}

	flags := uint8(0)
	if msg.IsChan {
		flags |= api.ChannelFlag
	}
	if msg.Chunked {
		flags |= api.ChunkedFlag
	}
	if msg.StreamHeader {
		flags |= api.StreamHeaderFlag
	}
	rxsum := []byte{flags} // prepend flags byte

	path := node.basePath

	if msg.IsChan {
		// prepend a uint16 of channel name length, little-endian
		t := uint16(len(msg.Name))
		rxsum = append(rxsum, byte(t>>8), byte(t&0xFF))
		rxsum = append(rxsum, []byte(msg.Name)...)

		// create channel dir if not exist
		path = filepath.Join(path, msg.Name)
		os.Mkdir(path, os.FileMode(int(0700)))
	}
	data = append(rxsum, data...)

	f, err := os.Create(filepath.Join(path, hex(node.outboxIndex)))
	if err != nil {
		return err
	}
	node.outboxIndex++
	defer f.Close()
	w := bufio.NewWriter(f)
	w.Write(data)
	w.Flush()
	return nil
}

// Start : starts the Connection Policy threads
func (node *Node) Start() error {
	// do not start again if the node is already running
	if node.IsRunning() {
		return nil
	}

	// init crypto keys
	node.contentKey.GenerateKey()
	node.routingKey.GenerateKey()

	node.setIsRunning(true)

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
			if !node.IsRunning() {
				break
			}

			// read message off the input channel
			message := <-node.In()
			events.Debug(node, "Message accepted on input channel")
			if err := node.SendMsg(message); err != nil {
				events.Error(node, err.Error())
			}
		}
	}()

	// dechunking loop
	go func() {
		for {
			time.Sleep(10 * time.Millisecond)
			// check if we should stop running
			if !node.IsRunning() {
				break
			}
			// for each stream, count chunks for that header
			for _, stream := range node.streams {
				if stream != nil {
					count := len(node.chunks[stream.StreamID])
					// if chunks == total chunks, re-assemble Msg and call Handle
					if uint32(count) == uint32(stream.NumChunks) {
						buf := bytes.NewBuffer([]byte{})
						for i := uint32(0); i < stream.NumChunks; i++ {
							chunk, ok := node.chunks[stream.StreamID][i]
							if !ok {
								events.Critical(node, "Chunk count miscalculated - code broken")
							}
							buf.Write(chunk.Data)
						}

						var msg api.Msg
						if len(stream.ChannelName) > 0 {
							msg.IsChan = true
							msg.Name = stream.ChannelName
						}
						msg.Content = buf

						select {
						case node.Out() <- msg:
							events.Debug(node, "Sent message "+fmt.Sprint(msg.Content.Bytes()))
							node.streams[stream.StreamID] = nil
							node.chunks[stream.StreamID] = make(map[uint32]*api.Chunk)
						default:
							events.Debug(node, "No message sent")
						}
					}
				}
			}
		}
	}()

	return nil
}

// Stop : sets the isRunning flag to false, indicating that all go routines should end
func (node *Node) Stop() {
	for _, policy := range node.policies {
		policy.Stop()
	}
	node.setIsRunning(false)
}
