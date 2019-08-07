package fs

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"

	"github.com/awgh/ratnet/api"
)

// GetChannelPrivKey : Return the private key of a given channel
func (node *Node) GetChannelPrivKey(name string) (string, error) {
	c, ok := node.channels[name]
	if !ok {
		return "", errors.New("Channel not found")
	}
	return c.Privkey.ToB64(), nil
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
	m := new(outboxMsg)
	path := node.basePath
	if msg.IsChan {
		// prepend a uint16 of channel name length, little-endian
		t := uint16(len(msg.Name))
		rxsum = append(rxsum, byte(t>>8), byte(t&0xFF))
		rxsum = append(rxsum, []byte(msg.Name)...)
		m.channel = msg.Name
		// create channel dir if not exist
		path = filepath.Join(path, msg.Name)
		os.Mkdir(path, os.FileMode(int(0700)))
	}
	message := append(rxsum, msg.Content.Bytes()...)
	/*
		for _, mail := range node.outbox {
			if mail.channel == channelName && bytes.Equal(mail.msg, message) {
				return nil // already have a copy... //todo: do we really need this check?
			}
		}
	*/

	f, err := os.Create(filepath.Join(path, hex(node.outboxIndex)))
	if err != nil {
		return err
	}
	node.outboxIndex++
	defer f.Close()
	w := bufio.NewWriter(f)
	w.Write(message)
	w.Flush()

	return nil
}

// Handle - Decrypt and handle an encrypted message
func (node *Node) Handle(msg api.Msg) (bool, error) {
	var clear []byte
	var err error
	var tagOK bool
	var clearMsg api.Msg // msg to out channel

	if msg.IsChan {
		v, ok := node.channels[msg.Name]
		if !ok || v.Privkey == nil {
			return tagOK, errors.New("Cannot Handle message for Unknown Channel")
		}
		clearMsg = api.Msg{Name: msg.Name, IsChan: true, Chunked: msg.Chunked, StreamHeader: msg.StreamHeader}
		tagOK, clear, err = v.Privkey.DecryptMessage(msg.Content.Bytes())
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
}
