// +build !no_json

package https

import (
	"encoding/json"

	"github.com/awgh/ratnet"
	"github.com/awgh/ratnet/api"
)

func init() {
	ratnet.Transports["https"] = NewFromMap // register this module by name (for deserialization support)
}

// NewFromMap : Makes a new instance of this transport module from a map of arguments (for deserialization support)
func NewFromMap(node api.Node, t map[string]interface{}) api.Transport {
	var certPem, keyPem string
	eccMode := true

	if _, ok := t["Cert"]; ok {
		certPem = t["Cert"].(string)
	}
	if _, ok := t["Key"]; ok {
		keyPem = t["Key"].(string)
	}
	if _, ok := t["EccMode"]; ok {
		eccMode = t["EccMode"].(bool)
	}
	return New([]byte(certPem), []byte(keyPem), node, eccMode)
}

// MarshalJSON : Create a serialied representation of the config of this module
func (h *Module) MarshalJSON() (b []byte, e error) {
	return json.Marshal(map[string]interface{}{
		"Transport": "https",
		"Cert":      string(h.Cert),
		"Key":       string(h.Key),
		"EccMode":   h.EccMode,
	})
}
