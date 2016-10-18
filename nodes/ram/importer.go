package ram

import (
	"encoding/json"

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
type importedNode struct {
	ContentKey string
	RoutingKey string
	Profiles   []profilePrivB64
	Contacts   []api.Contact
	Channels   []channelPrivB64
	Peers      []api.Peer
	Router     api.Router
}

// Import : Load a node configuration from a JSON config
func (node *Node) Import(jsonConfig []byte) error {
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
	if nj.Router != nil {
		node.SetRouter(nj.Router)
	}
	return nil
}

// Export : Save a node configuration to a JSON config
func (node *Node) Export() ([]byte, error) {
	var nj importedNode

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

	return json.Marshal(nj)
}

/* Sample JSON for a node configuration
	   {
		   "ContentKey":"CPRIVKEY_b64",
	       "RoutingKey":"RPRIVKEY_b64",

           "Profiles": [
           		{
           			"Name:"NAME",
	           		"Privkey":"CPRIVKEY_b64",
	           		"Enabled": true
	           	}
           ],
           "Contacts": [
           		{
           			"Name":DST_NAME",
           			"Pubkey":"DST_CPUBKEY_b64"
           		}
           ],
           "Channels": [
           		{
           			"Name":CHAN_NAME",
           			"Privkey":"CHAN_PRIVKEY_b64"
           		}
           ],
           "Peers": [
           		{
	           		"Name":"SRV_NAME",
	           		"URI":"SRV_URI",
	           		"Enabled": true,
	           		"Pubkey":"SRV_PUBKEY_b64"
	           	}
           	]
	   }
*/
