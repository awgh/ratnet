package qldb

import (
	"bytes"
	"encoding/gob"
	"errors"
	"log"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/ratnet/api"
	"github.com/awgh/ratnet/api/events"
)

// ID : Return routing key
func (node *Node) ID() (bc.PubKey, error) {
	return node.routingKey.GetPubKey(), nil
}

// Dropoff : Deliver a batch of  messages to a remote node
func (node *Node) Dropoff(bundle api.Bundle) error {
	events.Debug(node, "Dropoff called")
	if len(bundle.Data) < 1 { // todo: correct min length
		return errors.New("Dropoff called with no data")
	}
	tagOK, data, err := node.routingKey.DecryptMessage(bundle.Data)
	if err != nil {
		return err
	} else if !tagOK {
		return errors.New("Luggage Tag Check Failed in QLNode Dropoff")
	}
	var msgs [][]byte

	//Use default gob decoder
	reader := bytes.NewReader(data)
	dec := gob.NewDecoder(reader)
	if erra := dec.Decode(&msgs); erra != nil {
		log.Printf("dropoff gob decode failed, len %d\n", len(data))
		return erra
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
	events.Debug(node, "Dropoff returned")
	return nil
}

// Pickup : Get messages from a remote node
func (node *Node) Pickup(rpub bc.PubKey, lastTime int64, maxBytes int64, channelNames ...string) (api.Bundle, error) {
	events.Debug(node, "Pickup called")
	var retval api.Bundle

	msgs, lastTimeReturned, err := node.qlGetMessages(lastTime, maxBytes, channelNames...)
	if err != nil {
		return retval, err
	}
	//log.Printf("maxBytes = %d, bytesRead = %d\n", maxBytes, bytesRead)

	// Return things

	//log.Printf("rows returned by Pickup query: %d, lastTime: %d\n", len(msgs), lastTimeReturned)
	retval.Time = lastTimeReturned
	if len(msgs) > 0 {
		//use default gob encoder
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		if err := enc.Encode(msgs); err != nil {
			return retval, err
		}
		cipher, err := node.routingKey.EncryptMessage(buf.Bytes(), rpub)
		if err != nil {
			log.Printf("pickup gob encode failed, len %d\n", len(cipher))
			return retval, err
		}
		retval.Data = cipher

		msgs = nil
		return retval, err
	}
	events.Debug(node, "Pickup returned")
	return retval, nil
}
