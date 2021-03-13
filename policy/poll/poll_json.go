// +build !no_json

package poll

import (
	"encoding/json"

	"github.com/awgh/ratnet"
	"github.com/awgh/ratnet/api"
)

func init() {
	ratnet.Policies["poll"] = NewFromMap // register this module by name (for deserialization support)
}

// NewFromMap : Makes a new instance of this transport module from a map of arguments (for deserialization support)
func NewFromMap(transport api.Transport, node api.Node,
	t map[string]interface{}) api.Policy {
	interval := int(t["Interval"].(float64))
	jitter := int(t["Jitter"].(float64))
	var groups []string
	gi := []interface{}(t["Groups"].([]interface{}))
	for _, g := range gi {
		gstr := string(g.(string))
		groups = append(groups, gstr)
	}

	// groups :=
	return New(transport, node, interval, jitter, groups...)
}

// MarshalJSON : Create a serialied representation of the config of this policy
func (p *Poll) MarshalJSON() (b []byte, e error) {
	return json.Marshal(map[string]interface{}{
		"Policy":    "poll",
		"Transport": p.Transport,
		"Interval":  p.GetInterval(),
		"Jitter":    p.GetJitter(),
		"Groups":    p.Groups,
	})
}
