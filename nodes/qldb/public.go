package qldb

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/ratnet/api"
)

// ID : Return routing key
func (node *Node) ID() (bc.PubKey, error) {
	return node.routingKey.GetPubKey(), nil
}

// Dropoff : Deliver a batch of  messages to a remote node
func (node *Node) Dropoff(bundle api.Bundle) error {
	c := node.db()
	if len(bundle.Data) < 1 { // todo: correct min length
		return errors.New("Dropoff called with no data.")
	}
	data, err := node.routingKey.DecryptMessage(bundle.Data)
	if err != nil {
		return err
	}
	var lines []string
	if err := json.Unmarshal(data, &lines); err != nil {
		return err
	}
	for i := 0; i < len(lines); i++ {
		if len(lines[i]) < 16 { // aes.BlockSize == 16
			continue //todo: remove padding before here?
		}
		msg, err := base64.StdEncoding.DecodeString(lines[i])
		if err != nil {
			continue
		}
		checkMessageForMe := true

		var channelLen uint16 // beginning uint16 of message is channel name length
		channelLen = (uint16(msg[0]) << 8) | uint16(msg[1])

		if len(msg) < int(channelLen)+2+16+16 { // uint16 + nonce + hash //todo
			log.Println("Incorrect channel name length")
			continue
		}
		var crypt bc.KeyPair
		channelName := ""
		if channelLen == 0 { // private message (zero length channel)
			crypt = node.contentKey
		} else { // channel message
			channelName = string(msg[2 : 2+channelLen])
			var ok bool
			crypt, ok = node.channelKeys[channelName]
			if !ok { // we are not listening to this channel
				checkMessageForMe = false
			}
		}
		msg = msg[2+channelLen:] //skip over the channel name
		forward := true

		if node.seenRecently(msg[:16]) { // LOOP PREVENTION before handling or forwarding
			forward = false
			checkMessageForMe = false
		}
		if checkMessageForMe { // check to see if this is a msg for me
			pubkey := crypt.GetPubKey()
			hash, err := bc.DestHash(pubkey, msg[:16])
			if err != nil {
				continue
			}
			if bytes.Equal(hash, msg[16:16+len(hash)]) {
				if channelLen == 0 {
					forward = false
				}
				clear, err := crypt.DecryptMessage(msg[16+len(hash):])
				if err != nil {
					continue
				}
				var clearMsg api.Msg // write msg to out channel
				if channelLen == 0 {
					clearMsg = api.Msg{Name: "[content]", IsChan: false}
				} else {
					clearMsg = api.Msg{Name: channelName, IsChan: true}
				}
				clearMsg.Content = bytes.NewBuffer(clear)

				select {
				case node.Out() <- clearMsg:
					//fmt.Println("sent message", msg)
				default:
					//fmt.Println("no message sent")
				}
			}
		}
		if forward {
			// save message in my outbox, if not already present
			r1 := transactQueryRow(c, "SELECT channel FROM outbox WHERE channel==$1 AND msg==$2;", channelName, lines[i])
			var rc string
			err := r1.Scan(&rc)
			if err == sql.ErrNoRows {
				// we don't have this yet, so add it
				t := time.Now().UnixNano()
				transactExec(c, "INSERT INTO outbox(channel,msg,timestamp) VALUES($1,$2,$3);",
					channelName, lines[i], t)
			} else if err != nil {
				return err
			}
		}
	}
	//log.Println("Dropoff returned")
	return nil
}

/* todo: when multiple profiles enabled at once is implemented, switch to the below (or similar):
profiles, err := node.GetProfiles()
if err != nil {
	node.handleErr(err)
	continue
}
for _, profile := range profiles {
	if profile.Enabled {
		clearMsg.Name = profile.Name
		break
	}
}
*/

// Pickup : Get messages from a remote node
func (node *Node) Pickup(rpub bc.PubKey, lastTime int64, channelNames ...string) (api.Bundle, error) {
	c := node.db()
	var retval api.Bundle
	wildcard := false
	if len(channelNames) < 1 {
		wildcard = true // if no channels are given, get everything
	}
	sqlq := "SELECT msg, timestamp FROM outbox"
	if lastTime != 0 {
		sqlq += " WHERE (int64(" + strconv.FormatInt(lastTime, 10) +
			") < timestamp)"
	}
	if !wildcard && len(channelNames) > 0 { // QL is broken?  couldn't make it work with prepared stmts
		if lastTime != 0 {
			sqlq += " AND"
		} else {
			sqlq += " WHERE"
		}
		sqlq = sqlq + " channel IN( \"" + channelNames[0] + "\""
		for i := 1; i < len(channelNames); i++ {
			sqlq = sqlq + ",\"" + channelNames[i] + "\""
		}
		sqlq = sqlq + " )"
	}
	// todo:  ORDER BY breaks on android/arm and returns nothing without error, report to cznic
	//			sqlq = sqlq + " ORDER BY timestamp ASC;"
	sqlq = sqlq + ";"
	r := transactQuery(c, sqlq)

	var msgs []string
	for r.Next() {
		var msg string
		var ts int64
		r.Scan(&msg, &ts)
		if ts > lastTime { // do this instead of ORDER BY, for android
			lastTime = ts
		}
		msgs = append(msgs, msg)
	}
	retval.Time = lastTime
	if len(msgs) > 0 {
		j, err := json.Marshal(msgs)
		if err != nil {
			return retval, err
		}
		cipher, err := node.routingKey.EncryptMessage(j, rpub)
		if err != nil {
			return retval, err
		}
		retval.Data = cipher
		return retval, err
	}
	return retval, nil
}

/*  todo : removed this, channelNames should be []bytes here ultimately - human-readability is up to the app
else {
	for i := 0; i < len(channelNames); i++ {
		for _, char := range channelNames[i] {
			if strings.Index("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0987654321", string(char)) == -1 {
				return nil, errors.New("Invalid characters in channel name!")
			}
		}
	}
}
*/
