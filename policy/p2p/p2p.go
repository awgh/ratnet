package p2p

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math/rand"
	"net"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/awgh/ratnet/api"
	"github.com/awgh/ratnet/api/events"
	"github.com/awgh/ratnet/policy"

	dns "github.com/miekg/dns"
)

// P2P - a policy which makes peer-to-peer connections without a priori knowledge of peers
//
type P2P struct {
	negotiationRank uint64
	// last poll times - moving to peerTable
	// lastPollLocal, lastPollRemote int64

	ListenInterval    int
	AdvertiseInterval int

	isListening   uint32
	isAdvertising uint32

	AdminMode    bool
	ListenURI    string
	localAddress string
	Transport    api.Transport
	Node         api.Node
	pt           *policy.PeerTable

	listenSocket *net.UDPConn
	dialSocket   *net.UDPConn

	wg sync.WaitGroup
}

var (
	maxDatagramSize = 4096

	multicastAddr = &net.UDPAddr{
		IP:   net.IPv4(224, 0, 0, 251),
		Port: 5353,
	}
)

// New : Returns a new instance of a P2P Connection Policy
//
func New(transport api.Transport, listenURI string, node api.Node, adminMode bool,
	listenInterval int, advertiseInterval int) *P2P {
	s := new(P2P)
	s.Transport = transport
	s.ListenURI = listenURI
	s.AdminMode = adminMode
	s.Node = node
	s.ListenInterval = listenInterval
	s.AdvertiseInterval = advertiseInterval
	s.pt = policy.NewPeerTable()

	s.rerollNegotiationRank()
	return s
}

func (s *P2P) initListenSocket() {
	socket, err := net.ListenMulticastUDP("udp4", nil, multicastAddr)
	if err != nil {
		events.Critical(s.Node, err.Error())
	}
	err = socket.SetReadBuffer(maxDatagramSize)
	if err != nil {
		events.Critical(s.Node, err.Error())
	}
	s.listenSocket = socket
}

func (s *P2P) initDialSocket() error {
	socket, err := net.DialUDP("udp", nil, multicastAddr)
	if err != nil {
		events.Critical(s.Node, err.Error())
	}
	s.dialSocket = socket

	//
	// prepare the service string
	hp := strings.Split(s.ListenURI, ":") // listen URI has no protocol, is in format [HOST]:PORT
	if len(hp) < 1 {
		return errors.New("Split Host/Port failed with no port")
	}
	port := hp[len(hp)-1]
	ip := strings.Split(s.dialSocket.LocalAddr().String(), ":") // LocalAddr has no protocol, is in format [HOST]:PORT
	if len(ip) < 1 {
		return errors.New("Split Host/Port failed")
	}
	s.localAddress = s.Transport.Name() + "://" + ip[0] + ":" + port

	return nil
}

func (s *P2P) rerollNegotiationRank() {
	rand.Seed(time.Now().UnixNano()) // otherwise go defaults to seed(1). really.
	s.negotiationRank = uint64(rand.Uint32())<<32 + uint64(rand.Uint32())
}

// RunPolicy : Executes the policy as a goroutine
//
func (s *P2P) RunPolicy() error {
	s.initListenSocket()
	if err := s.initDialSocket(); err != nil {
		return err
	}

	s.Transport.Listen(s.ListenURI, s.AdminMode)
	s.setIsListening(true)
	s.setIsAdvertising(true)

	go s.mdnsListen()
	go func() {
		s.wg.Add(1)
		defer s.wg.Done()
		for s.IsListening() {
			if s.IsAdvertising() {
				if err := s.mdnsAdvertise(); err != nil {
					events.Warning(s.Node, "mdnsAdvertise errored: "+err.Error())
				}
			}
			time.Sleep(time.Duration(s.AdvertiseInterval) * time.Millisecond) // update interval
		}
	}()
	return nil
}

// Stop : Stops a policy
//
func (s *P2P) Stop() {
	s.Transport.Stop()
	s.setIsListening(false)
	s.setIsAdvertising(false)

	s.listenSocket.Close()
	s.dialSocket.Close()

	s.wg.Wait()
}

func (s *P2P) mdnsListen() error {
	peerlist := make(map[string]interface{})

	s.wg.Add(1)
	// defer conn.Close()
	defer s.wg.Done()
	for s.IsListening() {
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
		_, exists := peerlist[target]
		if !exists && (target != "" && targetNegRank > 0 && s.localAddress != target) {
			//s.setIsAdvertising(false)
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
					events.Critical(s.Node, "Couldn't get routing key in P2P.RunPolicy:\n"+err.Error())
				}
				events.Info(s.Node, "Won Negotiation, Push/Pulling target/me ", target, s.ListenURI)
				u, err := url.Parse(target)
				if err != nil {
					return err
				}

				// todo: no_json breaks this, have to use the transport from the constructor for the moment
				// t := make(map[string]interface{})
				// fromMapFn := ratnet.Transports[u.Scheme]
				// trans := fromMapFn(s.Node, t)

				trans := s.Transport
				peerlist[target] = trans
				go func() {
					for s.IsListening() {
						st := time.Now()
						if happy, err := s.pt.PollServer(trans, s.Node, target[len(u.Scheme)+3:], pubsrv); !happy {
							if err != nil {
								events.Warning(s.Node, err.Error())
							}
						}
						st2 := time.Now()
						events.Debug(s.Node, "p2p PollServer took: ", st2.Sub(st).String())
						runtime.GC()
						st3 := time.Now()
						events.Debug(s.Node, "p2p GC took: ", st3.Sub(st2).String())
						time.Sleep(time.Duration(s.ListenInterval) * time.Millisecond) // update interval
					}
				}()
			}
		}
	}
	return nil
}

func (s *P2P) mdnsAdvertise() error {
	events.Info(s.Node, "mdns Advertising...")
	a := make([]byte, 8)
	binary.LittleEndian.PutUint64(a, s.negotiationRank)
	encodedStr := hex.EncodeToString([]byte(s.localAddress))
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
		events.Warning(s.Node, "Pack failed with:", err.Error(), oname)
		return err
	}

	conn := s.dialSocket
	n, err := conn.Write(msgBytes)
	if err != nil || n != len(msgBytes) {
		events.Warning(s.Node, "Write failed with:", err.Error())
	}
	return nil
}

// GetTransport : Returns the transports associated with this policy
//
func (s *P2P) GetTransport() api.Transport {
	return s.Transport
}

// IsListening - returns true if this policy is listening
func (s *P2P) IsListening() bool {
	return atomic.LoadUint32(&s.isListening) == 1
}

func (s *P2P) setIsListening(b bool) {
	var listening uint32 = 0
	if b {
		listening = 1
	}
	atomic.StoreUint32(&s.isListening, listening)
}

// IsAdvertising - returns true if this policy is advertising
func (s *P2P) IsAdvertising() bool {
	return atomic.LoadUint32(&s.isAdvertising) == 1
}

func (s *P2P) setIsAdvertising(b bool) {
	var advertising uint32 = 0
	if b {
		advertising = 1
	}
	atomic.StoreUint32(&s.isAdvertising, advertising)
}
