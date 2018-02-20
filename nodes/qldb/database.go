package qldb

import (
	"database/sql"
	"log"
	"strconv"

	"github.com/awgh/ratnet/api"
)

var sqlDebug = false

// THIS SHOULD BE THE ONLY FILE THAT INCLUDES database/sql !!!
//		(ok, and qlnode, but only for the Node.db var declaration)
/*
var mutex = &sync.Mutex{}

func closeDB(db *sql.DB) {
	_ = db.Close()
}
*/
//
// Generic Database Functions
//

func transactExec(db *sql.DB, sql string, params ...interface{}) {
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err.Error())
	}
	if sqlDebug {
		log.Println(sql, params)
	}
	_, err = tx.Exec(sql, params...)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err.Error())
	}
}

func transactQuery(db *sql.DB, sql string, params ...interface{}) *sql.Rows {
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err.Error())
	}
	if sqlDebug {
		log.Println(sql, params)
	}
	r, err := tx.Query(sql, params...)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err.Error())
	}
	return r
}

func transactQueryRow(db *sql.DB, sql string, params ...interface{}) *sql.Row {
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err.Error())
	}
	if sqlDebug {
		log.Println(sql, params)
	}
	r := tx.QueryRow(sql, params...)
	err = tx.Commit()
	if err != nil {
		log.Fatal(err.Error())
	}
	return r
}

//
// End Generic Database Functions
//

//
// Specific Database Functions
//

func (node *Node) qlGetContactPubKey(name string) (string, error) {
	//mutex.Lock()
	//defer mutex.Unlock()
	c := node.db()
	//defer closeDB(c)
	//defer c.Close()
	r := c.QueryRow("SELECT cpubkey FROM contacts WHERE name==$1;", name)
	//r := transactQueryRow(c, "SELECT cpubkey FROM contacts WHERE name==$1;", name)
	var pubs string
	if err := r.Scan(&pubs); err == sql.ErrNoRows {
		return "", nil
	} else if err != nil {
		node.errMsg(err, true)
		return "", err
	}
	return pubs, nil
}

func (node *Node) qlGetContacts() ([]api.Contact, error) {
	c := node.db()
	s, err := c.Query("SELECT name,cpubkey FROM contacts;")
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
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

func (node *Node) qlGetChannelPrivKey(name string) (string, error) {
	c := node.db()
	r := c.QueryRow("SELECT privkey FROM channels WHERE name==$1;", name)
	var privkey string
	if err := r.Scan(&privkey); err == sql.ErrNoRows {
		return "", nil
	} else if err != nil {
		node.errMsg(err, true)
		return "", err
	}
	return privkey, nil
}

func (node *Node) qlGetChannels() ([]api.Channel, error) {
	c := node.db()
	r, err := c.Query("SELECT name,privkey FROM channels;")
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
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
		channels = append(channels,
			api.Channel{Name: n, Pubkey: prv.GetPubKey().ToB64()})
	}
	return channels, nil
}

func (node *Node) qlGetChannelPrivs() ([]api.ChannelPriv, error) {
	c := node.db()
	r, err := c.Query("SELECT name,privkey FROM channels;")
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	var channels []api.ChannelPriv
	for r.Next() {
		var n, p string
		if err := r.Scan(&n, &p); err != nil {
			return nil, err
		}
		prv := node.contentKey.Clone()
		if err := prv.FromB64(p); err != nil {
			return nil, err
		}
		channels = append(channels,
			api.ChannelPriv{Name: n, Privkey: prv,
				Pubkey: prv.GetPubKey().ToB64()})
	}
	return channels, nil
}

func (node *Node) qlGetProfile(name string) (*api.Profile, error) {
	c := node.db()
	r := c.QueryRow("SELECT enabled,privkey FROM profiles WHERE name==$1;", name)
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

func (node *Node) qlGetProfiles() ([]api.Profile, error) {
	c := node.db()
	r, err := c.Query("SELECT name,enabled,privkey FROM profiles;")
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
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

func (node *Node) qlGetPeer(name string) (*api.Peer, error) {
	c := node.db()
	r := c.QueryRow("SELECT uri,enabled FROM peers WHERE name==$1;", name)
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

func (node *Node) qlGetPeers() ([]api.Peer, error) {
	c := node.db()
	r, err := c.Query("SELECT name,uri,enabled FROM peers;")
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
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

func outboxBulkInsert(db *sql.DB, channelName string, timestamp int64, msgs [][]byte) {
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err.Error())
	}
	args := make([]interface{}, 1+(2*len(msgs)))
	args[0] = channelName
	//args[1] = timestamp
	idx := 2                                                    // starting 1-based index for 2nd arg
	sql := "INSERT INTO outbox(channel, msg, timestamp) VALUES" //($1,$2, $3);
	for i, v := range msgs {
		//sql += "($1,$" + strconv.Itoa(i+3) + ", $2)"
		sql += "($1,$" + strconv.Itoa(idx) + ", $" + strconv.Itoa(idx+1) + ")"
		if i != len(msgs) {
			sql += ", "
		} else {
			sql += ";"
		}
		args[idx-1] = v // convert to 0-based index here
		args[idx] = timestamp
		timestamp++ // increment timestamp by one each message to simplify queueing
		idx += 2
	}
	_, err = tx.Exec(sql, args...)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err.Error())
	}
}
