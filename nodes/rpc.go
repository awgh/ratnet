package nodes

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/ratnet/api"
)

// PublicRPC : Entrypoint for RPC functions that are exposed to the public/Internet
func PublicRPC(transport api.Transport, node api.Node, call api.RemoteCall) (interface{}, error) {
	switch call.Action {
	case api.ID:
		var i bc.PubKey
		i, err := node.ID()
		if err != nil {
			return nil, err
		} else if i == i.Nil() {
			return nil, errors.New("Node has no routing key set")
		}
		return i, nil

	case api.Pickup:
		if len(call.Args) < 2 {
			return nil, errors.New("Invalid argument count")
		}
		rpk, ok := call.Args[0].(bc.PubKey)
		if !ok {
			return nil, errors.New("Invalid argument 1")
		}
		i, ok := call.Args[1].(int64)
		if !ok {
			return nil, errors.New("Invalid argument 2")
		}
		var xargs []string // dunno how to type-assert slices
		for _, v := range call.Args[2:] {
			vs, ok := v.(string)
			if !ok {
				return nil, errors.New("Invalid argument 3+")
			}
			xargs = append(xargs, vs)
		}
		b, err := node.Pickup(rpk, i, transport.ByteLimit(), xargs...)
		if err != nil {
			return nil, err
		}
		return b, err

	case api.Dropoff:
		if len(call.Args) < 1 {
			return nil, errors.New("Invalid argument count")
		}
		bundle, ok := call.Args[0].(api.Bundle)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		return nil, node.Dropoff(bundle)

	default:
		return nil, fmt.Errorf("No such method: %d", call.Action)
	}
}

