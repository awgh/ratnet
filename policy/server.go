package policy

import (
	"encoding/json"

	"github.com/awgh/ratnet/api"
)

// Server : defines a Listen-only or 'Server' Connection Policy
//
type Server struct {
	Transport api.Transport
	ListenURI string
	AdminMode bool
}

// MarshalJSON : Create a serialied representation of the config of this policy
func (s *Server) MarshalJSON() (b []byte, e error) {
	return json.Marshal(map[string]interface{}{
		"type":      "server",
		"ListenURI": s.ListenURI,
		"AdminMode": s.AdminMode,
		"Transport": s.Transport})
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

// NewServer : Returns a new instance of a Server Connection Policy
//
func NewServer(transport api.Transport, listenURI string, adminMode bool) *Server {
	s := new(Server)
	s.Transport = transport
	s.ListenURI = listenURI
	s.AdminMode = adminMode
	return s
}
