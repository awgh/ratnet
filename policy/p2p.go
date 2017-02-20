package policy

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/awgh/ratnet"
	"github.com/awgh/ratnet/api"

	"log"
	"net"

	dns "github.com/miekg/dns"
)

// P2P - a policy which makes peer-to-peer connections without a priori knowledge of peers
//
type P2P struct {
	Transport api.Transport
	ListenURI string
	AdminMode bool

	IsListening   bool
	IsAdvertising bool

	listenSocket *net.UDPConn
	dialSocket   *net.UDPConn
}

var (
	maxDatagramSize = 4096

	multicastAddr = &net.UDPAddr{
		IP:   net.IPv4(224, 0, 0, 251),
		Port: 5353,
	}
)

func init() {
	ratnet.Policies["p2p"] = NewP2PFromMap // register this module by name (for deserialization support)
}

// NewP2PFromMap : Makes a new instance of this transport module from a map of arguments (for deserialization support)
func NewP2PFromMap(transport api.Transport, node api.Node, p map[string]interface{}) api.Policy {
	listenURI := p["ListenURI"].(string)
	adminMode := p["AdminMode"].(bool)
	return NewP2P(transport, listenURI, adminMode)
}

// NewP2P : Returns a new instance of a P2P Connection Policy
//
func NewP2P(transport api.Transport, listenURI string, adminMode bool) *P2P {
	s := new(P2P)
	s.Transport = transport
	s.ListenURI = listenURI
	s.AdminMode = adminMode
	return s
}

// MarshalJSON : Create a serialied representation of the config of this policy
func (s *P2P) MarshalJSON() (b []byte, e error) {
	return json.Marshal(map[string]interface{}{
		"Policy":    "p2p",
		"ListenURI": s.ListenURI,
		"AdminMode": s.AdminMode,
		"Transport": s.Transport})
}

func (s *P2P) initListenSocket() {
	socket, err := net.ListenMulticastUDP("udp4", nil, multicastAddr)
	if err != nil {
		log.Fatal(err.Error())
	}
	socket.SetReadBuffer(maxDatagramSize)
	s.listenSocket = socket
}

func (s *P2P) initDialSocket() {
	socket, err := net.DialUDP("udp", nil, multicastAddr)
	if err != nil {
		log.Fatal(err.Error())
	}
	s.dialSocket = socket
}

// RunPolicy : Executes the policy as a goroutine
//
func (s *P2P) RunPolicy() error {

	s.initListenSocket()
	s.initDialSocket()

	s.Transport.Listen(s.ListenURI, s.AdminMode)
	s.IsListening = true

	go s.mdnsListen()
	go func() {
		for s.IsListening {
			if err := s.mdnsAdvertise(); err != nil {
				log.Println("mdnsAdvertise errored: ", err.Error())
			}
			time.Sleep(time.Second) //time.Minute)
		}
	}()
	return nil
}

// Stop : Stops a policy
//
func (s *P2P) Stop() {
	s.Transport.Stop()
	s.IsListening = false

	s.listenSocket.Close()
	s.dialSocket.Close()
}

func (s *P2P) mdnsListen() error {
	for s.IsListening {
		log.Println("listen loop")
		b := make([]byte, maxDatagramSize)
		conn := s.listenSocket
		if _, _, err := conn.ReadFromUDP(b); err != nil {
			return err
		}
		msg := &dns.Msg{}
		msg.Unpack(b[:])
		for _, q := range msg.Question {
			if q.Name[:len("rn.")] == "rn." {
				qn := strings.Split(q.String(), ".")
				hexed := qn[1]
				dehexed, err := hex.DecodeString(hexed)
				if err != nil {
					return err
				}
				log.Println(string(dehexed))
			}
		}
	}
	return nil
}

func (s *P2P) mdnsAdvertise() error {

	log.Println("mdns Advertising...")

	// prepare the service string
	hp := strings.Split(s.ListenURI, ":") // listen URI has no protocol, is in format [HOST]:PORT
	if len(hp) < 1 {
		return errors.New("Split Host/Port failed with no port.")
	}
	port := hp[len(hp)-1]

	ip := strings.Split(s.dialSocket.LocalAddr().String(), ":") // LocalAddr has no protocol, is in format [HOST]:PORT
	if len(ip) < 1 {
		return errors.New("Split Host/Port failed.")
	}
	laddr := ip[0]
	clear := s.Transport.Name() + "://" + laddr + ":" + port
	encodedStr := hex.EncodeToString([]byte(clear))
	oname := "rn." + encodedStr + ".local."

	// send the query
	m := new(dns.Msg)
	m.Id = dns.Id()
	m.RecursionDesired = false
	m.Response = true
	m.Opcode = dns.OpcodeQuery
	m.Rcode = dns.RcodeSuccess
	m.Question = make([]dns.Question, 1)
	m.Question[0] = dns.Question{Name: oname, Qtype: dns.TypeSRV, Qclass: dns.ClassINET}
	msgBytes, err := m.Pack()
	if err != nil {
		log.Println("Pack failed with:", err.Error())
		return err
	}

	conn := s.dialSocket
	n, err := conn.Write(msgBytes)
	if err != nil || n != len(msgBytes) {
		log.Println("Write failed with:", err.Error())
	}
	return nil
}
