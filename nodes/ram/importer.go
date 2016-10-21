package ram

import (
	"encoding/json"
	"errors"

	"github.com/awgh/ratnet/api"
	"github.com/awgh/ratnet/policy"
	"github.com/awgh/ratnet/transports/https"
	"github.com/awgh/ratnet/transports/udp"
)

type profilePrivB64 struct {
	Name    string
	Privkey string //base64 encoded
	Enabled bool
}
type channelPrivB64 struct {
	Name    string
	Privkey string //base64 encoded
}

type exportedNode struct {
	ContentKey string
	RoutingKey string
	Profiles   []profilePrivB64
	Contacts   []api.Contact
	Channels   []channelPrivB64
	Peers      []api.Peer
	Router     api.Router
	Policies   []api.Policy
}

type importedNode struct {
	ContentKey string
	RoutingKey string
	Profiles   []profilePrivB64
	Contacts   []api.Contact
	Channels   []channelPrivB64
	Peers      []api.Peer
	Router     routerWrapper
	Policies   []map[string]interface{}
}

type routerWrapper struct {
	routerInst api.Router
}

func (r *routerWrapper) UnmarshalJSON(b []byte) error {
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	routerType := m["type"].(string)
	if routerType == "default" {
		dr := api.NewDefaultRouter()
		if err := json.Unmarshal(b, &dr); err != nil {
			return err
		}
		r.routerInst = dr
	}
	return nil
}

// Import : Load a node configuration from a JSON config
func (node *Node) Import(jsonConfig []byte) error {
	restartNode := false
	if node.isRunning {
		node.Stop()
		restartNode = true
	}
	var nj importedNode
	if err := json.Unmarshal(jsonConfig, &nj); err != nil {
		return err
	}
	if len(nj.ContentKey) > 0 {
		if err := node.contentKey.FromB64(nj.ContentKey); err != nil {
			return err
		}
	}
	if len(nj.RoutingKey) > 0 {
		if err := node.routingKey.FromB64(nj.RoutingKey); err != nil {
			return err
		}
	}
	for i := 0; i < len(nj.Channels); i++ {
		if err := node.AddChannel(nj.Channels[i].Name, nj.Channels[i].Privkey); err != nil {
			return err
		}
	}
	for i := 0; i < len(nj.Contacts); i++ {
		if err := node.AddContact(nj.Contacts[i].Name, nj.Contacts[i].Pubkey); err != nil {
			return err
		}
	}
	for i := 0; i < len(nj.Profiles); i++ {
		cp := new(api.ProfilePriv)
		cp.Privkey = node.contentKey.Clone()
		if err := cp.Privkey.FromB64(nj.Profiles[i].Privkey); err != nil {
			return err
		}
		cp.Enabled = nj.Profiles[i].Enabled
		cp.Name = nj.Profiles[i].Name

		node.profiles[cp.Name] = cp
	}
	node.SetRouter(nj.Router.routerInst)

	for _, p := range nj.Policies {
		// extract the inner Transport first
		var trans api.Transport
		t := p["Transport"].(map[string]interface{})
		switch t["type"] {
		case "https":
			certfile := t["Certfile"].(string)
			keyfile := t["Keyfile"].(string)
			eccMode := t["EccMode"].(bool)
			trans = https.New(certfile, keyfile, node, eccMode)
			break
		case "udp":
			trans = udp.New(node)
			break
		default:
			return errors.New("Unknown Transport")
		}

		var pol api.Policy
		switch p["type"].(string) {
		case "poll":
			pol = policy.NewPoll(trans, node)
			break
		case "server":
			listenURI := p["ListenURI"].(string)
			adminMode := p["AdminMode"].(bool)
			pol = policy.NewServer(trans, listenURI, adminMode)
			break
		default:
			return errors.New("Unknown Policy")
		}
		node.policies = append(node.policies, pol)
	}
	if restartNode {
		return node.Start()
	}
	return nil
}

// Export : Save a node configuration to a JSON config
func (node *Node) Export() ([]byte, error) {
	var nj exportedNode
	nj.ContentKey = node.contentKey.ToB64()
	nj.RoutingKey = node.routingKey.ToB64()
	nj.Channels = make([]channelPrivB64, len(node.channels))
	i := 0
	for _, v := range node.channels {
		nj.Channels[i].Name = v.Name
		nj.Channels[i].Privkey = v.Privkey.ToB64()
		i++
	}
	nj.Contacts = make([]api.Contact, len(node.contacts))
	i = 0
	for _, v := range node.contacts {
		nj.Contacts[i].Name = v.Name
		nj.Contacts[i].Pubkey = v.Pubkey
		i++
	}
	nj.Profiles = make([]profilePrivB64, len(node.profiles))
	i = 0
	for _, v := range node.profiles {
		nj.Profiles[i].Name = v.Name
		nj.Profiles[i].Enabled = v.Enabled
		nj.Profiles[i].Privkey = v.Privkey.ToB64()
		i++
	}
	nj.Router = node.router
	nj.Policies = node.policies
	return json.Marshal(nj)
}

/* Sample JSON for a node configuration
{
	"ContentKey": "JsQTwSsZW4srWuX+9iCi5SRCulXSWo3xwFIfbu3y9gtIIUmk8fzloo0Nik1R88mSpJ8ODsn9NzWJ22VQ/xtnnw==",
	"RoutingKey": "b6f1o1e51JvMmoerJKWI47ZbYSTO+Pi03dOXvZCYzGbsVJbuoEmqo48Wnxag2GzCVeOrtJZS02jT5Nq3jrpQgQ==",
	"Profiles": [],
	"Contacts": [],
	"Channels": [],
	"Peers": null,
	"Router": {
		"CheckChannels": true,
		"CheckContent": true,
		"CheckProfiles": false,
		"ForwardConsumedChannels": true,
		"ForwardConsumedContent": false,
		"ForwardConsumedProfiles": false,
		"ForwardUnknownChannels": true,
		"ForwardUnknownContent": true,
		"ForwardUnknownProfiles": false,
		"Patches": {},
		"type": "default"
	},
	"Policies": [ {
		"AdminMode": false,
		"ListenURI": ":20001",
		"Transport": {
			"Certfile": "cert.pem",
			"EccMode": true,
			"Keyfile": "key.pem",
			"type": "https"
		},
		"type": "server"
	}, {
		"AdminMode": true,
		"ListenURI": "localhost:20002",
		"Transport": {
			"Certfile": "cert.pem",
			"EccMode": true,
			"Keyfile": "key.pem",
			"type": "https"
		},
		"type": "server"
	} ]
}
*/
