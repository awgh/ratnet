package db

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"time"

	"github.com/awgh/ratnet/api"
)

// GetChannelPrivKey : Return the private key of a given channel
func (node *Node) GetChannelPrivKey(name string) (string, error) {
	return node.dbGetChannelPrivKey(name)
}

// Forward - Add an already-encrypted message to the outbound message queue (forward it along)
func (node *Node) Forward(msg api.Msg) error {

	flags := uint8(0)
	if msg.IsChan {
		flags |= api.ChannelFlag
	}
	if msg.Chunked {
		flags |= api.ChunkedFlag
	}
	if msg.StreamHeader {
		flags |= api.StreamHeaderFlag
	}
	rxsum := []byte{flags} // prepend flags byte
	if msg.IsChan {
		// prepend a uint16 of channel name length, little-endian
		t := uint16(len(msg.Name))
		rxsum = append(rxsum, byte(t>>8), byte(t&0xFF))
		rxsum = append(rxsum, []byte(msg.Name)...)
	}
	message := append(rxsum, msg.Content.Bytes()...)
	return node.dbOutboxEnqueue(msg.Name, message, time.Now().UnixNano(), false)
}

// Handle - Decrypt and handle an encrypted message
func (node *Node) Handle(msg api.Msg) (bool, error) {
	var clear []byte
	var err error
	var tagOK bool
	var clearMsg api.Msg // msg to out channel

	if msg.Chunked {
		err := node.HandleChunk(msg)
		if err != nil {
			return false, err
		}
		return true, err
	}

	if msg.IsChan {
		v, ok := node.channelKeys[msg.Name]
		if !ok {
			return false, errors.New("Cannot Handle message for Unknown Channel")
		}
		clearMsg = api.Msg{Name: msg.Name, IsChan: true, Chunked: msg.Chunked, StreamHeader: msg.StreamHeader}
		tagOK, clear, err = v.DecryptMessage(msg.Content.Bytes())
	} else {
		clearMsg = api.Msg{Name: "[content]", IsChan: false, Chunked: msg.Chunked, StreamHeader: msg.StreamHeader}
		tagOK, clear, err = node.contentKey.DecryptMessage(msg.Content.Bytes())
	}
	// DecryptMessage will return !tagOK if the quick-check fails, which is common
	if !tagOK || err != nil {
		return tagOK, err
	}
	clearMsg.Content = bytes.NewBuffer(clear)

	select {
	case node.Out() <- clearMsg:
		node.debugMsg("Sent message " + fmt.Sprint(msg.Content.Bytes()))
	default:
		node.debugMsg("No message sent")
	}
	return tagOK, nil
}

// HandleChunk - store a partial message (chunk) in the node for later reconstruction
func (node *Node) HandleChunk(msg api.Msg) error {

	if msg.StreamHeader {
		// save totalChunks by streamID
		var streamID, totalChunks uint32
		binary.Read(msg.Content, binary.LittleEndian, &streamID)
		binary.Read(msg.Content, binary.LittleEndian, &totalChunks)
		channel := ""
		if msg.IsChan {
			channel = msg.Name
		}
		pk := ""
		if msg.PubKey != nil {
			pk = msg.PubKey.ToB64()
		}
		return node.dbAddStream(streamID, totalChunks, channel, pk)
	} else if msg.Chunked {
		// save chunk
		var streamID, chunkNum uint32
		binary.Read(msg.Content, binary.LittleEndian, &streamID)
		binary.Read(msg.Content, binary.LittleEndian, &chunkNum)
		data, err := ioutil.ReadAll(msg.Content)
		if err != nil {
			return err
		}
		return node.dbAddChunk(streamID, chunkNum, data)
	}
	return errors.New("HandleChunk called on a non-chunk")
}

func (node *Node) refreshChannels() { // todo: this could be selective or somehow less heavy
	// refresh the channelKeys map
	channels, _ := node.dbGetChannelPrivs()
	for _, element := range channels {
		node.channelKeys[element.Name] = element.Privkey
	}
}

func (node *Node) signalMonitor() {
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, nil)
	go func() {
		defer node.Stop()
		for {
			switch <-sigChannel {
			case os.Kill:
				return
			}
		}
	}()
}

/*
	TODO:	encrypted debug and error messages?! yes please!
			- you may want an application that can detect that messages have happend
			  while only being able to read them within the admin context
*/
func (node *Node) debugMsg(content string) {
	if node.debugMode {
		msg := new(api.Msg)
		msg.Name = "[DEBUG]"
		msg.Content = bytes.NewBufferString(content)
		node.Err() <- *msg
	}
}

func (node *Node) errMsg(err error, fatal bool) {
	msg := new(api.Msg)
	if node.debugMode {
		msg.Content = bytes.NewBufferString(err.Error() + "\n---\n" + string(debug.Stack()))
	} else {
		msg.Content = bytes.NewBufferString(err.Error())
	}
	msg.Name = "[ERROR]"
	msg.IsChan = fatal // use the "is channel" message flag as the "is fatal" flag
	node.Err() <- *msg
	if msg.IsChan {
		node.Stop()
	}

	log.Fatal(err) //todo: remove me
}
