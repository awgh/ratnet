package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/awgh/ratnet/api"

	"upper.io/db.v3"
	"upper.io/db.v3/lib/sqlbuilder"
)

var sqlDebug = false

// THIS SHOULD BE THE ONLY FILE THAT INCLUDES upper db !!!
// ... other than the database var definition in dbnode.go and tests

func closeDB(database db.Database) {
	_ = database.Close()
}

//
// Specific Database Functions
//

func (node *Node) dbGetContactPubKey(name string) (string, error) {
	col := node.db.Collection("contacts")
	res := col.Find().Where("name = ?", name)
	var contact api.Contact

	if err := res.One(&contact); err != nil {
		return "", err
	}
	return contact.Pubkey, nil
}

func (node *Node) dbGetContacts() ([]api.Contact, error) {
	col := node.db.Collection("contacts")
	res := col.Find()
	var contacts []api.Contact

	if err := res.All(&contacts); err != nil {
		return nil, err
	}
	return contacts, nil
}

func (node *Node) dbAddContact(name, pubkey string) error {
	tx, err := node.db.NewTx(context.TODO())
	if err != nil {
		return err
	}
	col := tx.Collection("contacts")
	res := col.Find("name = ?", name)
	cnt, err := res.Count()
	if err != nil {
		return err
	}
	if cnt > 0 {
		_ = res.Delete()
	}
	contact := api.Contact{Name: name, Pubkey: pubkey}
	_, err = col.Insert(&contact)
	if err != nil {
		s := err.Error()
		log.Println(s)
		return err
	}
	return tx.Commit()
}

func (node *Node) dbDeleteContact(name string) {
	col := node.db.Collection("contacts")
	res := col.Find("name = ?", name)
	_ = res.Delete()
}

func (node *Node) dbGetChannelPrivKey(name string) (string, error) {
	col := node.db.Collection("channels")
	res := col.Find().Where("name = ?", name)
	var channel api.ChannelPrivDB
	if err := res.One(&channel); err != nil {
		return "", err
	}
	return channel.Privkey, nil
}

func (node *Node) dbGetChannels() ([]api.Channel, error) {
	col := node.db.Collection("channels")
	res := col.Find()
	var channels []api.ChannelPrivDB
	if err := res.All(&channels); err != nil {
		return nil, err
	}
	var retval []api.Channel
	for _, v := range channels {
		prv := node.contentKey.Clone()
		if err := prv.FromB64(v.Privkey); err != nil {
			return nil, err
		}
		retval = append(retval, api.Channel{Name: v.Name, Pubkey: prv.GetPubKey().ToB64()})
	}
	return retval, nil
}

func (node *Node) dbGetChannelPrivs() ([]api.ChannelPriv, error) {
	col := node.db.Collection("channels")
	res := col.Find()
	var channels []api.ChannelPrivDB
	if err := res.All(&channels); err != nil {
		return nil, err
	}
	var retval []api.ChannelPriv
	for _, v := range channels {
		prv := node.contentKey.Clone()
		if err := prv.FromB64(v.Privkey); err != nil {
			return nil, err
		}
		retval = append(retval, api.ChannelPriv{Name: v.Name, Pubkey: prv.GetPubKey().ToB64(), Privkey: prv})
	}
	return retval, nil
}

func (node *Node) dbAddChannel(name, privkey string) error {
	tx, err := node.db.NewTx(context.TODO())
	if err != nil {
		return err
	}
	col := tx.Collection("channels")
	res := col.Find("name = ?", name)
	cnt, err := res.Count()
	if err != nil {
		return err
	}
	if cnt > 0 {
		_ = res.Delete()
	}
	prv := node.contentKey.Clone()
	if err := prv.FromB64(privkey); err != nil {
		return err
	}
	channel := api.ChannelPrivDB{Name: name, Privkey: prv.ToB64()}
	_, err = col.Insert(&channel)
	if err != nil {
		s := err.Error()
		log.Println(s)
		return err
	}
	return tx.Commit()
}

