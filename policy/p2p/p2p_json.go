// +build !no_json

package p2p

import (
	"encoding/json"

	"github.com/awgh/ratnet"
	"github.com/awgh/ratnet/api"
)

func init() {
	ratnet.Policies["p2p"] = NewFromMap // register this module by name (for deserialization support)
}

// NewFromMap : Makes a new instance of this transport module from a map of arguments (for deserialization support)
func NewFromMap(transport api.Transport, node api.Node, p map[string]interface{}) api.Policy {
	listenURI := p["ListenURI"].(string)
	adminMode := p["AdminMode"].(bool)
	listenInterval := p["ListenInterval"].(int)
	advertiseInterval := p["AdvertiseInterval"].(int)
	return New(transport, listenURI, node, adminMode, listenInterval, advertiseInterval)
}

// MarshalJSON : Create a serialied representation of the config of this policy
func (s *P2P) MarshalJSON() (b []byte, e error) {
	return json.Marshal(map[string]interface{}{
		"Policy":            "p2p",
		"ListenURI":         s.ListenURI,
		"AdminMode":         s.AdminMode,
		"Transport":         s.Transport,
		"ListenInterval":    s.ListenInterval,
		"AdvertiseInterval": s.AdvertiseInterval,
	})
}
