package ram

import (
	"encoding/json"
	"errors"

	"github.com/awgh/bencrypt"
	"github.com/awgh/ratnet"
	"github.com/awgh/ratnet/api"
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
	ContentKey  string
	ContentType string
	RoutingKey  string
	RoutingType string
	Profiles    []profilePrivB64
	Contacts    []api.Contact
	Channels    []channelPrivB64
	Peers       []api.Peer
	Router      api.Router
	Policies    []api.Policy
}

type importedNode struct {
	ContentKey  string
	ContentType string
	RoutingKey  string
	RoutingType string
	Profiles    []profilePrivB64
	Contacts    []api.Contact
	Channels    []channelPrivB64
	Peers       []api.Peer
	Router      map[string]interface{}
	Policies    []map[string]interface{}
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
	// setup content and routing keys
	v, ok := bencrypt.KeypairTypes[nj.ContentType]
	if !ok {
		return errors.New("Unknown Content Keypair Type in Import")
	}
	node.contentKey = v()
	v, ok = bencrypt.KeypairTypes[nj.RoutingType]
	if !ok {
		return errors.New("Unknown Routing Keypair Type in Import")
	}
	node.routingKey = v()

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

	node.SetRouter(ratnet.NewRouterFromMap(nj.Router))
	for _, p := range nj.Policies {
		// extract the inner Transport first
		t := p["Transport"].(map[string]interface{})
		trans := ratnet.NewTransportFromMap(node, t)
		pol := ratnet.NewPolicyFromMap(trans, node, p)
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
	nj.ContentType = node.contentKey.GetName()
	nj.RoutingKey = node.routingKey.ToB64()
	nj.RoutingType = node.routingKey.GetName()
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
	return json.MarshalIndent(nj, "", "    ")
}

/* Sample JSON for a node configuration
 {
    "ContentKey": "JsQTwSsZW4srWuX+9iCi5SRCulXSWo3xwFIfbu3y9gtIIUmk8fzloo0Nik1R88mSpJ8ODsn9NzWJ22VQ/xtnnw==",
    "ContentType": "Curve25519,AES-CBC-256,HMAC-SHA-256",
    "RoutingKey": "b6f1o1e51JvMmoerJKWI47ZbYSTO+Pi03dOXvZCYzGbsVJbuoEmqo48Wnxag2GzCVeOrtJZS02jT5Nq3jrpQgQ==",
    "RoutingType": "Curve25519,AES-CBC-256,HMAC-SHA-256",
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
    "Policies": [
        {
            "AdminMode": false,
            "ListenURI": ":20001",
            "Transport": {
                "Certfile": "cert.pem",
                "EccMode": true,
                "Keyfile": "key.pem",
                "type": "https"
            },
            "type": "server"
        },
        {
            "AdminMode": true,
            "ListenURI": "localhost:20002",
            "Transport": {
                "Certfile": "cert.pem",
                "EccMode": true,
                "Keyfile": "key.pem",
                "type": "https"
            },
            "type": "server"
        }
    ]
}
*/
