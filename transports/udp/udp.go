package udp

import (
	"bufio"
	"encoding/json"
	"log"
	"sync"

	kcp "github.com/xtaci/kcp-go"

	"github.com/awgh/ratnet"
	"github.com/awgh/ratnet/api"
)

// DELIM : Message delimiter character
var DELIM byte

func init() {
	DELIM = 0x0a
	ratnet.Transports["udp"] = NewFromMap // register this module by name (for deserialization support)
}

// NewFromMap : Makes a new instance of this transport module from a map of arguments (for deserialization support)
func NewFromMap(node api.Node, t map[string]interface{}) api.Transport {
	return New(node)
}

// New : Makes a new instance of this transport module
func New(node api.Node) *Module {

	instance := new(Module)
	instance.node = node

	return instance
}

// Module : UDP Implementation of a Transport module
type Module struct {
	node      api.Node
	isRunning bool
	wg        sync.WaitGroup
}

// Name : Returns name of module
func (m *Module) Name() string {
	return "udp"
}

// MarshalJSON : Create a serialied representation of the config of this module
func (m *Module) MarshalJSON() (b []byte, e error) {
	return json.Marshal(map[string]interface{}{
		"Transport": "udp"})
}

// Listen : opens a UDP socket and listens
func (m *Module) Listen(listen string, adminMode bool) {
	// make sure we dont run twice
	if m.isRunning {
		return
	}
	lis, err := kcp.ListenWithOptions(listen, nil, 10, 3)
	if err != nil {
		log.Println(err.Error())
		return
	}
	m.isRunning = true

	// read loop
	go func() {
		m.wg.Add(1)
		defer lis.Close() // make sure the socket closes when we're done with it
		defer m.wg.Done()

		// read from socket
		for m.isRunning {
			conn, err := lis.Accept()
			if err != nil {
				log.Println(err)
				continue
			}

			reader := bufio.NewReader(conn)
			writer := bufio.NewWriter(conn)
			b, err := reader.ReadBytes(DELIM)
			if err != nil {
				log.Println(err)
				continue
			}

			var a api.RemoteCall
			if err := json.Unmarshal(b, &a); err != nil {
				log.Println(err.Error())
				continue
			}

			var result string
			if adminMode {
				result, err = m.node.AdminRPC(a.Action, a.Args...)
			} else {
				result, err = m.node.PublicRPC(a.Action, a.Args...)
			}
			if err != nil {
				log.Println(err.Error())
				result = err.Error()
			} else if len(result) < 1 {
				result = "OK" // todo: for backwards compatability, remove when nothing needs it
			}
			writer.Write(append([]byte(result), DELIM))
			writer.Flush()
		}
	}()
}

// RPC : transmit data via UDP
func (m *Module) RPC(host string, method string, args ...string) ([]byte, error) {

	// open client socket
	conn, err := kcp.DialWithOptions(host, nil, 10, 3)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	var a api.RemoteCall
	a.Action = method
	a.Args = args
	b, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	// send data
	_, err = writer.Write(b)
	if err != nil {
		return nil, err
	}
	writer.WriteByte(DELIM)
	writer.Flush()

	resp, err := reader.ReadBytes(DELIM)
	return resp, err
}

// Stop : Stops module
func (m *Module) Stop() {
	m.isRunning = false
	m.wg.Wait()
}
