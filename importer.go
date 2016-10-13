package ratnet

import (
	"encoding/json"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/ratnet/api"
)

type profilePrivB64 struct {
	Name    string
	Privkey string //base64 encoded
	Enabled bool
}
type channel struct {
	Name    string
	Privkey string
}
type importedNode struct {
	Profiles []profilePrivB64
	Contacts []api.Contact
	Channels []channel
	Peers    []api.Peer
}

// Import : Load a node configuration from a JSON config
func Import(node api.Node, jsonConfig []byte, crypto bc.KeyPair) error {
	var nj importedNode
	if err := json.Unmarshal(jsonConfig, &nj); err != nil {
		return err
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
/*
		cp := new(api.ProfilePriv)
		cp.Privkey = crypto.Clone()
		if err := cp.Privkey.FromB64(nj.Profiles[i].Privkey); err != nil {
			return err
		}
		cp.Enabled = nj.Profiles[i].Enabled
		cp.Name = nj.Profiles[i].Name

		if err := node.AddProfile(); err != nil {
			return err
		}
		*/
	}
	return nil
}

/* Sample JSON for a node configuration
	   {  // no way to set routing key manually at the moment
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
