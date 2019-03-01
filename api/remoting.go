package api

import (
	"bufio"
	"bytes"
	"encoding/binary"
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
	APITypeInt64  byte = 0x1
	APITypeString byte = 0x2
	APITypeBytes  byte = 0x3
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
		switch i.(type) {
		case int64:
			binary.Write(w, binary.BigEndian, APITypeInt64) //type
			binary.Write(w, binary.BigEndian, uint16(8))    //length
			binary.Write(w, binary.BigEndian, i)            //value
		case string:
			s := i.(string)
			binary.Write(w, binary.BigEndian, APITypeString)  //type
			binary.Write(w, binary.BigEndian, uint16(len(s))) //length
			binary.Write(w, binary.BigEndian, i)              //value
		case []byte:
			ba := i.([]byte)
			binary.Write(w, binary.BigEndian, APITypeBytes)    //type
			binary.Write(w, binary.BigEndian, uint16(len(ba))) //length
			binary.Write(w, binary.BigEndian, i)               //value
		default:
			//return nil, errors.New("Only []byte, string, and int64 can be serialized")
		}
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
		var t byte
		binary.Read(r, binary.BigEndian, &t)
		var l uint16
		binary.Read(r, binary.BigEndian, &l)
		v := make([]byte, l)
		binary.Read(r, binary.BigEndian, &v)
		switch t {
		case APITypeInt64:
			var vint uint64
			binary.Read(r, binary.BigEndian, &vint)
			output = append(output, vint)
			i += 8
		case APITypeString:
			output = append(output, string(v))
			i += len(v)
		case APITypeBytes:
			output = append(output, v)
			i += len(v)
		}
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