// AdminRPC : Entrypoint for administrative RPC functions that should not be exposed to the Internet
func AdminRPC(transport api.Transport, node api.Node, call api.RemoteCall) (interface{}, error) {
	switch call.Action {

	case api.CID:
		b, err := node.CID()
		if err != nil {
			return nil, err
		}
		return b, nil

	case api.GetContact:
		if len(call.Args) < 1 {
			return nil, errors.New("Invalid argument count")
		}
		contactName, ok := call.Args[0].(string)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		if peer, err := node.GetContact(contactName); err != nil {
			return nil, err
		} else if peer != nil {
			return peer, nil
		}
		return nil, nil

	case api.GetContacts:
		if peers, err := node.GetContacts(); err != nil {
			return nil, err
		} else {
			return peers, nil
		}

	case api.AddContact:
		if len(call.Args) < 2 {
			return nil, errors.New("Invalid argument count")
		}
		contactName, ok := call.Args[0].(string)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		contactB64Key, ok := call.Args[1].(string)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		return nil, node.AddContact(contactName, contactB64Key)

	case api.DeleteContact:
		if len(call.Args) < 1 {
			return nil, errors.New("Invalid argument count")
		}
		contactName, ok := call.Args[0].(string)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		return nil, node.DeleteContact(contactName)

	case api.GetChannel:
		if len(call.Args) < 1 {
			return nil, errors.New("Invalid argument count")
		}
		channelName, ok := call.Args[0].(string)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		if c, err := node.GetChannel(channelName); err != nil {
			return nil, err
		} else if c != nil {
			return c, nil
		}
		return nil, nil

	case api.GetChannels:
		if chans, err := node.GetChannels(); err != nil {
			return nil, err
		} else {
			return chans, nil
		}

	case api.AddChannel:
		if len(call.Args) < 2 {
			return nil, errors.New("Invalid argument count")
		}
		channelName, ok := call.Args[0].(string)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		channelB64Key, ok := call.Args[1].(string)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		return nil, node.AddChannel(channelName, channelB64Key)

	case api.DeleteChannel:
		if len(call.Args) < 1 {
			return nil, errors.New("Invalid argument count")
		}
		channelName, ok := call.Args[0].(string)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		return nil, node.DeleteChannel(channelName)

	case api.GetProfile:
		if len(call.Args) < 1 {
			return nil, errors.New("Invalid argument count")
		}
		profileName, ok := call.Args[0].(string)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		if profile, err := node.GetProfile(profileName); err != nil {
			return nil, err
		} else if profile != nil {
			return profile, nil
		}
		return nil, nil

	case api.GetProfiles:
		if profiles, err := node.GetProfiles(); err != nil {
			return nil, err
		} else {
			return profiles, nil
		}

	case api.AddProfile:
		if len(call.Args) < 2 {
			return nil, errors.New("Invalid argument count")
		}
		profileName, ok := call.Args[0].(string)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		profileEnabled, ok := call.Args[1].(string) // todo: convert this to bool?
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		enabled, err := strconv.ParseBool(profileEnabled)
		if err != nil {
			return nil, errors.New("Invalid bool format")
		}
		return nil, node.AddProfile(profileName, enabled)

	case api.DeleteProfile:
		if len(call.Args) < 1 {
			return nil, errors.New("Invalid argument count")
		}
		profileName, ok := call.Args[0].(string)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		return nil, node.DeleteProfile(profileName)

	case api.LoadProfile:
		if len(call.Args) < 1 {
			return nil, errors.New("Invalid argument count")
		}
		profileName, ok := call.Args[0].(string)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		if p, err := node.LoadProfile(profileName); err != nil {
			return nil, err
		} else if p != nil {
			return p, nil
		}
		return nil, nil

	case api.GetPeer:
		if len(call.Args) < 1 {
			return nil, errors.New("Invalid argument count")
		}
		peerName, ok := call.Args[0].(string)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		if peer, err := node.GetPeer(peerName); err != nil {
			return nil, err
		} else if peer != nil {
			return peer, nil
		}
		return nil, nil

	case api.GetPeers:
		group := ""
		if len(call.Args) > 1 {
			return nil, errors.New("Invalid argument count")
		}
		if len(call.Args) > 0 {
			var ok bool
			group, ok = call.Args[0].(string)
			if !ok {
				return nil, errors.New("Invalid argument")
			}
		}
		if peers, err := node.GetPeers(group); err != nil {
			return nil, err
		} else {
			return peers, nil
		}

	case api.AddPeer:
		if len(call.Args) < 3 {
			return nil, errors.New("Invalid argument count")
		}
		peerName, ok := call.Args[0].(string)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		peerEnabled, ok := call.Args[1].(string)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		peerURI, ok := call.Args[2].(string)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		enabled, err := strconv.ParseBool(peerEnabled)
		if err != nil {
			return nil, errors.New("Invalid bool format")
		}
		if len(call.Args) > 3 && call.Args[3] != nil {
			group, ok := call.Args[3].(string)
			if !ok {
				return nil, errors.New("Invalid argument")
			}
			return nil, node.AddPeer(peerName, enabled, peerURI, group)
		}
		return nil, node.AddPeer(peerName, enabled, peerURI)

	case api.DeletePeer:
		if len(call.Args) < 1 {
			return nil, errors.New("Invalid argument count")
		}
		peerName, ok := call.Args[0].(string)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		return nil, node.DeletePeer(peerName)

	case api.Send:
		if len(call.Args) < 2 {
			return nil, errors.New("Invalid argument count")
		}
		destName, ok := call.Args[0].(string)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		msg, ok := call.Args[1].([]byte)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		if len(call.Args) > 2 {
			rxPubKey, ok := call.Args[2].(bc.PubKey)
			if !ok {
				return nil, errors.New("Invalid argument")
			}
			return nil, node.Send(destName, msg, rxPubKey)
		}
		return nil, node.Send(destName, msg)

	case api.SendChannel:
		if len(call.Args) < 2 {
			return nil, errors.New("Invalid argument count")
		}
		channelName, ok := call.Args[0].(string)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		msg, ok := call.Args[1].([]byte)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		if len(call.Args) > 2 {
			rxPubKey, ok := call.Args[2].(bc.PubKey)
			if !ok {
				return nil, errors.New("Invalid argument")
			}
			return nil, node.SendChannel(channelName, msg, rxPubKey)
		}
		return nil, node.SendChannel(channelName, msg)

	default:
		return node.PublicRPC(transport, call)
	}
}
