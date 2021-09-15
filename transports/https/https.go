package https

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/awgh/ratnet/api"
	"github.com/awgh/ratnet/api/events"
)

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
		Transport: web.transport,
	}

	web.byteLimit = 125000 // 150000 was unstable, 125000 was 100% stable

	return web
}

// Module : HTTPS Implementation of a Transport module
type Module struct {
	transport *http.Transport
	client    *http.Client
	server    *http.Server
	node      api.Node
	isRunning uint32
	mutex     sync.Mutex

	Cert, Key []byte
	EccMode   bool

	byteLimit int64
}

// Name : Returns this module's common name, which should be unique
func (*Module) Name() string {
	return "https"
}

// ByteLimit - get limit on bytes per bundle for this transport
func (h *Module) ByteLimit() int64 { return atomic.LoadInt64(&h.byteLimit) }

// SetByteLimit - set limit on bytes per bundle for this transport
func (h *Module) SetByteLimit(limit int64) { atomic.StoreInt64(&h.byteLimit, limit) }

// OverrideServer - override the http.Server object with one supplied by the caller
func (h *Module) OverrideServer(server *http.Server) {
	h.server = server
}

// Listen : Server interface
func (h *Module) Listen(listen string, adminMode bool) {
	if len(listen) > 0 {

		// make sure we are not already running
		if h.IsRunning() {
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
			h.HandleResponse(w, r, h.node, adminMode)
		})

		h.mutex.Lock()
		h.server = &http.Server{
			Addr:      listen,
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{cert}},
			Handler:   serveMux,
		}
		h.mutex.Unlock()

		// start
		go func() {
			if err := h.server.ListenAndServeTLS("", ""); err != nil {
				events.Error(h.node, err.Error())
			}
		}()

	}

	h.setIsRunning(true)
}

func (h *Module) HandleResponse(w http.ResponseWriter, r *http.Request, node api.Node, adminMode bool) {
	buf, err := api.ReadBuffer(r.Body)
	if err != nil {
		events.Warning(h.node, err.Error())
		return
	}
	a, err := api.RemoteCallFromBytes(buf)
	if err != nil {
		events.Warning(h.node, "https listen remote deserialize failed: "+err.Error())
		return
	}

	var result interface{}
	if adminMode {
		result, err = node.AdminRPC(h, *a)
	} else {
		result, err = node.PublicRPC(h, *a)
	}

	rr := api.RemoteResponse{}
	if err != nil {
		rr.Error = err.Error()
	}
	if result != nil {
		rr.Value = result
	}

	rbytes := api.RemoteResponseToBytes(&rr)
	err = api.WriteBuffer(w, rbytes)
	if err != nil {
		events.Warning(h.node, "https listen remote write failed: "+err.Error())
		return
	}
}

// RPC : client interface
func (h *Module) RPC(host string, method api.Action, args ...interface{}) (interface{}, error) {
	events.Info(h.node, fmt.Sprintf("\n***\n***RPC %d on %s called with: %+v\n***\n", method, host, args))

	var a api.RemoteCall
	a.Action = method
	a.Args = args

	rbytes := api.RemoteCallToBytes(&a)
	var bbuf bytes.Buffer
	writer := bufio.NewWriter(&bbuf)
	err := api.WriteBuffer(writer, rbytes)
	if err != nil {
		events.Warning(h.node, "https RPC buffer write failed: "+err.Error())
		return nil, err
	}
	writer.Flush()

	req, _ := http.NewRequest("POST", "https://"+host, &bbuf)
	resp, err := h.client.Do(req)
	if err != nil {
		events.Warning(h.node, "https RPC remote write failed: "+err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	buf, err := api.ReadBuffer(resp.Body)
	if err != nil {
		events.Warning(h.node, "https RPC remote read failed: "+err.Error())
		return nil, err
	}
	rr, err := api.RemoteResponseFromBytes(buf)
	if err != nil {
		events.Warning(h.node, "https RPC decode failed: "+err.Error())
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
	h.mutex.Lock()
	h.server.Close()
	h.mutex.Unlock()
	h.setIsRunning(false)
}

// IsRunning - returns true if this node is running
func (h *Module) IsRunning() bool {
	return atomic.LoadUint32(&h.isRunning) == 1
}

func (h *Module) setIsRunning(b bool) {
	var running uint32 = 0
	if b {
		running = 1
	}
	atomic.StoreUint32(&h.isRunning, running)
}