func (node *Node) dbDeleteChannel(name string) {
	col := node.db.Collection("channels")
	res := col.Find("name = ?", name)
	_ = res.Delete()
}

func (node *Node) dbGetProfile(name string) (*api.Profile, error) {
	col := node.db.Collection("profiles")
	res := col.Find().Where("name = ?", name)
	var profile api.ProfilePrivDB
	if err := res.One(&profile); err != nil {
		return nil, err
	}
	prv := node.contentKey.Clone()
	if err := prv.FromB64(profile.Privkey); err != nil {
		return nil, err
	}
	return &api.Profile{Name: profile.Name, Enabled: profile.Enabled, Pubkey: prv.GetPubKey().ToB64()}, nil
}

func (node *Node) dbGetProfiles() ([]api.Profile, error) {
	col := node.db.Collection("profiles")
	res := col.Find()
	var profiles []api.ProfilePrivDB
	if err := res.All(&profiles); err != nil {
		return nil, err
	}
	var retval []api.Profile
	for _, v := range profiles {
		prv := node.contentKey.Clone()
		if err := prv.FromB64(v.Privkey); err != nil {
			return nil, err
		}
		retval = append(retval, api.Profile{Name: v.Name, Enabled: v.Enabled, Pubkey: prv.GetPubKey().ToB64()})
	}
	return retval, nil
}

func (node *Node) dbAddProfile(name string, enabled bool) error {
	col := node.db.Collection("profiles")
	res := col.Find().Where("name = ?", name)
	count, err := res.Count()
	if err != nil {
		return err
	}
	var profile api.ProfilePrivDB
	if count == 0 {
		// generate new profile keypair
		profileKey := node.contentKey.Clone()
		profileKey.GenerateKey()

		// insert new profile
		profile.Name = name
		profile.Enabled = enabled
		profile.Privkey = profileKey.ToB64()
		_, err = col.Insert(profile)
		return err
	}
	err = res.One(&profile)
	if err != nil {
		return err
	}
	profile.Name = name
	profile.Enabled = enabled
	return res.Update(profile)
}

func (node *Node) dbDeleteProfile(name string) {
	col := node.db.Collection("profiles")
	res := col.Find("name = ?", name)
	_ = res.Delete()
}

func (node *Node) dbGetProfilePrivateKey(name string) string {
	col := node.db.Collection("profiles")
	res := col.Find().Where("name = ?", name)
	var profile api.ProfilePrivDB
	if err := res.One(profile); err != nil {
		return ""
	}
	return profile.Privkey
}

func (node *Node) dbGetPeer(name string) (*api.Peer, error) {
	col := node.db.Collection("peers")
	res := col.Find().Where("name = ?", name)
	var peer api.Peer
	if err := res.One(&peer); err != nil {
		return nil, err
	}
	return &peer, nil
}

func (node *Node) dbGetPeers(group string) ([]api.Peer, error) {
	col := node.db.Collection("peers")
	res := col.Find().Where("peergroup = ?", group)
	var peers []api.Peer
	if err := res.All(&peers); err != nil {
		return nil, err
	}
	return peers, nil
}

func (node *Node) dbAddPeer(name string, enabled bool, uri string, group string) error {
	col := node.db.Collection("peers")
	res := col.Find().Where("name = ?", name).And("peergroup = ?", group)
	count, err := res.Count()
	if err != nil {
		return err
	}
	var peer api.Peer
	if count == 0 {
		// insert new profile
		peer.Name = name
		peer.Enabled = enabled
		peer.Group = group
		peer.URI = uri
		_, err = col.Insert(peer)
		return err
	}
	err = res.One(&peer)
	if err != nil {
		return err
	}
	peer.Name = name
	peer.Enabled = enabled
	peer.Group = group
	peer.URI = uri
	return res.Update(peer)
}

