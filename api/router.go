package api

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/awgh/bencrypt/bc"
)

// Router : defines an interface for a stateful Routing object
type Router interface {
	// Route - TBD
	Route(node Node, msg []byte) error
	Patch(from string, to ...string)
	MarshalJSON() (b []byte, e error)
}

// DefaultRouter - The Default router makes no changes at all,
//                 every message is sent out on the same channel it came in on,
//                 and non-channel messages are consumed but not forwarded
type DefaultRouter struct {
	// Internal
	initialized   bool
	recentPageIdx int
	recentPage1   map[string]byte
	recentPage2   map[string]byte

	Patches map[string][]string

	// Configuration Settings

	// CheckContent - Check if incoming messages are for the contentKey
	CheckContent bool
	// CheckChannels - Check if incoming messages are for any of the channel keys
	CheckChannels bool
	// CheckProfiles - Check if incoming messages are for any of the profile keys
	CheckProfiles bool

	// ForwardConsumedContent - Should node forward consumed messages that matched contentKey
	ForwardConsumedContent bool
	// ForwardConsumedContent - Should node forward consumed messages that matched a channel key
	ForwardConsumedChannels bool
	// ForwardConsumedProfile - Should node forward consumed messages that matched a profile key
	ForwardConsumedProfiles bool

	// ForwardUnknownContent - Should node forward non-consumed messages that matched contentKey
	ForwardUnknownContent bool
	// ForwardUnknownContent - Should node forward non-consumed messages that matched a channel key
	ForwardUnknownChannels bool
	// ForwardUnknownProfile - Should node forward non-consumed messages that matched a profile key
	ForwardUnknownProfiles bool
}

// NewDefaultRouter - returns a new instance of DefaultRouter
func NewDefaultRouter() *DefaultRouter {
	r := new(DefaultRouter)
	r.Patches = make(map[string][]string)
	r.CheckContent = true
	r.CheckChannels = true
	r.CheckProfiles = false
	r.ForwardUnknownContent = true
	r.ForwardUnknownChannels = true
	r.ForwardUnknownProfiles = false
	r.ForwardConsumedContent = false
	r.ForwardConsumedChannels = true
	r.ForwardConsumedProfiles = false
	return r
}

// Patch - Redirect messages from one input to different outputs
func (r *DefaultRouter) Patch(from string, to ...string) {
	r.Patches[from] = to
}

func (r *DefaultRouter) forward(node Node, channelName string, message []byte) error {
	v, ok := r.Patches[channelName]
	if ok {
		for i := 0; i < len(v); i++ {
			if err := node.Forward(v[i], message); err != nil {
				return err
			}
		}
	} else {
		if err := node.Forward(channelName, message); err != nil {
			return err
		}
	}
	return nil
}

func (r *DefaultRouter) check(node Node, pubkey bc.PubKey, channelName string, idx uint16, nonce []byte, message []byte) (bool, error) {
	hash, err := bc.DestHash(pubkey, nonce)
	if err != nil {
		return false, err
	}
	hashLen := uint16(len(hash))
	nonceHash := message[idx+16 : idx+16+hashLen]
	if bytes.Equal(hash, nonceHash) { // named channel key match
		if err := node.Handle(channelName, message[idx+16+hashLen:]); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// Route - Router that does default behavior
func (r *DefaultRouter) Route(node Node, message []byte) error {

	//  Stuff Everything will need just about every time...
	//
	var channelLen uint16 // beginning uint16 of message is channel name length
	channelName := ""
	channelLen = (uint16(message[0]) << 8) | uint16(message[1])
	if len(message) < int(channelLen)+2+16+16 { // uint16 + nonce + hash //todo
		return errors.New("Incorrect channel name length")
	}
	idx := 2 + channelLen //skip over the channel name
	nonce := message[idx : idx+16]
	if r.seenRecently(nonce) { // LOOP PREVENTION before handling or forwarding
		return nil
	}

	cid, err := node.CID() // we need this for cloning
	if err != nil {
		return err
	}
	//

	// When the channel tag is set...
	if channelLen > 0 { // channel message
		channelName = string(message[2 : 2+channelLen])
		consumed := false
		if r.CheckChannels {
			chn, err := node.GetChannel(channelName)
			if err == nil { // this is a channel key we know
				pubkey := cid.Clone()
				pubkey.FromB64(chn.Pubkey)
				consumed, err = r.check(node, pubkey, channelName, idx, nonce, message)
				if err != nil {
					return err
				}
			}
		}
		if (!consumed && r.ForwardUnknownChannels) || (consumed && r.ForwardConsumedChannels) {
			if err := r.forward(node, channelName, message); err != nil {
				return err
			}
		}
	} else { // private message (zero length channel)
		// content key case (to be removed, deprecated)
		consumed := false
		if r.CheckContent {
			consumed, err = r.check(node, cid, channelName, idx, nonce, message)
			if err != nil {
				return err
			}
		}
		if (!consumed && r.ForwardUnknownContent) || (consumed && r.ForwardConsumedContent) {
			if err := r.forward(node, channelName, message); err != nil {
				return err
			}
		}

		// profile keys case
		consumed = false
		if r.CheckProfiles {
			profiles, err := node.GetProfiles()
			if err != nil {
				return err
			}
			for _, profile := range profiles {
				if !profile.Enabled {
					continue
				}
				pubkey := cid.Clone()
				pubkey.FromB64(profile.Pubkey)
				consumed, err = r.check(node, pubkey, channelName, idx, nonce, message)
				if err != nil {
					return err
				}
				if consumed {
					break
				}
			}
		}
		if (!consumed && r.ForwardUnknownProfiles) || (consumed && r.ForwardConsumedProfiles) {
			if err := r.forward(node, channelName, message); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *DefaultRouter) seenRecently(hdr []byte) bool {
	if !r.initialized {
		// init page maps
		r.recentPage1 = make(map[string]byte)
		r.recentPage2 = make(map[string]byte)
		r.initialized = true
	}
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

func (r *DefaultRouter) MarshalJSON() (b []byte, e error) {

	return json.Marshal(map[string]interface{}{
		"type":                    "default",
		"CheckContent":            r.CheckContent,
		"CheckChannels":           r.CheckChannels,
		"CheckProfiles":           r.CheckProfiles,
		"ForwardUnknownContent":   r.ForwardUnknownContent,
		"ForwardUnknownChannels":  r.ForwardUnknownChannels,
		"ForwardUnknownProfiles":  r.ForwardUnknownProfiles,
		"ForwardConsumedContent":  r.ForwardConsumedContent,
		"ForwardConsumedChannels": r.ForwardConsumedChannels,
		"ForwardConsumedProfiles": r.ForwardConsumedProfiles,
		"Patches":                 r.Patches})
}
