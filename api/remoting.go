package api

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"

	"github.com/awgh/bencrypt/bc"

	"github.com/awgh/bencrypt/ecc"
	"github.com/awgh/bencrypt/rsa"
)

// API Call ID numbers
const (
	APINull          = 0
	APIID            = 1
	APIDropoff       = 2
	APIPickup        = 3
	APICID           = 16
	APIGetContact    = 17
	APIGetContacts   = 18
	APIAddContact    = 19
	APIDeleteContact = 20
	APIGetChannel    = 21
	APIGetChannels   = 22
	APIAddChannel    = 23
	APIDeleteChannel = 24
	APIGetProfile    = 25
	APIGetProfiles   = 26
	APIAddProfile    = 27
	APIDeleteProfile = 28
	APILoadProfile   = 29
	APIGetPeer       = 30
	APIGetPeers      = 31
	APIAddPeer       = 32
	APIDeletePeer    = 33
	APISend          = 34
	APISendChannel   = 35
)

// API Parameter Data types
const (
	APITypeNil    byte = 0x0
	APITypeInt64  byte = 0x1
	APITypeString byte = 0x2
	APITypeBytes  byte = 0x3

	APITypePubKeyECC byte = 0x10
	APITypePubKeyRSA byte = 0x11

	APITypeContactArray byte = 0x20
	APITypeChannelArray byte = 0x21
	APITypeProfileArray byte = 0x22
	APITypePeerArray    byte = 0x23

	APITypeContact byte = 0x30
	APITypeChannel byte = 0x31
	APITypeProfile byte = 0x32
	APITypePeer    byte = 0x33

	APITypeBundle byte = 0x40
)

// RemoteCall : defines a Remote Procedure Call
type RemoteCall struct {
	Action string
	Args   []interface{}
}

// RemoteResponse : defines a response returned from a Remote Procedure Call
type RemoteResponse struct {
	Error string
	Value interface{}
}

// IsNil - is this response Nil?
func (r *RemoteResponse) IsNil() bool { return r.Value == nil }

// IsErr - is this response an error?
func (r *RemoteResponse) IsErr() bool { return r.Error != "" }

// ActionToUint16 - returns the integer code for a string action name
func ActionToUint16(action string) uint16 {
	switch action {
	case "ID":
		return APIID
	case "Dropoff":
		return APIDropoff
	case "Pickup":
		return APIPickup
	case "CID":
		return APICID
	case "GetContact":
		return APIGetContact
	case "GetContacts":
		return APIGetContacts
	case "AddContact":
		return APIAddContact
	case "DeleteContact":
		return APIDeleteContact
	case "GetChannel":
		return APIGetChannel
	case "GetChannels":
		return APIGetChannels
	case "AddChannel":
		return APIAddChannel
	case "DeleteChannel":
		return APIDeleteChannel
	case "GetProfile":
		return APIGetProfile
	case "GetProfiles":
		return APIGetProfiles
	case "AddProfile":
		return APIAddProfile
	case "DeleteProfile":
		return APIDeleteProfile
	case "LoadProfile":
		return APILoadProfile
	case "GetPeer":
		return APIGetPeer
	case "GetPeers":
		return APIGetPeers
	case "AddPeer":
		return APIAddPeer
	case "DeletePeer":
		return APIDeletePeer
	case "Send":
		return APISend
	case "SendChannel":
		return APISendChannel
	}
	return APINull
}

// ActionFromUint16 - returns the string name for an integer action code
func ActionFromUint16(action uint16) string {
	switch action {
	case APIID:
		return "ID"
	case APIDropoff:
		return "Dropoff"
	case APIPickup:
		return "Pickup"
	case APICID:
		return "CID"
	case APIGetContact:
		return "GetContact"
	case APIGetContacts:
		return "GetContacts"
	case APIAddContact:
		return "AddContact"
	case APIDeleteContact:
		return "DeleteContact"
	case APIGetChannel:
		return "GetChannel"
	case APIGetChannels:
		return "GetChannels"
	case APIAddChannel:
		return "AddChannel"
	case APIDeleteChannel:
		return "DeleteChannel"
	case APIGetProfile:
		return "GetProfile"
	case APIGetProfiles:
		return "GetProfiles"
	case APIAddProfile:
		return "AddProfile"
	case APIDeleteProfile:
		return "DeleteProfile"
	case APILoadProfile:
		return "LoadProfile"
	case APIGetPeer:
		return "GetPeer"
	case APIGetPeers:
		return "GetPeers"
	case APIAddPeer:
		return "AddPeer"
	case APIDeletePeer:
		return "DeletePeer"
	case APISend:
		return "Send"
	case APISendChannel:
		return "SendChannel"
	}
	return ""
}

