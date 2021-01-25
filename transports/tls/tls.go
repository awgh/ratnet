package tls

import (
	"bufio"
	"crypto/tls"
	ctls "crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/awgh/ratnet/api"
	"github.com/awgh/ratnet/api/events"
)

// New : Makes a new instance of this transport module
func New(certPem, keyPem []byte, node api.Node, eccMode bool) *Module {
	tls := new(Module)

	tls.Cert = certPem
	tls.Key = keyPem
	tls.node = node
	tls.EccMode = eccMode

	tls.byteLimit = 8000 * 1024 // 125000 stable, 150000 was unstable

	tls.cachedSessions = make(map[string]*ctls.Conn)

	return tls
}

// Module : TLS Implementation of a Transport module
type Module struct {
	node           api.Node
	isRunning      uint32
	wg             sync.WaitGroup
	listeners      []net.Listener
	mutex          sync.Mutex
	cachedSessions map[string]*tls.Conn

	Cert, Key []byte
	EccMode   bool

	byteLimit int64
}

// Name : Returns this module's common name, which should be unique
func (*Module) Name() string {
	return "tls"
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

	h.mutex.Lock()

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

	h.mutex.Unlock()

	h.setIsRunning(true)

	h.wg.Add(1)
	go func() {
		defer tlsListener.Close()
		defer h.wg.Done()
		for h.IsRunning() {
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

	for h.IsRunning() { // read multiple messages on the same connection
		buf, err := api.ReadBuffer(reader)
		if err != nil {
			events.Warning(h.node, err.Error())
			break
		}
		a, err := api.RemoteCallFromBytes(buf)
		if err != nil {
			events.Warning(h.node, "tls listen remote deserialize failed: "+err.Error())
			break
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
		err = api.WriteBuffer(writer, rbytes)
		if err != nil {
			events.Warning(h.node, "tls listen remote write failed: "+err.Error())
			break
		}
		writer.Flush()
	}
}

// RPC : client interface
func (h *Module) RPC(host string, method api.Action, args ...interface{}) (interface{}, error) {
	events.Info(h.node, fmt.Sprintf("\n***\n***RPC %d on %s called with: %+v\n***\n", method, host, args))

	conn, ok := h.getCachedSession(host)
	if !ok {
		var err error
		conf := &tls.Config{InsecureSkipVerify: true}
		conn, err = tls.Dial("tcp", host, conf)
		if err != nil {
			events.Error(h.node, err.Error())
			return nil, err
		}
		h.setCachedSession(host, conn)
	}
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	var a api.RemoteCall
	a.Action = method
	a.Args = args

	rbytes := api.RemoteCallToBytes(&a)
	err := api.WriteBuffer(writer, rbytes)
	if err != nil {
		events.Warning(h.node, "tls RPC remote write failed: "+err.Error())
		h.deleteCachedSession(host) // something's wrong, make a new session next attempt
		return nil, err
	}
	writer.Flush()

	buf, err := api.ReadBuffer(reader)
	if err != nil {
		events.Warning(h.node, "tls RPC remote read failed: "+err.Error())
		h.deleteCachedSession(host) // something's wrong, make a new session next attempt
		return nil, err
	}
	rr, err := api.RemoteResponseFromBytes(buf)
	if err != nil {
		h.deleteCachedSession(host) // something's wrong, make a new session next attempt
		events.Warning(h.node, "tls RPC decode failed: "+err.Error())
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
	h.setIsRunning(false)

	h.mutex.Lock()
	for _, listener := range h.listeners {
		listener.Close()
	}
	h.mutex.Unlock()
	h.wg.Wait()

	h.clearCachedSessions()
}

func (h *Module) getCachedSession(host string) (*ctls.Conn, bool) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	v, ok := h.cachedSessions[host]
	return v, ok
}

func (h *Module) setCachedSession(host string, conn *ctls.Conn) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.cachedSessions[host] = conn
}

func (h *Module) deleteCachedSession(host string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	v, ok := h.cachedSessions[host]
	if ok {
		_ = v.Close()
	}
	delete(h.cachedSessions, host)
}

func (h *Module) clearCachedSessions() {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	for k, v := range h.cachedSessions {
		delete(h.cachedSessions, k)
		_ = v.Close()
	}
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
