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
	c := node.db()
	r := transactQueryRow(c, "SELECT cpubkey FROM contacts WHERE name==$1;", name)
	var pubs string
	if err := r.Scan(&pubs); err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	contact := new(api.Contact)
	contact.Name = name
	contact.Pubkey = pubs
	return contact, nil
}

// GetContacts : Return a list of  keys
func (node *Node) GetContacts() ([]api.Contact, error) {
	c := node.db()
	s := transactQuery(c, "SELECT name,cpubkey FROM contacts;")
	var contacts []api.Contact
	for s.Next() {
		var d api.Contact
		if err := s.Scan(&d.Name, &d.Pubkey); err != nil {
			return nil, err
		}
		contacts = append(contacts, d)
	}
	return contacts, nil
}

// AddContact : Add or Update a contact key to this node's database
func (node *Node) AddContact(name string, key string) error {
	c := node.db()
	if !node.contentKey.ValidatePubKey(key) {
		return errors.New("Invalid Public Key in AddContact")
	}
	if tx, err := c.Begin(); err != nil {
		node.errMsg(err, true)
	} else {
		_, _ = tx.Exec("DELETE FROM contacts WHERE name==$1;", name)
		_, err = tx.Exec("INSERT INTO contacts VALUES( $1, $2 )", name, key)
		if err != nil {
			node.errMsg(err, true)
		}
		err = tx.Commit()
		if err != nil {
			node.errMsg(err, true)
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
	c := node.db()
	r := transactQueryRow(c, "SELECT privkey FROM channels WHERE name==$1;", name)
	var privkey string
	if err := r.Scan(&privkey); err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else {
		prv := node.contentKey.Clone()
		if err := prv.FromB64(privkey); err != nil {
			return nil, err
		}
		channel := new(api.Channel)
		channel.Name = name
		channel.Pubkey = prv.GetPubKey().ToB64()
		return channel, nil
	}
}

// GetChannels : Return list of channels known to this node
func (node *Node) GetChannels() ([]api.Channel, error) {
	c := node.db()
	r := transactQuery(c, "SELECT name,privkey FROM channels;")
	var channels []api.Channel
	for r.Next() {
		var n, p string
		if err := r.Scan(&n, &p); err != nil {
			return nil, err
		}
		prv := node.contentKey.Clone()
		if err := prv.FromB64(p); err != nil {
			return nil, err
		}
		channels = append(channels, api.Channel{Name: n, Pubkey: prv.GetPubKey().ToB64()})
	}
	return channels, nil
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
	node.refreshChannels(c)
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
	c := node.db()
	r := transactQueryRow(c, "SELECT enabled,privkey FROM profiles WHERE name==$1;", name)
	var e bool
	var prv string
	if err := r.Scan(&e, &prv); err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	profile := new(api.Profile)
	profile.Enabled = e
	profile.Name = name
	pk := node.contentKey.Clone()
	if err := pk.FromB64(prv); err != nil {
		return nil, err
	}
	profile.Pubkey = pk.GetPubKey().ToB64()
	return profile, nil
}

// GetProfiles : Retrieve the list of profiles for this node
func (node *Node) GetProfiles() ([]api.Profile, error) {
	c := node.db()
	r := transactQuery(c, "SELECT name,enabled,privkey FROM profiles;")
	var profiles []api.Profile
	for r.Next() {
		var p api.Profile
		var prv string
		if err := r.Scan(&p.Name, &p.Enabled, &prv); err != nil {
			return nil, err
		}
		pk := node.contentKey.Clone()
		if err := pk.FromB64(prv); err != nil {
			return nil, err
		}
		p.Pubkey = pk.GetPubKey().ToB64()
		profiles = append(profiles, p)
	}
	return profiles, nil
}

// AddProfile : Add or Update a profile to this node's database
func (node *Node) AddProfile(name string, enabled bool) error {
	c := node.db()
	r := transactQueryRow(c, "SELECT * FROM profiles WHERE name==$1;", name)
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
	row := transactQueryRow(c, "SELECT privkey FROM profiles WHERE name==$1;", name)
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
	c := node.db()
	r := transactQueryRow(c, "SELECT uri,enabled FROM peers WHERE name==$1;", name)
	var u string
	var e bool
	if err := r.Scan(&u, &e); err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	peer := new(api.Peer)
	peer.Name = name
	peer.Enabled = e
	peer.URI = u
	return peer, nil
}

// GetPeers : Retrieve a list of peers in this node's database
func (node *Node) GetPeers() ([]api.Peer, error) {
	c := node.db()
	r := transactQuery(c, "SELECT name,uri,enabled FROM peers;")
	var peers []api.Peer
	for r.Next() {
		var s api.Peer
		if err := r.Scan(&s.Name, &s.URI, &s.Enabled); err != nil {
			return nil, err
		}
		peers = append(peers, s)
	}
	return peers, nil
}

// AddPeer : Add or Update a peer configuration
func (node *Node) AddPeer(name string, enabled bool, uri string) error {
	c := node.db()
	r := transactQueryRow(c, "SELECT name FROM peers WHERE name==$1;", name)
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
	c := node.db()
	var r1 *sql.Row
	var err error

	var destkey bc.PubKey
	if pubkey != nil && len(pubkey) > 0 && pubkey[0] != nil { // third argument is optional pubkey override
		destkey = pubkey[0]
	} else {
		var s string
		r1 = transactQueryRow(c, "SELECT cpubkey FROM contacts WHERE name==$1;", contactName)
		err = r1.Scan(&s)
		if err == sql.ErrNoRows {
			return errors.New("Unknown Contact")
		} else if err != nil {
			return err
		}
		destkey = node.contentKey.GetPubKey().Clone()
		if err := destkey.FromB64(s); err != nil {
			return err
		}
	}

	return node.send("", destkey, data, c)
}

// SendChannel : Transmit a message to a channel
func (node *Node) SendChannel(channelName string, data []byte, pubkey ...bc.PubKey) error {
	c := node.db()
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

	return node.send(channelName, destkey, data, c)
}

func (node *Node) send(channelName string, destkey bc.PubKey, msg []byte, c *sql.DB) error {

	//todo: is this passing msg by reference or not???
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
	//d := base64.StdEncoding.EncodeToString(data)
	transactExec(c, "INSERT INTO outbox(channel, msg, timestamp) VALUES($1,$2, $3);",
		channelName, data, ts)

	return nil
}

// SendBulk : Transmit messages to a single key
func (node *Node) SendBulk(contactName string, data [][]byte, pubkey ...bc.PubKey) error {
	c := node.db()
	var r1 *sql.Row
	var err error

	var destkey bc.PubKey
	if pubkey != nil && len(pubkey) > 0 && pubkey[0] != nil { // third argument is optional pubkey override
		destkey = pubkey[0]
	} else {
		var s string
		r1 = transactQueryRow(c, "SELECT cpubkey FROM contacts WHERE name==$1;", contactName)
		err = r1.Scan(&s)
		if err == sql.ErrNoRows {
			return errors.New("Unknown Contact")
		} else if err != nil {
			return err
		}
		destkey = node.contentKey.GetPubKey().Clone()
		if err := destkey.FromB64(s); err != nil {
			return err
		}
	}

	return node.sendBulk("", destkey, data, c)
}

// SendChannelBulk : Transmit messages to a channel
func (node *Node) SendChannelBulk(channelName string, data [][]byte, pubkey ...bc.PubKey) error {
	c := node.db()
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

	return node.sendBulk(channelName, destkey, data, c)
}

func (node *Node) sendBulk(channelName string, destkey bc.PubKey, msg [][]byte, c *sql.DB) error {

	// prepend a uint16 of channel name length, little-endian
	t := uint16(len(channelName))
	rxsum := []byte{byte(t >> 8), byte(t & 0xFF)}
	rxsum = append(rxsum, []byte(channelName)...)

	//todo: is this passing msg by reference or not???
	data := make([][]byte, len(msg))
	for i, _ := range msg {
		var err error
		data[i], err = node.contentKey.EncryptMessage(msg[i], destkey)
		if err != nil {
			return err
		}
		data[i] = append(rxsum, data[i]...)
	}
	ts := time.Now().UnixNano()
	//d := base64.StdEncoding.EncodeToString(data)
	outboxBulkInsert(c, channelName, ts, data)
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
