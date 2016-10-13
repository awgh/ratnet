package qldb

import (
	"database/sql"
	"encoding/base64"
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
		s.Scan(&d.Name, &d.Pubkey)
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
		log.Fatal(err.Error())
	} else {
		_, err = tx.Exec("DELETE FROM contacts WHERE name==$1;", name)
		_, err = tx.Exec("INSERT INTO contacts VALUES( $1, $2 )", name, key)
		if err != nil {
			log.Fatal(err.Error())
		}
		tx.Commit()
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
		r.Scan(&n, &p)
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
	}
	if _, err := tx.Exec("INSERT INTO channels VALUES( $1, $2 )", name, privkey); err != nil {
		return err
	}
	tx.Commit()
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
		r.Scan(&p.Name, &p.Enabled, &prv)
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
	row.Scan(&pk)
	profileKey := node.contentKey.Clone()
	if err := profileKey.FromB64(pk); err != nil {
		log.Println(err.Error())
		return nil, err
	}
	node.contentKey = profileKey
	log.Println("Profile Loaded: " + profileKey.GetPubKey().ToB64())
	return profileKey.GetPubKey(), nil
}

// GetPeer : Retrieve a peer by name
func (node *Node) GetPeer(name string) (*api.Peer, error) {
	c := node.db()
	r := transactQueryRow(c, "SELECT uri,enabled FROM peers WHERE name==$1;", name)
	var u string
	var e bool
	if err := r.Scan(&u, e); err == sql.ErrNoRows {
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
	var r *sql.Rows
	r = transactQuery(c, "SELECT name,uri,enabled FROM peers;")
	var peers []api.Peer
	for r.Next() {
		var s api.Peer
		r.Scan(&s.Name, &s.URI, &s.Enabled)
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
		//log.Println("-> New Server")
		transactExec(c, "INSERT INTO peers (name,uri,enabled) VALUES( $1, $2, $3 );",
			name, uri, enabled)
	} else if err == nil {
		//log.Println("-> Update Server")
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
	var rxsum []byte
	var err error

	var destkey bc.PubKey
	if len(pubkey) > 0 { // third argument is optional pubkey override
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

	// prepend a uint16 zero, meaning channel name length is zero
	rxsum = []byte{0, 0}

	return node.send("", rxsum, destkey, data, c)
}

// SendChannel : Transmit a message to a channel
func (node *Node) SendChannel(channelName string, data []byte, pubkey ...bc.PubKey) error {
	c := node.db()
	var destkey bc.PubKey
	var rxsum []byte

	if len(pubkey) > 0 { // third argument is optional PubKey override
		destkey = pubkey[0]
	} else {
		key, ok := node.channelKeys[channelName]
		if !ok {
			return errors.New("No public key for Channel")
		}
		destkey = key.GetPubKey()
	}

	// prepend a uint16 of channel name length, little-endian
	var t uint16
	t = uint16(len(channelName))
	rxsum = []byte{byte(t >> 8), byte(t & 0xFF)}
	rxsum = append(rxsum, []byte(channelName)...)

	return node.send(channelName, rxsum, destkey, data, c)
}

func (node *Node) send(channelName string, rxsum []byte, destkey bc.PubKey, msg []byte, c *sql.DB) error {
	// append a nonce
	salt, err := bc.GenerateRandomBytes(16)
	if err != nil {
		return err
	}
	rxsum = append(rxsum, salt...)

	// append a hash of content public key so recepient will know it's for them
	dh, err := bc.DestHash(destkey, salt)
	if err != nil {
		return err
	}
	rxsum = append(rxsum, dh...)

	//todo: is this passing msg by reference or not???
	data, err := node.contentKey.EncryptMessage(msg, destkey)
	if err != nil {
		return err
	}
	data = append(rxsum, data...)
	ts := time.Now().UnixNano()
	d := base64.StdEncoding.EncodeToString(data)
	transactExec(c, "INSERT INTO outbox(channel, msg, timestamp) VALUES($1,$2, $3);",
		channelName, d, ts)

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
			node.policies[i].RunPolicy()
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
				node.SendChannel(message.Name, message.Content.Bytes(), message.PubKey)
				break
			case false:
				node.Send(message.Name, message.Content.Bytes(), message.PubKey)
				break
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
