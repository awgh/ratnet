package nodes

import (
	"errors"
	"log"
	"strconv"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/ratnet/api"
)

// PublicRPC : Entrypoint for RPC functions that are exposed to the public/Internet
func PublicRPC(transport api.Transport, node api.Node, call api.RemoteCall) (interface{}, error) {

	//log.Printf("PublicRPC called with %+v\n", call)

	switch call.Action {
	case "ID":
		var i bc.PubKey
		i, err := node.ID()
		log.Printf("PublicRPC ID returned %+v : %+v\n", i, err)
		if err != nil {
			return nil, err
		} else if i == i.Nil() {
			return nil, errors.New("Node has no routing key set")
		}
		return i, nil

	case "Pickup":
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

	case "Dropoff":
		if len(call.Args) < 1 {
			return nil, errors.New("Invalid argument count")
		}
		bundle, ok := call.Args[0].(api.Bundle)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		return nil, node.Dropoff(bundle)

	default:
		return nil, errors.New("No such method: " + call.Action)
	}
}

// AdminRPC : Entrypoint for administrative RPC functions that should not be exposed to the Internet
func AdminRPC(transport api.Transport, node api.Node, call api.RemoteCall) (interface{}, error) {
	switch call.Action {

	case "CID":
		b, err := node.CID()
		if err != nil {
			return nil, err
		}
		return b, nil

	case "GetContact":
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

	case "GetContacts":
		if peers, err := node.GetContacts(); err != nil {
			return nil, err
		} else {
			return peers, nil
		}

	case "AddContact":
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

	case "DeleteContact":
		if len(call.Args) < 1 {
			return nil, errors.New("Invalid argument count")
		}
		contactName, ok := call.Args[0].(string)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		return nil, node.DeleteContact(contactName)

	case "GetChannel":
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

	case "GetChannels":
		if chans, err := node.GetChannels(); err != nil {
			return nil, err
		} else {
			return chans, nil
		}

	case "AddChannel":
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

	case "DeleteChannel":
		if len(call.Args) < 1 {
			return nil, errors.New("Invalid argument count")
		}
		channelName, ok := call.Args[0].(string)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		return nil, node.DeleteChannel(channelName)

	case "GetProfile":
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

	case "GetProfiles":
		if profiles, err := node.GetProfiles(); err != nil {
			return nil, err
		} else {
			return profiles, nil
		}

	case "AddProfile":
		if len(call.Args) < 2 {
			return nil, errors.New("Invalid argument count")
		}
		profileName, ok := call.Args[0].(string)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		profileEnabled, ok := call.Args[1].(string) //todo: convert this to bool?
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		enabled, err := strconv.ParseBool(profileEnabled)
		if err != nil {
			return nil, errors.New("Invalid bool format")
		}
		return nil, node.AddProfile(profileName, enabled)

	case "DeleteProfile":
		if len(call.Args) < 1 {
			return nil, errors.New("Invalid argument count")
		}
		profileName, ok := call.Args[0].(string)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		return nil, node.DeleteProfile(profileName)

	case "LoadProfile":
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

	case "GetPeer":
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

	case "GetPeers":
		if peers, err := node.GetPeers(); err != nil {
			var policy = ""
			if len(call.Args) > 1 {
				return nil, errors.New("Invalid argument count")
			}
			if len(call.Args) > 0 {
				var ok bool
				policy, ok = call.Args[0].(string)
				if !ok {
					return nil, errors.New("Invalid argument")
				}
			}
			return nil, err
		} else {
			return peers, nil
		}

	case "AddPeer":
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
		if len(call.Args) > 3 {
			policy, ok := call.Args[3].(string)
			if !ok {
				return nil, errors.New("Invalid argument")
			}
			return nil, node.AddPeer(peerName, enabled, peerURI, policy)
		}
		return nil, node.AddPeer(peerName, enabled, peerURI)

	case "DeletePeer":
		if len(call.Args) < 1 {
			return nil, errors.New("Invalid argument count")
		}
		peerName, ok := call.Args[0].(string)
		if !ok {
			return nil, errors.New("Invalid argument")
		}
		return nil, node.DeletePeer(peerName)

	case "Send":
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

	case "SendChannel":
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