// ArgsToBytes - converts an interface array to a byte array
func ArgsToBytes(args []interface{}) []byte {
	b := bytes.NewBuffer([]byte{})
	w := bufio.NewWriter(b)
	for _, i := range args {
		serialize(w, i)
	}
	w.Flush()
	return b.Bytes()
}

// ArgsFromBytes - converts a byte array to an interface array
func ArgsFromBytes(args []byte) ([]interface{}, error) {
	var output []interface{}
	r := bufio.NewReader(bytes.NewBuffer(args))

	for i := 0; i < len(args); i++ {
		// read a TLV field, add it to output array
		t, v, err := readTLV(r)
		if err != nil {
			return nil, err
		}
		i += 3 + len(v)
		b := bytes.NewBuffer(v)
		rt, err := deserialize(b, t, v)
		if err != nil {
			return nil, err
		}
		output = append(output, rt)
	}
	return output, nil
}

// Serialization byte order is BigEndian / network-order

// RemoteCallToBytes - converts a RemoteCall to a byte array
func RemoteCallToBytes(call *RemoteCall) []byte {
	b := bytes.NewBuffer([]byte{})
	w := bufio.NewWriter(b)
	// Action - bytes [0-1] uint16
	binary.Write(w, binary.BigEndian, ActionToUint16(call.Action))
	// Args - everything else
	binary.Write(w, binary.BigEndian, ArgsToBytes(call.Args))
	w.Flush()
	return b.Bytes()
}

// RemoteCallFromBytes - converts a RemoteCall from a byte array
func RemoteCallFromBytes(input []byte) (*RemoteCall, error) {
	if len(input) < 2 {
		return nil, errors.New("Input array too short")
	}
	call := new(RemoteCall)
	action := binary.BigEndian.Uint16(input[:2])
	call.Action = ActionFromUint16(action)
	args, err := ArgsFromBytes(input[2:])
	if err != nil {
		return nil, err
	}
	call.Args = args
	return call, nil
}

// RemoteResponseToBytes - converts a RemoteResponse to a byte array
func RemoteResponseToBytes(resp *RemoteResponse) []byte {
	b := bytes.NewBuffer([]byte{})
	w := bufio.NewWriter(b)

	writeTLV(w, APITypeString, []byte(resp.Error))

	serialize(w, resp.Value)
	w.Flush()
	return b.Bytes()
}

// RemoteResponseFromBytes - converts a RemoteResponse from a byte array
func RemoteResponseFromBytes(input []byte) (*RemoteResponse, error) {
	resp := new(RemoteResponse)
	r := bufio.NewReader(bytes.NewBuffer(input))

	// read the two TLV fields, add to struct
	// Error string
	t, v, err := readTLV(r)
	if err != nil {
		return nil, err
	}
	resp.Error = string(v)

	// Value interface{}
	t, v, err = readTLV(r)
	if err != nil {
		return nil, err
	}
	rv, err := deserialize(r, t, v)
	if err != nil {
		return nil, err
	}
	resp.Value = rv
	return resp, nil
}

