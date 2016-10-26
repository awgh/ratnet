package udp

import (
	"bufio"
	"encoding/json"
	"log"
	"net"
	"sync"

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

	// parse UDP address
	udpAddress, err := net.ResolveUDPAddr("udp", listen)
	if err != nil {
		log.Println(err.Error())
		return
	}

	// open socket
	socket, err := net.ListenUDP("udp", udpAddress)
	if err != nil {
		log.Println(err.Error())
		return
	}

	m.isRunning = true

	// read loop
	go func() {
		m.wg.Add(1)
		defer socket.Close() // make sure the socket closes when we're done with it
		defer m.wg.Done()

		// read from socket
		for m.isRunning {
			b := make([]byte, 8192)
			n, remoteAddr, err := socket.ReadFrom(b)
			if err != nil {
				log.Println(err)
				continue
			}
			var a api.RemoteCall
			if err := json.Unmarshal(b[:n], &a); err != nil {
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
			socket.WriteTo(append([]byte(result), DELIM), remoteAddr)
		}
	}()
}

// RPC : transmit data via UDP
func (m *Module) RPC(host string, method string, args ...string) ([]byte, error) {

	// parse UDP addresses
	udpRemoteAddress, err := net.ResolveUDPAddr("udp", host)
	if err != nil {
		return nil, err
	}
	udpClientAddress, err := net.ResolveUDPAddr("udp", "0.0.0.0:0")
	if err != nil {
		return nil, err
	}

	// open client socket
	conn, err := net.DialUDP("udp", udpClientAddress, udpRemoteAddress)
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
	writer.Write(b)
	writer.WriteByte(DELIM)
	writer.Flush()

	// get response
	resp, err := reader.ReadBytes(DELIM)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return resp, nil
}

// Stop : Stops module
func (m *Module) Stop() {
	m.isRunning = false
	m.wg.Wait()
}
