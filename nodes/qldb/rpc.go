package qldb

import (
	//	"encoding/json"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/ratnet/api"
)

// PublicRPC : Entrypoint for RPC functions that are exposed to the public/Internet
func (node *Node) PublicRPC(method string, args ...string) (string, error) {
	switch method {
	case "ID":
		var i bc.PubKey
		i, err := node.ID()
		if err != nil {
			return "", err
		}
		return i.ToB64(), nil

	case "Pickup":
		if len(args) < 2 {
			return "", errors.New("Invalid argument count")
		}

		var i int64
		i, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return "", errors.New("Invalid argument")
		}

		rpk := node.routingKey.GetPubKey().Clone()
		err = rpk.FromB64(args[0])
		if err != nil {
			return "", errors.New("Invalid argument")
		}

		b, err := node.Pickup(rpk, i, args[2:]...)
		if err != nil {
			return "", errors.New("Invalid argument")
		}

		var j []byte
		if j, err = json.Marshal(b); err != nil {
			return "", err
		}
		return string(j), err

	case "Dropoff":
		if len(args) < 1 {
			return "", errors.New("Invalid argument count")
		}
		var bundle api.Bundle
		if err := json.Unmarshal([]byte(args[0]), &bundle); err != nil {
			return "", errors.New("JSON Decode Error in Dropoff RPC Bundle Unpack")
		}
		return "", node.Dropoff(bundle)

	default:
		return "", errors.New("No such method: " + method)
	}
}

// AdminRPC : Entrypoint for administrative RPC functions that should not be exposed to the Internet
func (node *Node) AdminRPC(method string, args ...string) (string, error) {
	switch method {

	case "CID":
		b, err := node.CID()
		if err != nil {
			return "", err
		}
		return b.ToB64(), nil

	case "GetContact":
		if len(args) < 1 {
			return "", errors.New("Invalid argument count")
		}
		if peer, err := node.GetContact(args[0]); err != nil {
			return "", err
		} else if j, err := json.Marshal(peer); err != nil {
			return "", err
		} else {
			return string(j), nil
		}
	case "GetContacts":
		if peers, err := node.GetContacts(); err != nil {
			return "", err
		} else if j, err := json.Marshal(peers); err != nil {
			return "", err
		} else {
			return string(j), nil
		}
	case "AddContact":
		if len(args) < 2 {
			return "", errors.New("Invalid argument count")
		}
		return "", node.AddContact(args[0], args[1])
	case "DeleteContact":
		if len(args) < 1 {
			return "", errors.New("Invalid argument count")
		}
		return "", node.DeleteContact(args[0])

	case "GetChannel":
		if len(args) < 1 {
			return "", errors.New("Invalid argument count")
		}
		if c, err := node.GetChannel(args[0]); err != nil {
			return "", err
		} else if j, err := json.Marshal(c); err != nil {
			return "", err
		} else {
			return string(j), nil
		}
	case "GetChannels":
		if chans, err := node.GetChannels(); err != nil {
			return "", err
		} else if j, err := json.Marshal(chans); err != nil {
			return "", err
		} else {
			return string(j), nil
		}
	case "AddChannel":
		if len(args) < 2 {
			return "", errors.New("Invalid argument count")
		}
		return "", node.AddChannel(args[0], args[1])
	case "DeleteChannel":
		if len(args) < 1 {
			return "", errors.New("Invalid argument count")
		}
		return "", node.DeleteChannel(args[0])

	case "GetProfile":
		if len(args) < 1 {
			return "", errors.New("Invalid argument count")
		}
		if profile, err := node.GetProfile(args[0]); err != nil {
			return "", err
		} else if j, err := json.Marshal(profile); err != nil {
			return "", err
		} else {
			return string(j), nil
		}
	case "GetProfiles":
		if profiles, err := node.GetProfiles(); err != nil {
			return "", err
		} else if j, err := json.Marshal(profiles); err != nil {
			return "", err
		} else {
			return string(j), nil
		}
	case "AddProfile":
		if len(args) < 2 {
			return "", errors.New("Invalid argument count")
		}
		enabled, err := strconv.ParseBool(args[1])
		if err != nil {
			return "", errors.New("Invalid bool format")
		}
		return "", node.AddProfile(args[0], enabled)
	case "DeleteProfile":
		if len(args) < 1 {
			return "", errors.New("Invalid argument count")
		}
		return "", node.DeleteProfile(args[0])
	case "LoadProfile":
		if len(args) < 1 {
			return "", errors.New("Invalid argument count")
		}
		p, err := node.LoadProfile(args[0])
		if err != nil {
			return "", err
		}
		return p.ToB64(), nil

	case "GetPeer":
		if len(args) < 1 {
			return "", errors.New("Invalid argument count")
		}
		a := args[0]
		if peer, err := node.GetPeer(a); err != nil {
			return "", err
		} else if j, err := json.Marshal(peer); err != nil {
			return "", err
		} else {
			return string(j), nil
		}
	case "GetPeers":
		if peers, err := node.GetPeers(); err != nil {
			return "", err
		} else if j, err := json.Marshal(peers); err != nil {
			return "", err
		} else {
			return string(j), nil
		}
	case "AddPeer":
		if len(args) < 1 {
			return "", errors.New("Invalid argument count")
		}
		enabled, err := strconv.ParseBool(args[2])
		if err != nil {
			return "", errors.New("Invalid bool format")
		}
		return "", node.AddPeer(args[0], enabled, args[1])
	case "DeletePeer":
		if len(args) < 1 {
			return "", errors.New("Invalid argument count")
		}
		return "", node.DeletePeer(args[0])

	case "Send":
		if len(args) < 2 {
			return "", errors.New("Invalid argument count")
		}
		msg, err := base64.StdEncoding.DecodeString(args[1])
		if err != nil {
			return "", err
		}
		if len(args) > 2 {
			k := node.contentKey.GetPubKey().Clone()
			if err := k.FromB64(args[2]); err != nil {
				return "", err
			}
			return "", node.Send(args[0], msg, k)
		}
		return "", node.Send(args[0], msg)

	case "SendChannel":
		if len(args) < 2 {
			return "", errors.New("Invalid argument count")
		}
		msg, err := base64.StdEncoding.DecodeString(args[1])
		if err != nil {
			return "", err
		}
		if len(args) > 2 {
			k := node.contentKey.GetPubKey().Clone()
			if err := k.FromB64(args[2]); err != nil {
				return "", err
			}
			return "", node.SendChannel(args[0], msg, k)
		}
		return "", node.SendChannel(args[0], msg)

	default:
		return node.PublicRPC(method, args...)
	}
}