func serialize(w io.Writer, v interface{}) {
	switch v.(type) {
	case int64:
		binary.Write(w, binary.BigEndian, APITypeInt64) //type
		binary.Write(w, binary.BigEndian, uint16(8))    //length
		binary.Write(w, binary.BigEndian, v)            //value
	case string:
		s := v.(string)
		writeTLV(w, APITypeString, []byte(s))
	case []byte:
		ba := v.([]byte)
		writeTLV(w, APITypeBytes, ba)
	case bc.PubKey:
		pk, ok := v.(*ecc.PubKey)
		var kb []byte
		var typ byte
		if ok {
			kb = pk.ToBytes()
			typ = APITypePubKeyECC
		} else {
			rk := v.(*rsa.PubKey)
			kb = rk.ToBytes()
			typ = APITypePubKeyRSA
		}
		writeTLV(w, typ, kb)
	case *Contact:
		ap := v.(*Contact)
		b := bytes.NewBuffer([]byte{})
		writeLV(b, []byte(ap.Name))
		writeLV(b, []byte(ap.Pubkey))
		writeTLV(w, APITypeContact, b.Bytes())
	case []Contact:
		ac := v.([]Contact)
		b := bytes.NewBuffer([]byte{})
		for _, c := range ac {
			writeLV(b, []byte(c.Name))
			writeLV(b, []byte(c.Pubkey))
		}
		writeTLV(w, APITypeContactArray, b.Bytes())
	case *Channel:
		ap := v.(*Channel)
		b := bytes.NewBuffer([]byte{})
		writeLV(b, []byte(ap.Name))
		writeLV(b, []byte(ap.Pubkey))
		writeTLV(w, APITypeChannel, b.Bytes())
	case []Channel:
		ac := v.([]Channel)
		b := bytes.NewBuffer([]byte{})
		for _, c := range ac {
			writeLV(b, []byte(c.Name))
			writeLV(b, []byte(c.Pubkey))
		}
		writeTLV(w, APITypeChannelArray, b.Bytes())
	case *Profile:
		ap := v.(*Profile)
		b := bytes.NewBuffer([]byte{})
		writeLV(b, []byte(ap.Name))
		writeLV(b, []byte(ap.Pubkey))
		if ap.Enabled {
			b.WriteByte(1)
		} else {
			b.WriteByte(0)
		}
		writeTLV(w, APITypeProfile, b.Bytes())
	case []Profile:
		ac := v.([]Profile)
		b := bytes.NewBuffer([]byte{})
		for _, c := range ac {
			writeLV(b, []byte(c.Name))
			writeLV(b, []byte(c.Pubkey))
			if c.Enabled {
				b.WriteByte(1)
			} else {
				b.WriteByte(0)
			}
		}
		writeTLV(w, APITypeProfileArray, b.Bytes())

	case *Peer:
		ap := v.(*Peer)
		b := bytes.NewBuffer([]byte{})
		writeLV(b, []byte(ap.Name))
		writeLV(b, []byte(ap.Group))
		writeLV(b, []byte(ap.URI))
		if ap.Enabled {
			b.WriteByte(1)
		} else {
			b.WriteByte(0)
		}
		writeTLV(w, APITypePeer, b.Bytes())
	case []Peer:
		ac := v.([]Peer)
		b := bytes.NewBuffer([]byte{})
		for _, c := range ac {
			writeLV(b, []byte(c.Name))
			writeLV(b, []byte(c.Group))
			writeLV(b, []byte(c.URI))
			if c.Enabled {
				b.WriteByte(1)
			} else {
				b.WriteByte(0)
			}
		}
		writeTLV(w, APITypePeerArray, b.Bytes())
	case Bundle:
		bundle := v.(Bundle)
		b := bytes.NewBuffer([]byte{})
		writeLV(b, bundle.Data)
		binary.Write(b, binary.BigEndian, bundle.Time)
		writeTLV(w, APITypeBundle, b.Bytes())
	}
}

