// +build !no_json

package server

import (
	"encoding/json"

	"github.com/awgh/ratnet"
	"github.com/awgh/ratnet/api"
)

func init() {
	ratnet.Policies["server"] = NewFromMap // register this module by name (for deserialization support)
}

// NewFromMap : Makes a new instance of this transport module from a map of arguments (for deserialization support)
func NewFromMap(transport api.Transport, node api.Node, p map[string]interface{}) api.Policy {
	listenURI := p["ListenURI"].(string)
	adminMode := p["AdminMode"].(bool)
	return New(transport, listenURI, adminMode)
}

// MarshalJSON : Create a serialied representation of the config of this policy
func (s *Server) MarshalJSON() (b []byte, e error) {
	return json.Marshal(map[string]interface{}{
		"Policy":    "server",
		"ListenURI": s.ListenURI,
		"AdminMode": s.AdminMode,
		"Transport": s.Transport,
	})
}
