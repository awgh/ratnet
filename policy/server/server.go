package server

import (
	"github.com/awgh/ratnet/api"
)

// Server : defines a Listen-only or 'Server' Connection Policy
//
type Server struct {
	Transport api.Transport
	ListenURI string
	AdminMode bool
}

// New : Returns a new instance of a Server Connection Policy
//
func New(transport api.Transport, listenURI string, adminMode bool) *Server {
	s := new(Server)
	s.Transport = transport
	s.ListenURI = listenURI
	s.AdminMode = adminMode
	return s
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
