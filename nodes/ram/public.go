package ram

import (
	"encoding/base64"
	"encoding/json"
	"errors"

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
		return errors.New("Dropoff called with no data")
	}
	tagOK, data, err := node.routingKey.DecryptMessage(bundle.Data)
	if err != nil {
		return err
	} else if !tagOK {
		return errors.New("Luggage Tag Check Failed in Dropoff")
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
		err = node.router.Route(node, msg)
		if err != nil {
			return err
		}
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
	var retval api.Bundle
	var msgs []string

	retval.Time = lastTime

	for _, mail := range node.outbox {
		if lastTime < mail.timeStamp {
			pickupMsg := false
			if len(channelNames) > 0 {
				for _, channelName := range channelNames {
					if channelName == mail.channel {
						pickupMsg = true
					}
				}
			} else {
				pickupMsg = true
			}
			if pickupMsg {
				msgs = append(msgs, mail.msg)
				retval.Time = mail.timeStamp
			}
		}
	}

	// transmit
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
