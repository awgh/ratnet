package policy

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/ratnet"
	"github.com/awgh/ratnet/api"

	"log"
	"net"

	"math/rand"

	dns "github.com/miekg/dns"
)

// P2P - a policy which makes peer-to-peer connections without a priori knowledge of peers
//
type P2P struct {
	Transport api.Transport
	ListenURI string
	AdminMode bool
	Node      api.Node

	IsListening   bool
	IsAdvertising bool

	listenSocket *net.UDPConn
	dialSocket   *net.UDPConn

	negotiationRank uint64

	// last poll times
	lastPollLocal, lastPollRemote int64
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
	return NewP2P(transport, listenURI, node, adminMode)
}

// NewP2P : Returns a new instance of a P2P Connection Policy
//
func NewP2P(transport api.Transport, listenURI string, node api.Node, adminMode bool) *P2P {
	s := new(P2P)
	s.Transport = transport
	s.ListenURI = listenURI
	s.AdminMode = adminMode
	s.Node = node

	s.rerollNegotiationRank()
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

func (s *P2P) rerollNegotiationRank() {
	s.negotiationRank = uint64(rand.Uint32())<<32 + uint64(rand.Uint32())
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
		b := make([]byte, maxDatagramSize)
		conn := s.listenSocket
		if _, _, err := conn.ReadFromUDP(b); err != nil {
			return err
		}
		msg := &dns.Msg{}
		msg.Unpack(b[:])

		target := ""
		var targetNegRank uint64
		prefixLen := 3 // .rn or .ng

		for _, q := range msg.Question {
			if len(q.Name) > prefixLen {
				if q.Name[:prefixLen] == "rn." {
					qn := strings.Split(q.String(), ".")
					hexed := qn[1]
					dehexed, err := hex.DecodeString(hexed)
					if err != nil {
						return err
					}
					target = string(dehexed)
				} else if q.Name[:prefixLen] == "ng." {
					qm := strings.Split(q.String(), ".")
					hexed := qm[1]
					dehexed, err := hex.DecodeString(hexed)
					if err != nil {
						return err
					}
					targetNegRank = binary.LittleEndian.Uint64(dehexed)
				}
			}
		}
		if target != "" && targetNegRank > 0 {
			/*
				Negotiation:
					- The lowest rank does a push/pull
					- In case of collision, they both do a push/pull
						(not ideal, but loop detection should eat it and we made it a uint64...
						 don't want to reroll because that way the push/pull relationships can be more long-lived)
			*/
			if s.negotiationRank <= targetNegRank {
				pubsrv, err := s.Node.ID()
				if err != nil {
					log.Fatal("Couldn't get routing key in P2P.RunPolicy:\n" + err.Error())
				}
				log.Println("Won Negotiation, Push/Pulling target/me ", target, s.ListenURI)
				_, err = s.pollServer(s.Transport, s.Node, target, pubsrv) //actually, we might want a different transport here based on the scheme
				return err
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
	a := make([]byte, 8)
	binary.LittleEndian.PutUint64(a, s.negotiationRank)
	encodedStr := hex.EncodeToString([]byte(clear))
	oname := "rn." + encodedStr + ".local."
	oneg := "ng." + hex.EncodeToString(a) + ".local."

	// send the query
	m := new(dns.Msg)
	m.Id = dns.Id()
	m.RecursionDesired = false
	m.Response = true
	m.Opcode = dns.OpcodeQuery
	m.Rcode = dns.RcodeSuccess
	m.Question = make([]dns.Question, 2)
	m.Question[0] = dns.Question{Name: oname, Qtype: dns.TypeSRV, Qclass: dns.ClassINET}
	m.Question[1] = dns.Question{Name: oneg, Qtype: dns.TypeSRV, Qclass: dns.ClassINET}
	msgBytes, err := m.Pack()
	if err != nil {
		log.Println("Pack failed with:", err.Error(), oname)
		return err
	}

	conn := s.dialSocket
	n, err := conn.Write(msgBytes)
	if err != nil || n != len(msgBytes) {
		log.Println("Write failed with:", err.Error())
	}
	return nil
}

// pollServer will keep trying until either we get a result or the timeout expires
// todo: this is exactly the same as the one in Poll... merge somehow?
func (p *P2P) pollServer(transport api.Transport, node api.Node, host string, pubsrv bc.PubKey) (bool, error) {

	// Pickup Local
	rpubkey, err := transport.RPC(host, "ID")
	if err != nil {
		return false, err
	}
	rpk := pubsrv.Clone()
	if err := rpk.FromB64(string(rpubkey)); err != nil {
		return false, err
	}

	toRemoteRaw, err := node.Pickup(rpk, p.lastPollLocal)
	if err != nil {
		return false, err
	}

	// Pickup Remote
	toLocalRaw, err := transport.RPC(host, "Pickup", pubsrv.ToB64(), strconv.FormatInt(p.lastPollRemote, 10))
	if err != nil {
		return false, err
	}
	var toLocal api.Bundle
	if err := json.Unmarshal(toLocalRaw, &toLocal); err != nil {
		return false, err
	}

	p.lastPollLocal = toRemoteRaw.Time
	p.lastPollRemote = toLocal.Time

	toRemote, err := json.Marshal(toRemoteRaw)
	if err != nil {
		return false, err
	}

	// Dropoff Remote
	if len(toRemoteRaw.Data) > 0 {
		if _, err := transport.RPC(host, "Dropoff", string(toRemote)); err != nil {
			return false, err
		}
	}
	// Dropoff Local
	if len(toLocal.Data) > 0 {
		if err := node.Dropoff(toLocal); err != nil {
			return false, err
		}
	}
	return true, nil
}
