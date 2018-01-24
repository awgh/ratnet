package ram

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"time"

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
func (node *Node) Forward(channelName string, message []byte) error {
	// prepend a uint16 of channel name length, little-endian
	t := uint16(len(channelName))
	rxsum := []byte{byte(t >> 8), byte(t & 0xFF)}
	rxsum = append(rxsum, []byte(channelName)...)
	message = append(rxsum, message...)

	b64msg := base64.StdEncoding.EncodeToString(message)
	for _, mail := range node.outbox {
		if mail.channel == channelName && mail.msg == b64msg {
			return nil // already have a copy... //todo: do we really need this check?
		}
	}

	m := new(outboxMsg)
	m.channel = channelName
	m.timeStamp = time.Now().UnixNano()
	m.msg = b64msg
	node.outbox = append(node.outbox, m)
	return nil
}

// Handle - Decrypt and handle an encrypted message
func (node *Node) Handle(channelName string, message []byte) (bool, error) {
	var clear []byte
	var err error
	tagOK := false
	var clearMsg api.Msg // msg to out channel
	channelLen := len(channelName)

	if channelLen > 0 {
		v, ok := node.channels[channelName]
		if !ok || v.Privkey == nil {
			return tagOK, errors.New("Cannot Handle message for Unknown Channel")
		}
		clearMsg = api.Msg{Name: channelName, IsChan: true}
		tagOK, clear, err = v.Privkey.DecryptMessage(message)
	} else {
		clearMsg = api.Msg{Name: "[content]", IsChan: false}
		tagOK, clear, err = node.contentKey.DecryptMessage(message)
	}
	if err != nil {
		return tagOK, err
	} else if !tagOK {
		return false, errors.New("Luggage Tag Check Failed in Dropoff")
	}
	clearMsg.Content = bytes.NewBuffer(clear)

	select {
	case node.Out() <- clearMsg:
		node.debugMsg("Sent message " + fmt.Sprint(message))
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