func deserialize(r io.Reader, t byte, v []byte) (interface{}, error) {
	switch t {
	case APITypeNil:
		return nil, nil
	case APITypeInt64:
		var vint int64
		if err := binary.Read(r, binary.BigEndian, &vint); err != nil {
			return nil, err
		}
		return vint, nil
	case APITypeString:
		return string(v), nil
	case APITypeBytes:
		return v, nil
	case APITypePubKeyECC:
		key := new(ecc.PubKey)
		if err := key.FromBytes(v); err != nil {
			return nil, err
		}
		return key, nil
	case APITypePubKeyRSA:
		key := new(rsa.PubKey)
		if err := key.FromBytes(v); err != nil {
			return nil, err
		}
		return key, nil

	case APITypeContact:
		var contact Contact
		b := bytes.NewBuffer(v)
		va, err := readLV(b)
		if err != nil {
			return nil, err
		}
		contact.Name = string(va)
		va, err = readLV(b)
		if err != nil {
			return nil, err
		}
		contact.Pubkey = string(va)
		return &contact, nil

	case APITypeContactArray:
		bytesRead := 0
		var contacts []Contact
		b := bytes.NewBuffer(v)
		for bytesRead < len(v) {
			var contact Contact
			va, err := readLV(b)
			if err != nil {
				return nil, err
			}
			bytesRead += len(va) + 2
			contact.Name = string(va)
			va, err = readLV(b)
			if err != nil {
				return nil, err
			}
			bytesRead += len(va) + 2
			contact.Pubkey = string(va)
			contacts = append(contacts, contact)
		}
		return contacts, nil

	case APITypeChannel:
		var channel Channel
		b := bytes.NewBuffer(v)
		va, err := readLV(b)
		if err != nil {
			return nil, err
		}
		channel.Name = string(va)
		va, err = readLV(b)
		if err != nil {
			return nil, err
		}
		channel.Pubkey = string(va)
		return &channel, nil

	case APITypeChannelArray:
		bytesRead := 0
		var channels []Channel
		b := bytes.NewBuffer(v)
		for bytesRead < len(v) {
			var channel Channel
			va, err := readLV(b)
			if err != nil {
				return nil, err
			}
			bytesRead += len(va) + 2
			channel.Name = string(va)
			va, err = readLV(b)
			if err != nil {
				return nil, err
			}
			bytesRead += len(va) + 2
			channel.Pubkey = string(va)
			channels = append(channels, channel)
		}
		return channels, nil

	case APITypeProfile:
		var profile Profile
		b := bytes.NewBuffer(v)
		va, err := readLV(b)
		if err != nil {
			return nil, err
		}
		profile.Name = string(va)
		va, err = readLV(b)
		if err != nil {
			return nil, err
		}
		profile.Pubkey = string(va)
		bt, err := b.ReadByte()
		if err != nil {
			return nil, err
		}
		if bt == 1 {
			profile.Enabled = true
		} else {
			profile.Enabled = false
		}
		return &profile, nil

	case APITypeProfileArray:
		bytesRead := 0
		var profiles []Profile
		b := bytes.NewBuffer(v)
		for bytesRead < len(v) {
			var profile Profile
			va, err := readLV(b)
			if err != nil {
				return nil, err
			}
			bytesRead += len(va) + 2
			profile.Name = string(va)
			va, err = readLV(b)
			if err != nil {
				return nil, err
			}
			bytesRead += len(va) + 2
			profile.Pubkey = string(va)

			bt, err := b.ReadByte()
			if err != nil {
				return nil, err
			}
			bytesRead++
			if bt == 1 {
				profile.Enabled = true
			} else {
				profile.Enabled = false
			}
			profiles = append(profiles, profile)
		}
		return profiles, nil

	case APITypePeer:
		var peer Peer
		b := bytes.NewBuffer(v)
		va, err := readLV(b)
		if err != nil {
			return nil, err
		}
		peer.Name = string(va)
		va, err = readLV(b)
		if err != nil {
			return nil, err
		}
		peer.Group = string(va)
		va, err = readLV(b)
		if err != nil {
			return nil, err
		}
		peer.URI = string(va)
		bt, err := b.ReadByte()
		if err != nil {
			return nil, err
		}
		if bt == 1 {
			peer.Enabled = true
		} else {
			peer.Enabled = false
		}
		return &peer, nil

	case APITypePeerArray:
		bytesRead := 0
		var peers []Peer
		b := bytes.NewBuffer(v)
		for bytesRead < len(v) {
			var peer Peer
			va, err := readLV(b)
			if err != nil {
				return nil, err
			}
			bytesRead += len(va) + 2
			peer.Name = string(va)
			va, err = readLV(b)
			if err != nil {
				return nil, err
			}
			bytesRead += len(va) + 2
			peer.Group = string(va)
			va, err = readLV(b)
			if err != nil {
				return nil, err
			}
			bytesRead += len(va) + 2
			peer.URI = string(va)
			bt, err := b.ReadByte()
			if err != nil {
				return nil, err
			}
			bytesRead++
			if bt == 1 {
				peer.Enabled = true
			} else {
				peer.Enabled = false
			}
			peers = append(peers, peer)
		}
		return peers, nil

	case APITypeBundle:
		var bundle Bundle
		b := bytes.NewBuffer(v)
		data, err := readLV(b)
		if err != nil {
			return nil, err
		}
		bundle.Data = data
		var vint int64
		if err := binary.Read(b, binary.BigEndian, &vint); err != nil {
			return nil, err
		}
		bundle.Time = vint
		return bundle, nil
	}
	return nil, errors.New("Unknown Type")
}

func writeTLV(w io.Writer, typ byte, value []byte) {
	binary.Write(w, binary.BigEndian, typ) //type
	writeLV(w, value)
}

func writeLV(w io.Writer, value []byte) {
	length := uint16(len(value))
	binary.Write(w, binary.BigEndian, length) //length
	w.Write(value)                            //value
}

func readTLV(r io.Reader) (byte, []byte, error) {
	var t byte
	if err := binary.Read(r, binary.BigEndian, &t); err == io.EOF {
		return 0, nil, nil //EOF
	} else if err != nil {
		return t, nil, err
	}
	v, err := readLV(r)
	return t, v, err
}

func readLV(r io.Reader) ([]byte, error) {
	var l uint16
	if err := binary.Read(r, binary.BigEndian, &l); err != nil {
		return nil, err
	}
	if l == 0 {
		return nil, nil
	}
	v := make([]byte, l)
	if err := binary.Read(r, binary.BigEndian, &v); err != nil {
		return nil, err
	}
	return v, nil
}
