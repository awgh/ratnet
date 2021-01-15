package api

// Action - API Call ID numbers
type Action uint8

const (
	Null          Action = 0
	ID            Action = 1
	Dropoff       Action = 2
	Pickup        Action = 3
	CID           Action = 16
	GetContact    Action = 17
	GetContacts   Action = 18
	AddContact    Action = 19
	DeleteContact Action = 20
	GetChannel    Action = 21
	GetChannels   Action = 22
	AddChannel    Action = 23
	DeleteChannel Action = 24
	GetProfile    Action = 25
	GetProfiles   Action = 26
	AddProfile    Action = 27
	DeleteProfile Action = 28
	LoadProfile   Action = 29
	GetPeer       Action = 30
	GetPeers      Action = 31
	AddPeer       Action = 32
	DeletePeer    Action = 33
	Send          Action = 34
	SendChannel   Action = 35
)

/*
// ActionToUint16 - returns the integer code for a string action name
func ActionToUint16(action string) uint16 {
	switch action {
	case "ID":
		return ID
	case "Dropoff":
		return Dropoff
	case "Pickup":
		return Pickup
	case "CID":
		return CID
	case "GetContact":
		return GetContact
	case "GetContacts":
		return GetContacts
	case "AddContact":
		return AddContact
	case "DeleteContact":
		return DeleteContact
	case "GetChannel":
		return GetChannel
	case "GetChannels":
		return GetChannels
	case "AddChannel":
		return AddChannel
	case "DeleteChannel":
		return DeleteChannel
	case "GetProfile":
		return GetProfile
	case "GetProfiles":
		return GetProfiles
	case "AddProfile":
		return AddProfile
	case "DeleteProfile":
		return DeleteProfile
	case "LoadProfile":
		return LoadProfile
	case "GetPeer":
		return GetPeer
	case "GetPeers":
		return GetPeers
	case "AddPeer":
		return AddPeer
	case "DeletePeer":
		return DeletePeer
	case "Send":
		return Send
	case "SendChannel":
		return SendChannel
	}
	return Null
}

// ActionFromUint16 - returns the string name for an integer action code
func ActionFromUint16(action uint16) string {
	switch action {
	case ID:
		return "ID"
	case Dropoff:
		return "Dropoff"
	case Pickup:
		return "Pickup"
	case CID:
		return "CID"
	case GetContact:
		return "GetContact"
	case GetContacts:
		return "GetContacts"
	case AddContact:
		return "AddContact"
	case DeleteContact:
		return "DeleteContact"
	case GetChannel:
		return "GetChannel"
	case GetChannels:
		return "GetChannels"
	case AddChannel:
		return "AddChannel"
	case DeleteChannel:
		return "DeleteChannel"
	case GetProfile:
		return "GetProfile"
	case GetProfiles:
		return "GetProfiles"
	case AddProfile:
		return "AddProfile"
	case DeleteProfile:
		return "DeleteProfile"
	case LoadProfile:
		return "LoadProfile"
	case GetPeer:
		return "GetPeer"
	case GetPeers:
		return "GetPeers"
	case AddPeer:
		return "AddPeer"
	case DeletePeer:
		return "DeletePeer"
	case Send:
		return "Send"
	case SendChannel:
		return "SendChannel"
	}
	return ""
}
*/
