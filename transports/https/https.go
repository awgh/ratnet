package https

import (
	"bytes"
	"crypto/tls"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/awgh/ratnet"
	"github.com/awgh/ratnet/api"
	"github.com/awgh/ratnet/api/events"
)

func init() {
	ratnet.Transports["https"] = NewFromMap // register this module by name (for deserialization support)
}

// NewFromMap : Makes a new instance of this transport module from a map of arguments (for deserialization support)
func NewFromMap(node api.Node, t map[string]interface{}) api.Transport {
	var certPem, keyPem string
	eccMode := true

	if _, ok := t["Cert"]; ok {
		certPem = t["Cert"].(string)
	}
	if _, ok := t["Key"]; ok {
		keyPem = t["Key"].(string)
	}
	if _, ok := t["EccMode"]; ok {
		eccMode = t["EccMode"].(bool)
	}
	return New([]byte(certPem), []byte(keyPem), node, eccMode)
}

// New : Makes a new instance of this transport module
func New(certPem []byte, keyPem []byte, node api.Node, eccMode bool) *Module {

	web := new(Module)

	web.Cert = certPem
	web.Key = keyPem
	web.node = node
	web.EccMode = eccMode

	web.transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	web.client = &http.Client{
		Timeout:   time.Second * 10,
		Transport: web.transport}

	web.byteLimit = 125000 // 150000 was unstable, 125000 was 100% stable

	return web
}

// Module : HTTPS Implementation of a Transport module
type Module struct {
	transport *http.Transport
	client    *http.Client
	node      api.Node
	isRunning bool
	wg        sync.WaitGroup
	listeners []net.Listener

	Cert, Key []byte
	EccMode   bool

	byteLimit int64
}

// Name : Returns this module's common name, which should be unique
func (*Module) Name() string {
	return "https"
}

// MarshalJSON : Create a serialied representation of the config of this module
func (h *Module) MarshalJSON() (b []byte, e error) {
	return json.Marshal(map[string]interface{}{
		"Transport": "https",
		"Cert":      string(h.Cert),
		"Key":       string(h.Key),
		"EccMode":   h.EccMode})
}

// ByteLimit - get limit on bytes per bundle for this transport
func (h *Module) ByteLimit() int64 { return h.byteLimit }

// SetByteLimit - set limit on bytes per bundle for this transport
func (h *Module) SetByteLimit(limit int64) { h.byteLimit = limit }

// Listen : Server interface
func (h *Module) Listen(listen string, adminMode bool) {
	// make sure we are not already running
	if h.isRunning {
		events.Warning(h.node, "This listener is already running.")
		return
	}

	// init ssl components
	cert, err := tls.X509KeyPair(h.Cert, h.Key)
	if err != nil {
		events.Error(h.node, err.Error())
		return
	}

	// build http handler
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		h.handleResponse(w, r, h.node, adminMode)
	})

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

	// start
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		if err := http.Serve(tlsListener, serveMux); err != nil {
			events.Error(h.node, err.Error())
		}
	}()
	h.isRunning = true
}

func (h *Module) handleResponse(w http.ResponseWriter, r *http.Request, node api.Node, adminMode bool) {

	var a api.RemoteCall

	dec := gob.NewDecoder(r.Body)
	if err := dec.Decode(&a); err != nil {
		events.Warning(h.node, "https handleResponse gob decode failed: "+err.Error())
		return
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

	enc := gob.NewEncoder(w)
	if err := enc.Encode(rr); err != nil {
		events.Warning(h.node, "https listen gob encode failed: "+err.Error())
	}
}

// RPC : client interface
func (h *Module) RPC(host string, method string, args ...interface{}) (interface{}, error) {
	events.Info(h.node, fmt.Sprintf("\n***\n***RPC %s on %s called with: %+v\n***\n", method, host, args))

	var a api.RemoteCall
	a.Action = method
	a.Args = args

	var buf bytes.Buffer
	//use default gob encoder
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(a); err != nil {
		events.Error(h.node, "https rpc gob encode failed: "+err.Error())
		return nil, err
	}

	req, _ := http.NewRequest("POST", "https://"+host, &buf)

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rr api.RemoteResponse
	dec := gob.NewDecoder(resp.Body)
	if err := dec.Decode(&rr); err != nil {
		events.Warning(h.node, "https rpc gob decode failed: "+err.Error())
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

// Stop : stops the HTTPS transport from running
func (h *Module) Stop() {
	h.isRunning = false
	for _, listener := range h.listeners {
		listener.Close()
	}
	h.wg.Wait()
}
