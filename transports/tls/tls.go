package tls

import (
	"bufio"
	"crypto/tls"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/ratnet"
	"github.com/awgh/ratnet/api"
	"github.com/awgh/ratnet/api/events"
)

var cachedSessions map[string]*tls.Conn

func init() {
	ratnet.Transports["tls"] = NewFromMap // register this module by name (for deserialization support)

	cachedSessions = make(map[string]*tls.Conn)
}

// NewFromMap : Makes a new instance of this transport module from a map of arguments (for deserialization support)
func NewFromMap(node api.Node, t map[string]interface{}) api.Transport {
	certfile := "cert.pem"
	keyfile := "key.pem"
	eccMode := true

	if _, ok := t["Certfile"]; ok {
		certfile = t["Certfile"].(string)
	}
	if _, ok := t["KeyFile"]; ok {
		keyfile = t["KeyFile"].(string)
	}
	if _, ok := t["EccMode"]; ok {
		eccMode = t["EccMode"].(bool)
	}
	return New(certfile, keyfile, node, eccMode)
}

// New : Makes a new instance of this transport module
func New(certfile string, keyfile string, node api.Node, eccMode bool) *Module {

	tls := new(Module)

	tls.Certfile = certfile
	tls.Keyfile = keyfile
	tls.node = node
	tls.EccMode = eccMode

	tls.byteLimit = 8000 * 1024 //125000 stable, 150000 was unstable

	return tls
}

// Module : TLS Implementation of a Transport module
type Module struct {
	node      api.Node
	isRunning bool
	wg        sync.WaitGroup
	listeners []net.Listener

	Certfile, Keyfile string
	EccMode           bool

	byteLimit int64
}

// Name : Returns this module's common name, which should be unique
func (*Module) Name() string {
	return "tls"
}

// MarshalJSON : Create a serialied representation of the config of this module
func (h *Module) MarshalJSON() (b []byte, e error) {
	return json.Marshal(map[string]interface{}{
		"Transport": "tls",
		"Certfile":  h.Certfile,
		"Keyfile":   h.Keyfile,
		"EccMode":   h.EccMode})
}

// ByteLimit - get limit on bytes per bundle for this transport
func (h *Module) ByteLimit() int64 { return h.byteLimit }

// SetByteLimit - set limit on bytes per bundle for this transport
func (h *Module) SetByteLimit(limit int64) {
	h.byteLimit = limit
}

// Listen : Server interface
func (h *Module) Listen(listen string, adminMode bool) {
	// make sure we are not already running
	if h.isRunning {
		events.Warning(h.node, "This listener is already running.")
		return
	}

	// init ssl components
	bc.InitSSL(h.Certfile, h.Keyfile, h.EccMode)
	cert, err := tls.LoadX509KeyPair(h.Certfile, h.Keyfile)
	if err != nil {
		events.Error(h.node, err.Error())
		return
	}

	// setup Listener
	listener, err := net.Listen("tcp", listen)
	if err != nil {
		events.Error(h.node, err.Error())
		return
	}

	// transform Listener into TLS Listener
	tlsListener := tls.NewListener(
		listener,
		&tls.Config{Certificates: []tls.Certificate{cert}},
	)

	// add Listener to the Listener pool
	h.listeners = append(h.listeners, listener)
	h.isRunning = true

	h.wg.Add(1)
	go func() {
		defer tlsListener.Close()
		defer h.wg.Done()
		for h.isRunning {
			conn, err := tlsListener.Accept()
			if err != nil {
				events.Error(h.node, err.Error())
				continue
			}
			go h.handleConnection(conn, h.node, adminMode)
		}
	}()

}

func (h *Module) handleConnection(conn net.Conn, node api.Node, adminMode bool) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	for h.isRunning { // read multiple messages on the same connection
		var a api.RemoteCall

		//use default gob encoder
		dec := gob.NewDecoder(reader)
		if err := dec.Decode(&a); err != nil {
			events.Warning(h.node, "tls handleConnection gob decode failed: "+err.Error())
			break
		}

		var err error
		var result interface{}
		if adminMode {
			result, err = node.AdminRPC(h, a)
		} else {
			result, err = node.PublicRPC(h, a)
		}

		rr := api.RemoteResponse{}
		if err != nil {
			rr.Error = err.Error()
		}
		if result != nil { // gob cannot encode typed Nils, only interface{} Nils...wtf?
			rr.Value = result
		}
		enc := gob.NewEncoder(writer)
		if err := enc.Encode(rr); err != nil {
			events.Warning(h.node, "tls handleConnection gob encode failed: "+err.Error())
			break
		}
		writer.Flush()
	}
}

// RPC : client interface
func (h *Module) RPC(host string, method string, args ...interface{}) (interface{}, error) {

	events.Info(h.node, fmt.Sprintf("\n***\n***RPC %s on %s called with: %+v\n***\n", method, host, args))

	conn, ok := cachedSessions[host]
	if !ok {
		var err error
		conf := &tls.Config{InsecureSkipVerify: true}
		conn, err = tls.Dial("tcp", host, conf)
		if err != nil {
			events.Error(h.node, err.Error())
			return nil, err
		}
		cachedSessions[host] = conn
	}
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	var a api.RemoteCall
	a.Action = method
	a.Args = args

	//use default gob encoder
	enc := gob.NewEncoder(writer)
	if err := enc.Encode(a); err != nil {
		events.Warning(h.node, "tls rpc gob encode failed: "+err.Error())
		delete(cachedSessions, host) // something's wrong, make a new session next attempt
		_ = conn.Close()
		return nil, err
	}
	writer.Flush()
	var rr api.RemoteResponse
	dec := gob.NewDecoder(reader)
	if err := dec.Decode(&rr); err != nil {
		events.Warning(h.node, "tls rpc gob decode failed: "+err.Error())
		delete(cachedSessions, host) // something's wrong, make a new session next attempt
		_ = conn.Close()
		return nil, err
	}

	if rr.IsErr() {
		return nil, errors.New(rr.Error)
	}
	if rr.IsNil() {
		return nil, nil
	}
	return rr.Value, nil
}

// Stop : stops the TLS transport from running
func (h *Module) Stop() {
	h.isRunning = false
	for _, listener := range h.listeners {
		listener.Close()
	}
	h.wg.Wait()
	for _, v := range cachedSessions {
		_ = v.Close()
	}
}
