package ram

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
					fmt.Println("sent message", msg)
				default:
					fmt.Println("no message sent")
				}
			}
		}
		if forward {
			for _, mail := range node.outbox {
				if mail.channel == channelName && mail.msg == lines[i] {
					m := new(outboxMsg)
					m.channel = channelName
					m.timeStamp = time.Now().UnixNano()
					m.msg = lines[i]
					node.outbox = append(node.outbox, m)
				}
			}
		}
	}
	log.Println("Dropoff returned")
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
	var retval api.Bundle
	var msgs []string
	for _, mail := range node.outbox {
		if lastTime != 0 {
			if lastTime < mail.timeStamp {
				msgs = append(msgs, mail.msg)
			}
		} else {
			msgs = append(msgs, mail.msg)
		}
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
