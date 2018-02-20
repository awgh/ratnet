package qldb

import (
	"database/sql"
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
	c := node.db()
	if !node.contentKey.ValidatePubKey(key) {
		return errors.New("Invalid Public Key in AddContact")
	}
	if tx, err := c.Begin(); err != nil {
		node.errMsg(err, true)
		return err
	} else {
		_, _ = tx.Exec("DELETE FROM contacts WHERE name==$1;", name)
		_, err = tx.Exec("INSERT INTO contacts VALUES( $1, $2 );", name, key)
		if err != nil {
			node.errMsg(err, true)
			return err
		}
		err = tx.Commit()
		if err != nil {
			node.errMsg(err, true)
			return err
		}
	}
	return nil
}

// DeleteContact : Remove a contact from this node's database
func (node *Node) DeleteContact(name string) error {
	c := node.db()
	transactExec(c, "DELETE FROM contacts WHERE name==$1;", name)
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
	c := node.db()
	// todo: sanity check key via bencrypt
	tx, err := c.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM channels WHERE name==$1;", name); err != nil {
		return err
	}
	if _, err := tx.Exec("INSERT INTO channels VALUES( $1, $2 )", name, privkey); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	node.refreshChannels()
	return nil
}

// DeleteChannel : Remove a channel from this node's database
func (node *Node) DeleteChannel(name string) error {
	c := node.db()
	transactExec(c, "DELETE FROM channels WHERE name==$1;", name)
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
	c := node.db()

	r := c.QueryRow("SELECT * FROM profiles WHERE name==$1;", name)

	var n, key, al string
	if err := r.Scan(&n, &key, &al); err == sql.ErrNoRows {
		// generate new profile keypair
		profileKey := node.contentKey.Clone()
		profileKey.GenerateKey()

		// insert new profile
		transactExec(c, "INSERT INTO profiles VALUES( $1, $2, $3 )",
			name, profileKey.ToB64(), enabled)

	} else if err == nil {
		// update profile
		transactExec(c, "UPDATE profiles SET enabled=$1 WHERE name==$2;",
			enabled, name)
	} else {
		return err
	}
	return nil
}

// DeleteProfile : Remove a profile from this node's database
func (node *Node) DeleteProfile(name string) error {
	c := node.db()
	transactExec(c, "DELETE FROM profiles WHERE name==$1;", name)
	return nil
}

// LoadProfile : Load a profile key from the database as the content key
func (node *Node) LoadProfile(name string) (bc.PubKey, error) {
	c := node.db()
	row := c.QueryRow("SELECT privkey FROM profiles WHERE name==$1;", name)
	var pk string
	if err := row.Scan(&pk); err != nil {
		return nil, err
	}
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
	c := node.db()
	r := c.QueryRow("SELECT name FROM peers WHERE name==$1;", name)
	var n string
	if err := r.Scan(&n); err == sql.ErrNoRows {
		node.debugMsg("New Server")
		transactExec(c, "INSERT INTO peers (name,uri,enabled) VALUES( $1, $2, $3 );",
			name, uri, enabled)
	} else if err == nil {
		node.debugMsg("Update Server")
		transactExec(c, "UPDATE peers SET enabled=$1,uri=$2 WHERE name==$3;",
			enabled, uri, name)
	} else {
		return err
	}
	return nil
}

// DeletePeer : Remove a peer from this node's database
func (node *Node) DeletePeer(name string) error {
	c := node.db()
	transactExec(c, "DELETE FROM peers WHERE name==$1;", name)
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
