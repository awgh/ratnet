package transports

import (
	"database/sql"
	"errors"
	"log"
	"net/url"
	"ratnet"
	"sync"
)

// SyncHole - a container for transports to hold their wait groups in
var SyncHole = make(map[string]*sync.WaitGroup)

// Servers - every server transport provider module must register here and implement Server
var Servers = make(map[string]Server)

// Clients - every client transport provider module must register here and implement Client
var Clients = make(map[string]Client)

// Server - Implement this in a transport plugin.
type Server interface {
	Accept(listen string, certfile string, keyfile string, db func() *sql.DB, adminMode bool)
	getName() string
}

// NewServer - This uses a transport plugin to start a server
func NewServer(module string, listen string,
	certfile string, keyfile string, database string, adminMode bool) func() *sql.DB {

	s, found := Servers[module]
	if !found {
		log.Fatal("No Server module found for that protocol")
	}
	db := ratnet.BootstrapDB(database)
	s.Accept(listen, certfile, keyfile, db, adminMode)
	return db
}

// Client - Implement this in a transport plugin
type Client interface {
	RemoteAPIImpl(host string, a *ratnet.ApiCall) ([]byte, error)
}

// RemoteAPI - This uses a transport plugin to make a remote API call
func RemoteAPI(host string, a *ratnet.ApiCall) ([]byte, error) {
	u, err := url.Parse(host)
	if err != nil {
		return nil, err
	}
	c, found := Clients[u.Scheme]
	if !found {
		log.Println(host)
		return nil, errors.New("No Client module found for scheme " + u.Scheme)
	}
	return c.RemoteAPIImpl(host, a)
}
