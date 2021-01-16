package udp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	kcp "github.com/xtaci/kcp-go/v5"

	"github.com/awgh/ratnet/api"
	"github.com/awgh/ratnet/api/events"
)

var cachedSessions map[string]*kcp.UDPSession

func init() {
	cachedSessions = make(map[string]*kcp.UDPSession)
}

// New : Makes a new instance of this transport module
func New(node api.Node) *Module {
	instance := new(Module)
	instance.node = node

	instance.byteLimit = 8000 * 1024 // 125000

	return instance
}

// Module : UDP Implementation of a Transport module
type Module struct {
	node      api.Node
	isRunning uint32
	wg        sync.WaitGroup
	byteLimit int64
}

// Name : Returns name of module
func (m *Module) Name() string {
	return "udp"
}

// ByteLimit - get limit on bytes per bundle for this transport
func (m *Module) ByteLimit() int64 { return m.byteLimit }

// SetByteLimit - set limit on bytes per bundle for this transport
func (m *Module) SetByteLimit(limit int64) { m.byteLimit = limit }

// Listen : opens a UDP socket and listens
func (m *Module) Listen(listen string, adminMode bool) {
	// make sure we dont run twice
	if m.IsRunning() {
		return
	}
	lis, err := kcp.ListenWithOptions(listen, nil, 10, 0) // disabled FEC
	if err != nil {
		events.Error(m.node, err.Error())
		return
	}
	m.setIsRunning(true)
	m.wg.Add(1)

	// read loop
	go func() {
		defer lis.Close() // make sure the socket closes when we're done with it
		defer m.wg.Done()

		// read from socket
		for m.IsRunning() {
			lis.SetReadDeadline(time.Now().Add(1 * time.Second))
			lis.SetWriteDeadline(time.Now().Add(1 * time.Second))
			c, err := lis.Accept()
			if err != nil {
				events.Error(m.node, err.Error())
				continue
			}

			events.Debug(m.node, "UDP accepted new connection")

			c.SetReadDeadline(time.Now().Add(35 * time.Second))
			c.SetWriteDeadline(time.Now().Add(35 * time.Second))

			go func(conn net.Conn) {
				reader := bufio.NewReader(conn)
				writer := bufio.NewWriter(conn)

				for m.IsRunning() { // read multiple messages on the same connection

					buf, err := api.ReadBuffer(reader)
					if err != nil {
						events.Warning(m.node, err.Error())
						break
					}
					a, err := api.RemoteCallFromBytes(buf)
					if err != nil {
						events.Warning(m.node, "udp listen remote deserialize failed: "+err.Error())
						break
					}

					var result interface{}
					if adminMode {
						result, err = m.node.AdminRPC(m, *a)
					} else {
						result, err = m.node.PublicRPC(m, *a)
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
						events.Warning(m.node, "udp listen remote write failed: "+err.Error())
						break
					}
					writer.Flush()
				}
			}(c)
		}
	}()
}

// RPC : transmit data via UDP
func (m *Module) RPC(host string, method api.Action, args ...interface{}) (interface{}, error) {
	events.Info(m.node, fmt.Sprintf("\n***\n***RPC %d on %s called with: %+v\n***\n", method, host, args))

	conn, ok := cachedSessions[host]
	if !ok {
		// open client socket
		var err error
		conn, err = kcp.DialWithOptions(host, nil, 10, 0) // disabled FEC
		if err != nil {
			events.Warning(m.node, "kcp dial error in udp:", err)
			return nil, err
		}
		conn.SetStreamMode(false)
		conn.SetWindowSize(512, 512)
		conn.SetNoDelay(1, 20, 2, 1)
		conn.SetACKNoDelay(true)

		cachedSessions[host] = conn
	}
	conn.SetReadDeadline(time.Now().Add(35 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(35 * time.Second))

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	var a api.RemoteCall
	a.Action = method
	a.Args = args

	rbytes := api.RemoteCallToBytes(&a)
	err := api.WriteBuffer(writer, rbytes)
	if err != nil {
		events.Warning(m.node, "udp RPC remote write failed: "+err.Error())
		delete(cachedSessions, host) // something's wrong, make a new session next attempt
		_ = conn.Close()
		return nil, err
	}
	writer.Flush()

	buf, err := api.ReadBuffer(reader)
	if err != nil {
		events.Warning(m.node, "udp RPC remote read failed: "+err.Error())
		delete(cachedSessions, host) // something's wrong, make a new session next attempt
		_ = conn.Close()
		return nil, err
	}
	rr, err := api.RemoteResponseFromBytes(buf)
	if err != nil {
		delete(cachedSessions, host) // something's wrong, make a new session next attempt
		_ = conn.Close()
		if err == io.EOF {
			return nil, nil
		}
		events.Warning(m.node, "udp RPC decode failed: "+err.Error())
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

// Stop : Stops module
func (m *Module) Stop() {
	m.setIsRunning(false)

	for k, v := range cachedSessions {
		delete(cachedSessions, k)
		_ = v.Close()
	}
	m.wg.Wait()
}

// IsRunning - returns true if this node is running
func (m *Module) IsRunning() bool {
	return atomic.LoadUint32(&m.isRunning) == 1
}

func (m *Module) setIsRunning(b bool) {
	var running uint32 = 0
	if b {
		running = 1
	}
	atomic.StoreUint32(&m.isRunning, running)
}
