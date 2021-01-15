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