func (node *Node) dbDeletePeer(name string) {
	col := node.db.Collection("peers")
	res := col.Find("name = ?", name)
	_ = res.Delete()
}

func (node *Node) dbOutboxEnqueue(channelName string, msg []byte, ts int64, checkExists bool) error {
	col := node.db.Collection("outbox")
	doInsert := !checkExists
	var outboxmsg api.OutboxMsg

	if checkExists {
		// save message in my outbox, if not already present
		res := col.Find("channel = ?", channelName).And("msg = ?", msg)
		if err := res.One(&outboxmsg); err != nil {
			return err
		}
		count, err := res.Count()
		if err != nil {
			return err
		}
		if count == 0 {
			// we don't have this yet, so add it
			doInsert = true
		}
	}
	if doInsert {
		outboxmsg.Channel = channelName
		outboxmsg.Msg = msg
		outboxmsg.Timestamp = ts
		_, err := col.Insert(&outboxmsg)
		return err
	}
	return nil
}

func (node *Node) outboxBulkInsert(channelName string, timestamp int64, msgs [][]byte) error {
	tx, err := node.db.NewTx(context.TODO())
	if err != nil {
		return err
	}
	col := tx.Collection("outbox")
	//todo: convert this to BatchInserter?
	for i, v := range msgs {
		var outboxmsg api.OutboxMsg
		outboxmsg.Channel = channelName
		outboxmsg.Msg = v
		outboxmsg.Timestamp = timestamp + int64(i) // increment timestamp by one each message to simplify queueing
		_, err := col.Insert(outboxmsg)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	tx.Commit()
	return nil
}

func (node *Node) dbGetMessages(lastTime, maxBytes int64, channelNames ...string) ([][]byte, int64, error) {
	lastTimeReturned := lastTime
	var args []interface{}
	var msgs [][]byte
	var bytesRead int64

	// Build the query
	wildcard := false
	if len(channelNames) < 1 {
		wildcard = true // if no channels are given, get everything
	}
	sqlq := "SELECT msg, timestamp FROM outbox"
	if lastTime != 0 {
		sqlq += " WHERE (? < timestamp)"
		args = append(args, lastTime)
	}
	if !wildcard && len(channelNames) > 0 {
		if lastTime != 0 {
			sqlq += " AND"
		} else {
			sqlq += " WHERE"
		}
		sqlq = sqlq + " channel IN( ?"
		args = append(args, channelNames[0])
		for i := 1; i < len(channelNames); i++ {
			sqlq = sqlq + ",?"
			args = append(args, channelNames[i])
		}
		sqlq = sqlq + " )"
	}
	sqlq = sqlq + " ORDER BY timestamp ASC;"
	res, err := node.db.Query(sqlq, args...)

	if res == nil || err != nil {
		return nil, lastTimeReturned, err
	}
	n := 0
	for res.Next() {
		n++
		var msg []byte
		var ts int64
		res.Scan(&msg, &ts)
		if bytesRead+int64(len(msg)) >= maxBytes { // no room for next msg
			log.Printf("skipping messages after %d results\n", n)
			if n == 0 {
				return nil, lastTimeReturned, errors.New("Result too big to be fetched on this transport! Flush and rechunk")
			}
		}
		if ts > lastTimeReturned {
			lastTimeReturned = ts
		} else {
			log.Printf("Timestamps not increasing - prev: %d  cur: %d\n", lastTimeReturned, ts)
		}
		msgs = append(msgs, msg)
		bytesRead += int64(len(msg))
	}
	//log.Println("last time/returned:", lastTime, lastTimeReturned)
	return msgs, lastTimeReturned, nil
}

func (node *Node) dbClearStream(streamID uint32) error {
	col := node.db.Collection("chunks")
	res := col.Find("streamid = ?", streamID)
	_ = res.Delete()
	col = node.db.Collection("streams")
	res = col.Find("streamid = ?", streamID)
	return res.Delete()
}

func (node *Node) AddStream(streamID uint32, totalChunks uint32, channelName string) error {
	col := node.db.Collection("streams")
	res := col.Find().Where("streamid = ?", streamID)
	count, err := res.Count()
	if err != nil {
		return err
	}
	var stream api.StreamHeader
	if count == 0 {
		// insert new stream
		stream.StreamID = streamID
		stream.NumChunks = totalChunks
		stream.ChannelName = channelName
		_, err = col.Insert(stream)
		return err
	}
	err = res.One(&stream)
	if err != nil {
		return err
	}
	log.Printf("warning: over-writing stream header: %x\n", streamID)
	stream.StreamID = streamID
	stream.NumChunks = totalChunks
	stream.ChannelName = channelName
	return res.Update(stream)
}

func (node *Node) AddChunk(streamID uint32, chunkNum uint32, data []byte) error {
	col := node.db.Collection("chunks")
	res := col.Find().Where("streamid = ?", streamID).And("chunknum = ?", chunkNum)
	count, err := res.Count()
	if err != nil {
		return err
	}
	var chunk api.Chunk
	if count == 0 {
		// insert new chunk
		chunk.StreamID = streamID
		chunk.ChunkNum = chunkNum
		chunk.Data = data
		_, err = col.Insert(chunk)
		return err
	}
	err = res.One(&chunk)
	if err != nil {
		return err
	}
	log.Printf("warning: over-writing chunk: %x:%x\n", streamID, chunkNum)
	chunk.StreamID = streamID
	chunk.ChunkNum = chunkNum
	chunk.Data = data
	return res.Update(chunk)
}

func (node *Node) dbGetStreams() ([]api.StreamHeader, error) {
	col := node.db.Collection("streams")
	res := col.Find()
	var streams []api.StreamHeader
	if err := res.All(&streams); err != nil {
		return nil, err
	}
	return streams, nil
}

func (node *Node) dbGetChunkCount(streamID uint32) (uint64, error) {
	col := node.db.Collection("chunks")
	res := col.Find().Where("streamid = ?", streamID)
	return res.Count()
}

func (node *Node) dbGetChunks(streamID uint32) ([]api.Chunk, error) {
	col := node.db.Collection("chunks")
	res := col.Find().Where("streamid = ?", streamID).OrderBy("chunknum")
	var chunks []api.Chunk
	if err := res.All(&chunks); err != nil {
		return nil, err
	}
	return chunks, nil
}

// FlushOutbox : Deletes outbound messages older than maxAgeSeconds seconds
func (node *Node) FlushOutbox(maxAgeSeconds int64) {
	ts := time.Now().UnixNano()
	ts = ts - (maxAgeSeconds * 1000000000)
	col := node.db.Collection("outbox")
	res := col.Find("timestamp < ?", ts)
	_ = res.Delete()
}

type connectionURL struct {
	url string
}

func (c connectionURL) String() string { return c.url }

// BootstrapDB - Initialize or open a database file
func (node *Node) BootstrapDB(dbAdapter, dbConnectionString string) sqlbuilder.Database {
	if node.db != nil {
		return node.db
	}
	var err error
	node.db, err = sqlbuilder.Open(dbAdapter, connectionURL{url: dbConnectionString})
	if err != nil {
		//node.errMsg(errors.New("DB Error Opening: "+dbAdapter+" => "+err.Error()), true)
		log.Fatal(err)
	}

	strName := getBackendType(dbAdapter, "string")
	blobName := getBackendType(dbAdapter, "blob")
	int64Name := getBackendType(dbAdapter, "int64")

	checkErr := func(e error) {
		if e != nil {
			panic(e)
		}
	}

	// One-time Initialization
	_, err = node.db.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS contacts (
			name	%s	NOT NULL,
			pubkey	%s	NOT NULL
		);		
	`, strName, strName))
	checkErr(err)

	_, err = node.db.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS channels ( 			
			name	%s	NOT NULL,
			privkey	%s	NOT NULL
		);
	`, strName, strName))
	checkErr(err)

	_, err = node.db.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS config ( 
			name	%s	NOT NULL,
			value	%s	NOT NULL
		);
	`, strName, strName))
	checkErr(err)

	_, err = node.db.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS outbox (
			channel		%s, 
			msg			%s	NOT NULL,
			timestamp	%s	NOT NULL
		);
	`, strName, blobName, int64Name))
	checkErr(err)

	_, err = node.db.Exec(`
			CREATE INDEX IF NOT EXISTS outboxID ON outbox (timestamp);
	`)
	checkErr(err)

	_, err = node.db.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS peers (
			name		%s		NOT NULL,  
			uri			%s		NOT NULL,
			enabled		bool	NOT NULL,
			peergroup   %s  	NOT NULL,
			pubkey		%s		
		);
	`, strName, strName, strName, strName))
	checkErr(err)

	_, err = node.db.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS profiles (
			name	%s		NOT NULL,
			privkey	%s		NOT NULL,
			enabled	bool	NOT NULL
		);
	`, strName, strName))
	checkErr(err)

	_, err = node.db.Exec(fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS chunks (		
		streamid	%s	NOT NULL,
		chunknum	%s	NOT NULL,
		data		%s	NOT NULL
	);
	`, int64Name, int64Name, blobName))
	checkErr(err)

	_, err = node.db.Exec(fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS streams (		
		streamid		%s	NOT NULL,
		parts			%s	NOT NULL,
		channel			%s	NOT NULL
	);
	`, int64Name, int64Name, strName))
	checkErr(err)

	// Content Key Setup
	col := node.db.Collection("config")
	res1 := col.Find("name = ?", "contentkey")
	cnt, err := res1.Count()
	if err != nil {
		node.errMsg(err, true)
		log.Fatal(err)
	} else if cnt == 0 {
		node.contentKey.GenerateKey()
		bs := node.contentKey.ToB64()
		cv := api.ConfigValue{Name: "contentkey", Value: bs}
		_, err = col.Insert(cv)
	} else {
		var cv api.ConfigValue
		res1.One(&cv)
		err = node.contentKey.FromB64(cv.Value)
	}
	if err != nil {
		node.errMsg(err, true)
	}

	// Routing Key Setup
	res2 := col.Find("name = ?", "routingkey")
	cnt, err = res2.Count()
	if err != nil {
		node.errMsg(err, true)
	} else if cnt == 0 {
		node.routingKey.GenerateKey()
		bs := node.routingKey.ToB64()
		cv := api.ConfigValue{Name: "routingkey", Value: bs}
		_, err = col.Insert(cv)
	} else {
		var cv api.ConfigValue
		res1.One(&cv)
		err = node.routingKey.FromB64(cv.Value)
	}
	if err != nil {
		node.errMsg(err, true)
	}

	node.refreshChannels()
	return node.db
}

func getBackendType(dbAdapter, dbType string) string {
	switch dbAdapter {
	case "postgresql":
		switch dbType {
		case "string":
			return "text"
		case "blob":
			return "bytea"
		case "int64":
			return "bigint"
		default:
		}
	case "mysql":
		switch dbType {
		case "string":
			return "text"
		case "blob":
			return dbType
		case "int64":
			return "bigint"
		default:
		}
	case "sqllite":
		switch dbType {
		case "string":
			return "text"
		case "blob":
			return dbType
		case "int64":
			return "integer"
		default:
		}
	case "mssql":
		switch dbType {
		case "string":
			return "varchar"
		case "blob":
			return "varbinary"
		case "int64":
			return "bigint"
		default:
		}
	case "ql":
		return dbType
	case "mongodb":
		switch dbType {
		case "string":
			return dbType
		case "blob":
			return "binData"
		case "int64":
			return "long"
		default:
		}
	default:
		panic(fmt.Sprintf("invalid database backend %s", dbAdapter))
	}

	panic(fmt.Sprintf("invalid data type %s", dbType))
}
