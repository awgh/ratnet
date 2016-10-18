package api

import (
	"bytes"
	"errors"

	"github.com/awgh/bencrypt/bc"
)

// Router : defines an interface for a stateful Routing object
type Router interface {
	// Route - TBD
	Route(node Node, msg []byte) error
}

// DefaultRouter - The Default router makes no changes at all,
//                 every message is sent out on the same channel it came in on,
//                 and non-channel messages are consumed but not forwarded
type DefaultRouter struct {
	recentPageIdx int
	recentPage1   map[string]byte
	recentPage2   map[string]byte
}

// NewDefaultRouter - returns a new instance of DefaultRouter
func NewDefaultRouter() *DefaultRouter {
	r := new(DefaultRouter)
	// init page maps
	r.recentPage1 = make(map[string]byte)
	r.recentPage2 = make(map[string]byte)
	return r
}

// Route - Router that does default behavior
func (r *DefaultRouter) Route(node Node, message []byte) error {

	checkMessageForMe := true
	var pubkey bc.PubKey
	//
	var channelLen uint16 // beginning uint16 of message is channel name length
	channelName := ""
	channelLen = (uint16(message[0]) << 8) | uint16(message[1])
	if len(message) < int(channelLen)+2+16+16 { // uint16 + nonce + hash //todo
		return errors.New("Incorrect channel name length")
	}
	cid, err := node.CID()
	if err != nil {
		return err
	}
	if channelLen > 0 { // channel message
		channelName = string(message[2 : 2+channelLen])
		chn, err := node.GetChannel(channelName)
		if err != nil {
			checkMessageForMe = false
		} else {
			pubkey = cid.Clone()
			pubkey.FromB64(chn.Pubkey)
		}
	} else { // private message (zero length channel)
		pubkey = cid
	}

	idx := 2 + channelLen //skip over the channel name
	forward := true

	nonce := message[idx : idx+16]
	if r.seenRecently(nonce) { // LOOP PREVENTION before handling or forwarding
		forward = false
		checkMessageForMe = false
	}
	if checkMessageForMe { // check to see if this is a msg for me
		hash, err := bc.DestHash(pubkey, nonce)
		if err != nil {
			return err
		}
		hashLen := uint16(len(hash))
		nonceHash := message[idx+16 : idx+16+hashLen]
		if bytes.Equal(hash, nonceHash) {
			if channelLen == 0 {
				forward = false
			}
			node.Handle(channelName, message[idx+16+hashLen:])
		}
	}
	if forward {
		return node.Forward(channelName, message)
	}
	return nil
}

func (r *DefaultRouter) seenRecently(hdr []byte) bool {
	shdr := string(hdr)
	_, aok := r.recentPage1[shdr]
	_, bok := r.recentPage2[shdr]
	retval := aok || bok

	switch r.recentPageIdx {
	case 1:
		if len(r.recentPage1) >= 50 {
			if len(r.recentPage2) >= 50 {
				r.recentPage2 = nil
				r.recentPage2 = make(map[string]byte)
			}
			r.recentPageIdx = 2
			r.recentPage2[shdr] = 1
		} else {
			r.recentPage1[shdr] = 1
		}
	case 2:
		if len(r.recentPage2) >= 50 {
			if len(r.recentPage1) >= 50 {
				r.recentPage1 = nil
				r.recentPage1 = make(map[string]byte)
			}
			r.recentPageIdx = 1
			r.recentPage1[shdr] = 1
		} else {
			r.recentPage2[shdr] = 1
		}
	}
	return retval
}
