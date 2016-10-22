package qldb

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/ratnet/api"
)

// ID : Return routing key
func (node *Node) ID() (bc.PubKey, error) {
	return node.routingKey.GetPubKey(), nil
}

// Dropoff : Deliver a batch of  messages to a remote node
func (node *Node) Dropoff(bundle api.Bundle) error {
	node.debugMsg("Dropoff called")
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
		node.router.Route(node, msg)
	}
	node.debugMsg("Dropoff returned")
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
	node.debugMsg("Pickup called")
	c := node.db()
	var retval api.Bundle
	wildcard := false
	if len(channelNames) < 1 {
		wildcard = true // if no channels are given, get everything
	} else {
		for _, cname := range channelNames {
			for _, char := range cname {
				if strings.Index("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0987654321", string(char)) == -1 {
					return retval, errors.New("Invalid character in channel name!")
				}
			}
		}
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
	node.debugMsg("Pickup returned")
	return retval, nil
}
