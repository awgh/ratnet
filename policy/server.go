package policy

import (
	"github.com/awgh/ratnet/api"
)

// Server : defines a Listen-only or 'Server' Connection Policy
//
type Server struct {
	transport api.Transport
	listenURI string
	adminMode bool
}

// RunPolicy : Executes the policy as a goroutine
//
func (s *Server) RunPolicy() error {
	s.transport.Listen(s.listenURI, s.adminMode)
	return nil
}

// Stop : Stops a policy
//
func (s *Server) Stop() {
	s.transport.Stop()
}

// NewServer : Returns a new instance of a Server Connection Policy
//
func NewServer(transport api.Transport, listenURI string, adminMode bool) *Server {
	s := new(Server)
	s.transport = transport
	s.listenURI = listenURI
	s.adminMode = adminMode
	return s
}
