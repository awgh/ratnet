package ram

import (
	"bytes"
	"encoding/gob"
	"errors"
	"log"

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

	var msgs [][]byte

	//Use default gob decoder
	reader := bytes.NewReader(data)
	dec := gob.NewDecoder(reader)
	if err := dec.Decode(&msgs); err != nil {
		log.Printf("dropoff gob decode failed, len %d\n", len(data))
		return err
	}
	for i := 0; i < len(msgs); i++ {
		if len(msgs[i]) < 16 { // aes.BlockSize == 16
			continue //todo: remove padding before here?
		}
		err = node.router.Route(node, msgs[i])
		if err != nil {
			log.Println("error in dropoff: " + err.Error())
			continue // we don't want to return routing errors back out the remote public interface
		}
	}

	node.debugMsg("Dropoff returned")
	return nil
}

// Pickup : Get messages from a remote node
func (node *Node) Pickup(rpub bc.PubKey, lastTime int64, maxBytes int64, channelNames ...string) (api.Bundle, error) {
	node.debugMsg("Pickup called")
	var retval api.Bundle
	var msgs [][]byte

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
				msgsSize := 0
				for i := range msgs {
					msgsSize += len(msgs[i])
				}

				proposedSize := len(mail.msg) + msgsSize

				if maxBytes > 0 && int64(proposedSize) > maxBytes {

					if msgsSize == 0 {
						log.Fatal("Bailing with zero return results!", proposedSize, len(mail.msg), msgsSize, maxBytes)
					}

					break
				}
				retval.Time = mail.timeStamp
				msgs = append(msgs, mail.msg)
			}
		}
	}

	// transmit
	if len(msgs) > 0 {

		//use default gob encoder
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		if err := enc.Encode(msgs); err != nil {
			return retval, err
		}
		cipher, err := node.routingKey.EncryptMessage(buf.Bytes(), rpub)
		if err != nil {
			return retval, err
		}
		retval.Data = cipher
		return retval, err
	}
	node.debugMsg("Pickup returned")
	return retval, nil
}
