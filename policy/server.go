package policy

import (
	"encoding/json"

	"github.com/awgh/ratnet"
	"github.com/awgh/ratnet/api"
)

// Server : defines a Listen-only or 'Server' Connection Policy
//
type Server struct {
	Transport api.Transport
	ListenURI string
	AdminMode bool
}

func init() {
	ratnet.Policies["server"] = NewFromMap // register this module by name (for deserialization support)
}

// NewFromMap : Makes a new instance of this transport module from a map of arguments (for deserialization support)
func NewFromMap(transport api.Transport, node api.Node, p map[string]interface{}) api.Policy {
	listenURI := p["ListenURI"].(string)
	adminMode := p["AdminMode"].(bool)
	return NewServer(transport, listenURI, adminMode)
}

// NewServer : Returns a new instance of a Server Connection Policy
//
func NewServer(transport api.Transport, listenURI string, adminMode bool) *Server {
	s := new(Server)
	s.Transport = transport
	s.ListenURI = listenURI
	s.AdminMode = adminMode
	return s
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

// RunPolicy : Executes the policy as a goroutine
//
func (s *Server) RunPolicy() error {
	s.Transport.Listen(s.ListenURI, s.AdminMode)
	return nil
}

// Stop : Stops a policy
//
func (s *Server) Stop() {
	s.Transport.Stop()
}

// GetTransport : Returns the transports associated with this policy
//
func (s *Server) GetTransport() api.Transport {
	return s.Transport
}
