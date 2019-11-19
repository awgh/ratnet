package db

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/awgh/ratnet/api"
	"github.com/awgh/ratnet/api/chunking"
	"github.com/awgh/ratnet/api/events"
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

	if msg.Chunked {
		err = chunking.HandleChunked(node, clearMsg)
		if err != nil {
			return false, err
		}
		return true, err
	}

	select {
	case node.Out() <- clearMsg:
		events.Debug(node, "Sent message "+fmt.Sprint(msg.Content.Bytes()))
	default:
		events.Debug(node, "No message sent")
	}
	return tagOK, nil
}

func (node *Node) refreshChannels() { // todo: this could be selective or somehow less heavy
	// refresh the channelKeys map
	channels, _ := node.dbGetChannelsPriv()
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
