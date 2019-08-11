package main

import (
	"bytes"
	"errors"
	"log"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/bencrypt/ecc"
	"github.com/awgh/ratnet/api"
	"github.com/awgh/ratnet/nodes/db"
	"github.com/awgh/ratnet/nodes/fs"
	"github.com/awgh/ratnet/nodes/qldb"
	"github.com/awgh/ratnet/nodes/ram"
	"github.com/awgh/ratnet/transports/https"
	"github.com/awgh/ratnet/transports/tls"
	"github.com/awgh/ratnet/transports/udp"

	//_ "modernc.org/ql/driver"
	_ "upper.io/db.v3/ql" // this requires PR #507: https://github.com/upper/db/pull/507
)

type TestNode struct {
	Node    api.Node
	Public  api.Transport
	Admin   api.Transport
	started bool
}

const (
	UDP int = iota
	TLS
	HTTPS
	NumTransports
)

const (
	RAM int = iota
	QL
	FS
	DB
	NumNodes
)

var nodeType int
var transportType int

func init() {
	nodeType = DB
	transportType = UDP
}

var (
	server1 TestNode
	server2 TestNode
	server3 TestNode

	p2p1 TestNode
	p2p2 TestNode

	server6 TestNode
	server7 TestNode
	server8 TestNode
)

func initNode(n int64, testNode TestNode, nodeType int, transportType int, p2pMode bool) TestNode {
	num := strconv.FormatInt(n, 10)

	if !testNode.started {
		testNode.started = true
		if nodeType == RAM {
			// RamNode Mode:
			testNode.Node = ram.New(new(ecc.KeyPair), new(ecc.KeyPair))
		} else if nodeType == QL {
			// QLDB Mode
			s := qldb.New(new(ecc.KeyPair), new(ecc.KeyPair))
			if err := os.RemoveAll("qltmp" + num); err != nil {
				log.Printf("error removing directory %s: %s\n", "qltmp"+num, err.Error())
			}
			os.Mkdir("qltmp"+num, os.FileMode(int(0755)))
			dbfile := "qltmp" + num + "/ratnet_test" + num + ".ql"
			s.BootstrapDB(dbfile)
			s.FlushOutbox(0)
			testNode.Node = s
		} else if nodeType == DB {
			// DB Mode
			s := db.New(new(ecc.KeyPair), new(ecc.KeyPair))
			if err := os.RemoveAll("dbtmp" + num); err != nil {
				log.Printf("error removing directory %s: %s\n", "dbtmp"+num, err.Error())
			}
			os.Mkdir("dbtmp"+num, os.FileMode(int(0755)))
			dbfile := "file://dbtmp" + num + "/ratnet_test" + num + ".ql"
			s.BootstrapDB("ql", dbfile)
			s.FlushOutbox(0)
			testNode.Node = s
		} else if nodeType == FS {
			testNode.Node = fs.New(new(ecc.KeyPair), new(ecc.KeyPair), "queue")
		}

		if transportType == UDP {
			testNode.Public = udp.New(testNode.Node)
			testNode.Admin = udp.New(testNode.Node)
		} else if transportType == TLS {
			testNode.Public = tls.New("tmp/cert"+num+".pem", "tmp/key"+num+".pem", testNode.Node, true)
			testNode.Admin = tls.New("tmp/cert"+num+".pem", "tmp/key"+num+".pem", testNode.Node, true)
		} else {
			testNode.Public = https.New("tmp/cert"+num+".pem", "tmp/key"+num+".pem", testNode.Node, true)
			testNode.Admin = https.New("tmp/cert"+num+".pem", "tmp/key"+num+".pem", testNode.Node, true)
		}
		if p2pMode {
			go p2p(testNode.Public, testNode.Admin, testNode.Node, "localhost:3000"+num, "localhost:30"+num+"0"+num)
		} else {
			go serve(testNode.Public, testNode.Admin, testNode.Node, "localhost:3000"+num, "localhost:30"+num+"0"+num)
		}

		time.Sleep(2 * time.Second)
	}
	return testNode
}

func Test_server_ID_1(t *testing.T) {
	server1 = initNode(1, server1, nodeType, transportType, false)

	var err error
	var r1, r2 interface{}
	if r1, err = server1.Public.RPC("localhost:30001", "ID"); err != nil {
		t.Error(err.Error())
	} else {
		t.Logf("%+v\n", r1)
	}
	// should work on both interfaces
	if r2, err = server1.Admin.RPC("localhost:30101", "ID"); err != nil {
		t.Error(err.Error())
	} else {
		t.Logf("%+v\n", r2)
	}
	r1k, ok := r1.(bc.PubKey)
	if !ok {
		t.Error("Public RPC did not return a PubKey")
	}
	r2k, ok := r2.(bc.PubKey)
	if !ok {
		t.Error("Admin RPC did not return a PubKey")
	}
	if !bytes.Equal(r1k.ToBytes(), r2k.ToBytes()) {
		t.Error(errors.New("Public and Admin interfaces returned different results"))
	}
}

func Test_server_CID_1(t *testing.T) {
	server1 = initNode(1, server1, nodeType, transportType, false)

	// should not work on public interface
	_, err := server1.Public.RPC("localhost:30001", "CID")
	if err == nil {
		t.Error(errors.New("CID was accessible on Public network interface"))
	}

	_, err = server1.Admin.RPC("localhost:30101", "CID")
	if err != nil {
		t.Error(err.Error())
	}
}

func Test_server_AddContact_1(t *testing.T) {
	server1 = initNode(1, server1, nodeType, transportType, false)
	server2 = initNode(2, server2, nodeType, transportType, false)

	p1, err := server2.Admin.RPC("localhost:30202", "CID")
	if err != nil {
		t.Error(err.Error())
	}
	t.Logf("Got CID: %+v\n", p1)
	r1 := p1.(bc.PubKey)
	t.Logf("CID cast to PubKey: %+v -> %s\n", r1, r1.ToB64())

	t.Log("Trying AddContact on Public interface")
	// should not work on public interface
	if _, erra := server1.Public.RPC("localhost:30001", "AddContact", "destname1", r1.ToB64()); erra == nil {
		t.Error(errors.New("AddContact was accessible on Public network interface"))
	}

	//t.Logf("r1: %T %v %v\n", r1, r1, ok)

	t.Log("Trying AddContact on Admin interface")
	_, errb := server1.Admin.RPC("localhost:30101", "AddContact", "destname1", r1.ToB64())
	if errb != nil {
		t.Error(errb.Error())
	}
	t.Log("Trying AddContact on local interface")
	if errc := server1.Node.AddContact("destname1", r1.ToB64()); errc != nil {
		t.Error(errc.Error())
	}
}

func Test_server_GetContact_1(t *testing.T) {
	server1 = initNode(1, server1, nodeType, transportType, false)

	t.Log("Trying GetContact on Public interface")
	// should not work on public interface
	if _, erra := server1.Public.RPC("localhost:30001", "GetContact", "destname1"); erra == nil {
		t.Error(errors.New("GetContact was accessible on Public network interface"))
	}

	t.Log("Trying GetContacts on Public interface")
	// should not work on public interface
	if _, erra := server1.Public.RPC("localhost:30001", "GetContacts"); erra == nil {
		t.Error(errors.New("GetContacts was accessible on Public network interface"))
	}

	t.Log("Trying GetContact on Admin interface")
	contact, err := server1.Admin.RPC("localhost:30101", "GetContact", "destname1")
	if err != nil {
		t.Error(err.Error())
	}
	t.Logf("Got Contact: %+v\n", contact)

	t.Log("Trying GetContacts on Admin interface")
	contactsRaw, err := server1.Admin.RPC("localhost:30101", "GetContacts")
	if err != nil {
		t.Error(err.Error())
	}
	contacts := contactsRaw.([]api.Contact)
	t.Logf("Got Contacts: %+v\n", contacts)
	if len(contacts) < 1 {
		t.Fail()
	}
}

func Test_server_AddChannel_1(t *testing.T) {
	server1 = initNode(1, server1, nodeType, transportType, false)

	// todo: add RSA test?
	chankey := pubprivkeyb64Ecc

	t.Log("Trying AddChannel on Public interface")
	// should not work on public interface
	_, err := server1.Public.RPC("localhost:30001", "AddChannel", "channel1", chankey)
	if err == nil {
		t.Error(errors.New("AddChannel was accessible on Public network interface"))
	}

	t.Log("Trying AddChannel on Admin interface")
	_, err = server1.Admin.RPC("localhost:30101", "AddChannel", "channel1", chankey)
	if err != nil {
		t.Error(err.Error())
	}
}

func Test_server_GetChannel_1(t *testing.T) {
	server1 = initNode(1, server1, nodeType, transportType, false)

	t.Log("Trying GetChannel on Public interface")
	// should not work on public interface
	if _, erra := server1.Public.RPC("localhost:30001", "GetChannel", "channel1"); erra == nil {
		t.Error(errors.New("GetChannel was accessible on Public network interface"))
	}

	t.Log("Trying GetChannels on Public interface")
	// should not work on public interface
	if _, erra := server1.Public.RPC("localhost:30001", "GetChannels"); erra == nil {
		t.Error(errors.New("GetChannels was accessible on Public network interface"))
	}

	t.Log("Trying GetChannel on Admin interface")
	channel, err := server1.Admin.RPC("localhost:30101", "GetChannel", "channel1")
	if err != nil {
		t.Error(err.Error())
	}
	t.Logf("Got Channel: %+v\n", channel)
	if channel == nil {
		t.Fail()
	}

	t.Log("Trying GetChannels on Admin interface")
	channelsRaw, err := server1.Admin.RPC("localhost:30101", "GetChannels")
	if err != nil {
		t.Error(err.Error())
	}
	channels := channelsRaw.([]api.Channel)
	t.Logf("Got Channels: %+v\n", channels)
	if len(channels) < 1 {
		t.Fail()
	}
}

func Test_server_AddProfile_1(t *testing.T) {
	server1 = initNode(1, server1, nodeType, transportType, false)

	t.Log("Trying AddProfile on Public interface")
	// should not work on public interface
	if _, erra := server1.Public.RPC("localhost:30001", "AddProfile", "profile1", "false"); erra == nil {
		t.Error(errors.New("AddProfile was accessible on Public network interface"))
	}

	t.Log("Trying AddProfile on Admin interface")
	_, errb := server1.Admin.RPC("localhost:30101", "AddProfile", "profile1", "false")
	if errb != nil {
		t.Error(errb.Error())
	}
}

func Test_server_GetProfile_1(t *testing.T) {
	server1 = initNode(1, server1, nodeType, transportType, false)

	/* todo: AddProfile twice in a row on ramnode is a bug
	t.Log("Trying AddProfile on Admin interface")
	_, errb := server1.Admin.RPC("localhost:30101", "AddProfile", "profile1", "false")
	if errb != nil {
		t.Error(errb.Error())
	}
	*/

	t.Log("Trying GetProfile on Public interface")
	// should not work on public interface
	if _, erra := server1.Public.RPC("localhost:30001", "GetProfile", "profile1"); erra == nil {
		t.Error(errors.New("GetProfile was accessible on Public network interface"))
	}

	t.Log("Trying GetProfiles on Public interface")
	// should not work on public interface
	if _, erra := server1.Public.RPC("localhost:30001", "GetProfiles"); erra == nil {
		t.Error(errors.New("GetProfiles was accessible on Public network interface"))
	}

	t.Log("Trying GetProfile on Admin interface")
	profile, err := server1.Admin.RPC("localhost:30101", "GetProfile", "profile1")
	if err != nil {
		t.Error(err.Error())
	}
	t.Logf("Got Profile: %+v\n", profile)
	if profile == nil {
		t.Fail()
	}

	t.Log("Trying GetProfiles on Admin interface")
	profilesRaw, err := server1.Admin.RPC("localhost:30101", "GetProfiles")
	if err != nil {
		t.Error(err.Error())
	}
	profiles := profilesRaw.([]api.Profile)
	t.Logf("Got Profiles: %+v\n", profiles)
	if len(profiles) < 1 {
		t.Fail()
	}

}

func Test_server_AddPeer_1(t *testing.T) {
	server1 = initNode(1, server1, nodeType, transportType, false)

	t.Log("Trying AddPeer on Public interface")
	// should not work on public interface
	if _, erra := server1.Public.RPC("localhost:30001", "AddPeer", "peer1", "false", "https://1.2.3.4:443"); erra == nil {
		t.Error(errors.New("AddPeer was accessible on Public network interface"))
	}

	t.Log("Trying AddPeer on Admin interface")
	_, errb := server1.Admin.RPC("localhost:30101", "AddPeer", "peer1", "false", "https://1.2.3.4:443")
	if errb != nil {
		t.Error(errb.Error())
	}

	t.Log("Trying AddPeer on Admin interface with a group name")
	_, errc := server1.Admin.RPC("localhost:30101", "AddPeer", "peer2", "false", "https://2.3.4.5:123", "groupnametest")
	if errc != nil {
		t.Error(errc.Error())
	}

	t.Log("Trying AddPeer on Admin interface with a group name that already exists")
	_, errd := server1.Admin.RPC("localhost:30101", "AddPeer", "peer3", "false", "https://3.4.5.6:234", "groupnametest")
	if errd != nil {
		t.Error(errd.Error())
	}
}

func Test_server_GetPeer_1(t *testing.T) {
	server1 = initNode(1, server1, nodeType, transportType, false)

	//
	t.Log("Trying AddPeer on Public interface")
	// should not work on public interface
	if _, erra := server1.Public.RPC("localhost:30001", "AddPeer", "peer1", "false", "https://1.2.3.4:443"); erra == nil {
		t.Error(errors.New("AddPeer was accessible on Public network interface"))
	}

	t.Log("Trying AddPeer on Admin interface")
	_, errb := server1.Admin.RPC("localhost:30101", "AddPeer", "peer1", "false", "https://1.2.3.4:443")
	if errb != nil {
		t.Error(errb.Error())
	}

	t.Log("Trying AddPeer on Admin interface with a group name")
	_, errc := server1.Admin.RPC("localhost:30101", "AddPeer", "peer2", "false", "https://2.3.4.5:123", "groupnametest")
	if errc != nil {
		t.Error(errc.Error())
	}

	t.Log("Trying AddPeer on Admin interface with a group name that already exists")
	_, errd := server1.Admin.RPC("localhost:30101", "AddPeer", "peer3", "false", "https://3.4.5.6:234", "groupnametest")
	if errd != nil {
		t.Error(errd.Error())
	}
	//

	t.Log("Trying GetPeer on Public interface")
	// should not work on public interface
	if _, erra := server1.Public.RPC("localhost:30001", "GetPeer", "peer1"); erra == nil {
		t.Fatal(errors.New("GetPeer was accessible on Public network interface"))
	}

	t.Log("Trying GetPeers on Public interface")
	// should not work on public interface
	if _, erra := server1.Public.RPC("localhost:30001", "GetPeers"); erra == nil {
		t.Fatal(errors.New("GetPeers was accessible on Public network interface"))
	}

	t.Log("Trying GetPeer on Admin interface")
	peer, err := server1.Admin.RPC("localhost:30101", "GetPeer", "peer1")
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Logf("Got Peer: %+v\n", peer)
	if peer == nil {
		t.Fatal(errors.New("GetPeer on Admin interface failed"))
	}

	t.Log("Trying GetPeers on Admin interface")
	peersRaw, err := server1.Admin.RPC("localhost:30101", "GetPeers")
	if err != nil {
		t.Fatal(err.Error())
	}
	peers := peersRaw.([]api.Peer)
	t.Logf("Got Peers: %+v\n", peers)
	if len(peers) != 1 {
		t.Fatal(errors.New("GetPeers on Admin interface failed"))
	}

	t.Log("Trying GetPeers on Admin interface with a group that has no peers")
	groupedPeers, err := server1.Admin.RPC("localhost:30101", "GetPeers", "not-a-group")
	if err != nil {
		t.Fatal(err.Error())
	}
	var groupPeers []api.Peer
	groupPeers = groupedPeers.([]api.Peer)
	t.Logf("Got Peers: %+v\n", groupPeers)
	if len(groupPeers) != 0 {
		t.Fatal(errors.New("GetPeers with a group with no peers returned results"))
	}

	t.Log("Trying GetPeers on Admin interface with a group that has peers")
	peers2, err := server1.Admin.RPC("localhost:30101", "GetPeers", "groupnametest")
	if err != nil {
		t.Fatal(err.Error())
	}
	var peergroup []api.Peer
	peergroup = peers2.([]api.Peer)
	t.Logf("Got Peers: %+v\n", peergroup)
	if len(peergroup) != 2 {
		t.Fatal(errors.New("GetPeers with a group with peers did not return two results"))
	}
}

func Test_server_Send_1(t *testing.T) {
	server1 = initNode(1, server1, nodeType, transportType, false)

	// should not work on public interface
	_, err := server1.Public.RPC("localhost:30001", "Send", "destname1", []byte(testMessage1))
	if err == nil {
		t.Error(errors.New("Send was accessible on Public network interface"))
	}

	_, err = server1.Admin.RPC("localhost:30101", "Send", "destname1", []byte(testMessage1))
	if err != nil {
		t.Error(err.Error())
	}
}

func Test_server_SendChannel_1(t *testing.T) {
	server1 = initNode(1, server1, nodeType, transportType, false)

	// should not work on public interface
	_, err := server1.Public.RPC("localhost:30001", "SendChannel", "channel1", []byte(testMessage2))
	if err == nil {
		t.Error(errors.New("SendChannel was accessible on Public network interface"))
	}

	_, err = server1.Admin.RPC("localhost:30101", "SendChannel", "channel1", []byte(testMessage2))
	if err != nil {
		t.Error(err.Error())
	}
}

var randmessage []byte

func Test_server_SendChannel_2(t *testing.T) { // now with chunking
	server1 = initNode(1, server1, nodeType, transportType, false)

	randmessage, err := bc.GenerateRandomBytes(8675)
	if err != nil {
		t.Error(err.Error())
	}
	//override byte limit to trigger chunking
	oldByteLimit := server1.Public.ByteLimit()
	server1.Public.SetByteLimit(4096)

	// should not work on public interface
	_, err = server1.Public.RPC("localhost:30001", "SendChannel", "channel1", []byte(randmessage))
	if err == nil {
		t.Error(errors.New("SendChannel was accessible on Public network interface"))
	}

	_, err = server1.Admin.RPC("localhost:30101", "SendChannel", "channel1", []byte(randmessage))
	if err != nil {
		t.Error(err.Error())
	}

	//restore byte limit
	server1.Public.SetByteLimit(oldByteLimit)
}

func Test_server_PickupDropoff_1(t *testing.T) {
	server1 = initNode(1, server1, nodeType, transportType, false)
	server2 = initNode(2, server2, nodeType, transportType, false)

	go func() {
		msg := <-server2.Node.Out()
		t.Log("server2.Out Got: ")
		t.Log(msg)
	}()

	chankey := pubprivkeyb64Ecc
	t.Log("Trying AddChannel on server2 Admin interface")
	_, err := server2.Admin.RPC("localhost:30202", "AddChannel", "channel1", chankey)
	if err != nil {
		t.Error(err.Error())
	}

	pubsrv, err := server1.Public.RPC("localhost:30002", "ID")
	if err != nil {
		t.Error("XXX:" + err.Error())
	}
	//p1 := pubsrv.(bc.PubKey)
	bundle, err := server1.Public.RPC("localhost:30001", "Pickup", pubsrv, int64(31536000)) // 31536000 seconds in a year
	if err != nil {
		t.Error("YYY:" + err.Error())
	}

	_, err = server1.Public.RPC("localhost:30002", "Dropoff", bundle)
	if err != nil {
		t.Error("ZZZ:" + err.Error())
	}
	//time.Sleep(3 * time.Second)
}

func Test_server_PickupDropoff_2(t *testing.T) {
	server1 = initNode(1, server1, nodeType, transportType, false)
	server3 = initNode(3, server3, nodeType, transportType, false)

	chankey := pubprivkeyb64Ecc

	go func() {
		msg := <-server3.Node.Out()
		t.Log("server3.Out Got: ")
		t.Log(msg)
	}()

	t.Log("Trying AddChannel on server3 Admin interface")
	_, err := server3.Admin.RPC("localhost:30303", "AddChannel", "channel1", chankey)
	if err != nil {
		t.Error(err.Error())
	}
	pubsrv, err := server1.Public.RPC("localhost:30003", "ID")
	if err != nil {
		t.Error(err.Error())
	}
	_, err = server1.Admin.RPC("localhost:30101", "SendChannel", "channel1", []byte(testMessage2))
	if err != nil {
		t.Error(err.Error())
	}
	result, err := server1.Public.RPC("localhost:30001", "Pickup", pubsrv, int64(31536000), "channel1") // seconds in a year
	if err != nil {
		t.Error(err.Error())
	}
	_, err = server1.Public.RPC("localhost:30003", "Dropoff", result)
	if err != nil {
		t.Error(err.Error())
	}
}

func Test_p2p_Basic_1(t *testing.T) {

	p2p1 = initNode(4, p2p1, nodeType, transportType, true)
	p2p2 = initNode(5, p2p2, nodeType, transportType, true)

	for p2p1.Node == nil || p2p2.Node == nil {
		time.Sleep(1 * time.Second)
	}

	go func() {
		msg := <-p2p2.Node.Out()
		t.Log("p2p2.Out Got: ")
		t.Log(msg)
	}()

	if err := p2p1.Node.AddChannel("test1", pubprivkeyb64Ecc); err != nil {
		t.Error(err.Error())
	}
	if err := p2p2.Node.AddChannel("test1", pubprivkeyb64Ecc); err != nil {
		t.Error(err.Error())
	}

	if err := p2p1.Node.SendChannel("test1", []byte(testMessage1)); err != nil {
		t.Error(err.Error())
	}
}

// Test Messages

var testMessage1 = `'In THAT direction,' the Cat said, waving its right paw round, 'lives a Hatter: and in THAT direction,' waving the other paw, 'lives a March Hare. Visit either you like: they're both mad.'
'But I don't want to go among mad people,' Alice remarked.
'Oh, you can't help that,' said the Cat: 'we're all mad here. I'm mad. You're mad.'
'How do you know I'm mad?' said Alice.
'You must be,' said the Cat, 'or you wouldn't have come here.'`

var testMessage2 = `The spiders have always been slandered
in the idiotic pages
of exasperating simplifiers
who take the fly's point of view,
who describe them as devouring,
carnal, unfaithful, lascivious.
For me, that reputation
discredits just those who concocted it:
the spider is an engineer,
a divine maker of watches,
for one fly more or less
let the imbeciles detest them.
I want to have a talk with the spider,
I want her to weave me a star.`

// RSA TEST KEYS
/*
var pubkeypem = `-----BEGIN PUBLIC KEY-----
MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAsHZQ6wRM/V5r6DGCr2io
Us10ORayIPZEm4RwaqJSf8KdnaWa8sPdQInrpf0l9b2dqOHWk45pbHqRReyhOC8I
OmmdfIwqbmxqznQxCXtlek+wwrA7KXdrUkr4eIHbdo1g6TFBGweRmGklGkyZnL5X
WkMYFgCbn7s191Eron/yxj0WuKD3wpgZoN7qymT1dRLEQ8bgIE4YD7P7QbpcF3+D
ibuEQnwGQq3YXByBkFB97sT3cR5j3fvfRpwU0yz0a7qEzt6nE5RWrkh4bCQ6O4x8
3Yv6jHkOjjMtSTVTl4O35nuAaXEpu4J9KA6UzWw3tx1zPsE6Gai7wKI1ZjD8Cbpy
SS7zgkJdxYh3FagCwH7e8zeCEnXmgHGMEiBOeah7S+z17ZTaHqseml20QGSD7Axz
LV9KXyt63fVz8Pa9zp09n3Q/2jJXFQo762CGJlenOWNz9KdUnoS18NYIGC2/h94D
7VnKCo5JERrG/WkvzkxoJs3LfDIL5VHERR9OqlVpV3z1CrGc5zZ+SMynCUlaWSXt
1FsR3jtZeqw8vfYeLWbDLz/3ABP0Mplom1lUYAPYK9P+WLKy9PXiYFWfIvbWbEtg
cfIwUpmaj3/9DNEp8pVI8RXRSkr1AuZkKX105cDzjgTn7uRMfPRDyVppl5ila7d/
p7Jq7xy90lg2zUXU0UuCXhkCAwEAAQ==
-----END PUBLIC KEY-----`

var pubkeyb64 = "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUlJQ0lqQU5CZ2txaGtpRzl3MEJB" +
	"UUVGQUFPQ0FnOEFNSUlDQ2dLQ0FnRUFzSFpRNndSTS9WNXI2REdDcjJpbwpVczEw" +
	"T1JheUlQWkVtNFJ3YXFKU2Y4S2RuYVdhOHNQZFFJbnJwZjBsOWIyZHFPSFdrNDVw" +
	"YkhxUlJleWhPQzhJCk9tbWRmSXdxYm14cXpuUXhDWHRsZWsrd3dyQTdLWGRyVWty" +
	"NGVJSGJkbzFnNlRGQkd3ZVJtR2tsR2t5Wm5MNVgKV2tNWUZnQ2JuN3MxOTFFcm9u" +
	"L3l4ajBXdUtEM3dwZ1pvTjdxeW1UMWRSTEVROGJnSUU0WUQ3UDdRYnBjRjMrRApp" +
	"YnVFUW53R1FxM1lYQnlCa0ZCOTdzVDNjUjVqM2Z2ZlJwd1UweXowYTdxRXp0Nm5F" +
	"NVJXcmtoNGJDUTZPNHg4CjNZdjZqSGtPampNdFNUVlRsNE8zNW51QWFYRXB1NEo5" +
	"S0E2VXpXdzN0eDF6UHNFNkdhaTd3S0kxWmpEOENicHkKU1M3emdrSmR4WWgzRmFn" +
	"Q3dIN2U4emVDRW5YbWdIR01FaUJPZWFoN1MrejE3WlRhSHFzZW1sMjBRR1NEN0F4" +
	"egpMVjlLWHl0NjNmVno4UGE5enAwOW4zUS8yakpYRlFvNzYyQ0dKbGVuT1dOejlL" +
	"ZFVub1MxOE5ZSUdDMi9oOTRECjdWbktDbzVKRVJyRy9Xa3Z6a3hvSnMzTGZESUw1" +
	"VkhFUlI5T3FsVnBWM3oxQ3JHYzV6WitTTXluQ1VsYVdTWHQKMUZzUjNqdFplcXc4" +
	"dmZZZUxXYkRMei8zQUJQME1wbG9tMWxVWUFQWUs5UCtXTEt5OVBYaVlGV2ZJdmJX" +
	"YkV0ZwpjZkl3VXBtYWozLzlETkVwOHBWSThSWFJTa3IxQXVaa0tYMTA1Y0R6amdU" +
	"bjd1Uk1mUFJEeVZwcGw1aWxhN2QvCnA3SnE3eHk5MGxnMnpVWFUwVXVDWGhrQ0F3" +
	"RUFBUT09Ci0tLS0tRU5EIFBVQkxJQyBLRVktLS0tLQo="

var privkeypem = `-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----`

var privkeyb64 = "LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlKSndJQkFBS0NBZ0VB" +
	"c0haUTZ3Uk0vVjVyNkRHQ3IyaW9VczEwT1JheUlQWkVtNFJ3YXFKU2Y4S2RuYVdh" +
	"CjhzUGRRSW5ycGYwbDliMmRxT0hXazQ1cGJIcVJSZXloT0M4SU9tbWRmSXdxYm14" +
	"cXpuUXhDWHRsZWsrd3dyQTcKS1hkclVrcjRlSUhiZG8xZzZURkJHd2VSbUdrbEdr" +
	"eVpuTDVYV2tNWUZnQ2JuN3MxOTFFcm9uL3l4ajBXdUtEMwp3cGdab043cXltVDFk" +
	"UkxFUThiZ0lFNFlEN1A3UWJwY0YzK0RpYnVFUW53R1FxM1lYQnlCa0ZCOTdzVDNj" +
	"UjVqCjNmdmZScHdVMHl6MGE3cUV6dDZuRTVSV3JraDRiQ1E2TzR4ODNZdjZqSGtP" +
	"ampNdFNUVlRsNE8zNW51QWFYRXAKdTRKOUtBNlV6V3czdHgxelBzRTZHYWk3d0tJ" +
	"MVpqRDhDYnB5U1M3emdrSmR4WWgzRmFnQ3dIN2U4emVDRW5YbQpnSEdNRWlCT2Vh" +
	"aDdTK3oxN1pUYUhxc2VtbDIwUUdTRDdBeHpMVjlLWHl0NjNmVno4UGE5enAwOW4z" +
	"US8yakpYCkZRbzc2MkNHSmxlbk9XTno5S2RVbm9TMThOWUlHQzIvaDk0RDdWbktD" +
	"bzVKRVJyRy9Xa3Z6a3hvSnMzTGZESUwKNVZIRVJSOU9xbFZwVjN6MUNyR2M1elor" +
	"U015bkNVbGFXU1h0MUZzUjNqdFplcXc4dmZZZUxXYkRMei8zQUJQMApNcGxvbTFs" +
	"VVlBUFlLOVArV0xLeTlQWGlZRldmSXZiV2JFdGdjZkl3VXBtYWozLzlETkVwOHBW" +
	"SThSWFJTa3IxCkF1WmtLWDEwNWNEempnVG43dVJNZlBSRHlWcHBsNWlsYTdkL3A3" +
	"SnE3eHk5MGxnMnpVWFUwVXVDWGhrQ0F3RUEKQVFLQ0FnQitycUlPNW9xREJ1cyt5" +
	"WFNCaXdmMFVlMFRJdmtFY3VmMElkTTJxb3ZCanF6cXhUNEU5Sm45UUVaOQpac3gr" +
	"cS83b2hDRXcwM2RaMm5BNm05TnQ2MDNqNlhpWE5tVXIyd2VlYVluZXZjaXZVMUNa" +
	"cEQwRTJ1ZWdMNVJMCnB5WXY2UFZlMCs1aWdqK0RCRkVQblZoV1Q4dVVVRUNWWXlC" +
	"V1B1ZFNRdUtwaVdOMzc5bEUrTUtGMy8zZUlNcTgKUEZoL0VOYjN0V21ucDRqY2xT" +
	"QlhJbndFbnBXSEpxaWZ0andrV0h2UVBPTERBUlkzZVE0UEZuc3BuUzNBbWtMVgpE" +
	"QnY0enZHVE5nTUtLbDlFUldDMmVoZVlNcFpkMHFVdmZhVDJiMFVlbm5zQmRoMXJD" +
	"TlM2WGZSUTJqQVJ0czRhCjM0cnNHZWRuY1A3Tjl2VzdLSGhmZ2VFZTlzd2VEVEdG" +
	"dHZmRHMwN0U4ZzdRbFN5VzF4Z1o4SGZLQy9tRFYyWEQKM045OThhaDFJYktCaEVT" +
	"VlpCUVRhb1IyallIRjBwbCtSUFFKQXB2TVluWGQwZzhWK3RaQUs0YlhtSHYyN2J1" +
	"cQpIQ2t5UTZ4WnZIMUZablpJQkt1ejNyK1c5SGpYYnVPMk42WStqK09EbWZGT2Zj" +
	"b09TQlV4eGdYRmc2ZTVUc053ClplNVZiajJQQmxZSnFINDlxVjFmTjFMNmd0QU9x" +
	"NWxleVhxWmhKVXladmp1c1FXSE4rRXVXV2lJVlllWGM4THoKMXFRZ1JsazJZZTNZ" +
	"ZGZpUVdDNTFYWURlL3BJcVdwbU91MUR6aEVzaHFpWGlubDh5bSsxcFl1d09GMDNX" +
	"c2lOdwp3SlQzMEdoaXl4a3JvNmd3WWMrcmsrTjdrZFJpUWpySlZycHVXeUpyWm1m" +
	"a2NaZUpSUUtDQVFFQTZ2MFlWZHVqCjMyMzFUQWUrQkdqc1ZtN2ZETmtXbGszak8r" +
	"b29PcjNSaWkrRmFleFB5Q016Q3JrOWNmQytZZjdvMGR1OGVWNkMKK1dzdG1oRWhC" +
	"SXZpSGwxT0VXYmpUTlNaL01TZEZFUXN2eVRTSG1Wc1ZGbjk3emViT0F6MjcyUmtP" +
	"WEFTazBpMQovTVMzWTlKc2pyQXN2cFdTVjhjSzZqdkQraFp2LzN0S2JEa3M0djdS" +
	"c3RybExzRFRYcysvQXpaT0ZVVmhyTHpOCnIwb0pkdXpaeEZrQ0t5cmJRbkVyYkN5" +
	"cmIvRDQyVGlVTUZkMWgrN1FwZ2tTZDNuQTJsTVl0OGVjS0hqREk3M28KMjk5ZUNi" +
	"YU5UcGFtVXFYVjlGb3Vva2diNld5bmhCTmxTN1NHVEljYXlSZHNLYWdkT1p2ditS" +
	"NnROM0JzWXFHdQo5Vi8wcTZkMGJHL3dLd0tDQVFFQXdEMkwxZmFOUk8xVTZvb2hQ" +
	"TGJQRjh2OHI0Mys0U2pwZkVxZVFRaXFLSCtQCkcrczNqYmFsVlVwYlhrUTdpUk9U" +
	"dHFuVExlVWJOMkc1eWlZWWRKRFBRN01OT2V5RDhyMXdIM00vL215ZU52WHgKQ3c4" +
	"Z0QzeE4vZTl3bHZBSlNNWGxhNTR0bkozdmZhY2F5SUF2dHZyRUw1d25PTlU3RXlQ" +
	"UkExOG9oNzYwcThZTgpBUXV0N3hHT0VoZ3pXdVQxWDZ1OGd3TG11RXRoTnIxcmkr" +
	"OGF2ZjZpM1FCNXRUSmhnL29rbmplQUtsQStwWnVwCjdPL3pwYnRLQmJMZkh5YVJs" +
	"NXN6MFVmYmhkRXFaWmdCVUcycGM4TE04TjFDRjlFdm9FOWM2bU5COFc0cnFrQy8K" +
	"eDJ0VUl3dVFqS0VLa2JYRGVQTS9IMWExSmE2elFJaTY4WTRINDVyRXl3S0NBUUEx" +
	"d3lZWUxxSTFjaURXL2taNQpGOUJLamg4MS8wenRvbkJFdXZQdFRKUnVPeVVZOE5u" +
	"bi9qV2xWSEE4YTBvRGZhQ2lzdFZTSjA5cjVSdVB6aTl4CnJOZFUveC9uVjFUVnRT" +
	"WnQ4RVhINXprZG1qMEFlMC9ubEpkR2JjQnplSFBlbldkWXhNMWJLUjJKOFMvTUJN" +
	"NlYKYnJVdC9XWjM4ckFLbXhYaFY5VFQ3TTlBSjJ5Zm1wRTdqRjAyN3lMczVEYmRj" +
	"MFUyRktPZU02d1RXS3NGckhhKwpOMmNKblVxQXp3ZVNQajRTNUZ6cXhja1JybERU" +
	"Z3MzMXpzbU0wQ3hSUndXMnRsS0I1Kzh0ZER1Y1ltUlBjSmF2CnprUExVT204ZUEy" +
	"SFQxd2pjWnA1Mno0bnJldTBBbzBjU09HVVBrUkJjKzNaWHkxZUs3aUFjR0ZvL2tV" +
	"cUtLdTQKUzN2MUFvSUJBSDRKRzkvb3FFL3paY1ByVWNVemVXejVvUzRiNDJvTlg1" +
	"N01jUnJrS21NbzFsT1FrRGlKOGJXTQpiWUROTFZjK2pZNnBvcm1wUm9HMXdaQW1E" +
	"OHlFa0U2cldsV0ttaXVRUmExbzZ5RE1aNkpTOW5pd3J1MVlLdTM4CmlJMTh6Q2w1" +
	"RFdQVUxjVkx5cE5QOW9CVGduVHR6YWdGTWJYU0hzdjZwSE1ZZFVNaUplT2traXdJ" +
	"VXoyMC9iY2gKUkxJb0FETjhMYmliTTFiS25PNjltNEFBQUVGbWE3S0hPRVF5eHJv" +
	"M1NzQ3NWSXZwVmxsUFNFWCtQM2g5NVJiOQpZY2xUaVFxamg0S0RJUXFIeXNzV3NH" +
	"M2hwOElzaWg2MGdUdUtPelpZTWV1OXJhTXkvcys5YWI2OXdFakZzVHh4CjdMTUJQ" +
	"eW5TR0tWY1BLRjYreXlwT0I5Y1poRzBDNmNDZ2dFQVBmUEJ2T2tldGlLdHZraEI3" +
	"T0FpajdSQkwyKzgKSUNrK1lqS1FoU0xqdXBmTFdXQ3YzbmxxWmlZcXhFOEZOOUJH" +
	"SGRoQzUyaEhKVG14ekZ2d0VLK3k1NW1kcXQvUApjVm9nVXhIMW1MU1RGOXBidWlT" +
	"Sm9FMWlwQlUvemNtT2FoZVBCOEFZc0Z5WGVROGJCRG1lYW1WR2h5YTNTWUU4Ck9u" +
	"aHF1VWJiMFF1bm9qLzNRTzJzOGxGbCtIMkRML0FyYmcxc0R2Ui9rTm5lSW5sd3U3" +
	"ZkJNQ1RES3B3S1ZtdXUKSTZzNnFJb3l5UWpjeGQvTWMrOFNwWVlacVZxK2dkQlcv" +
	"bXBMclNTS2VUN2RTbzdXeTU3c0ozOGR3cHZlSk9TWAp6bjRjUDU5NGJQNHA4MkRy" +
	"cm5MelV2WHZGRXdqeUFCUml2OTlwa3ppc3F1c05aTmY3N3RYVXR5c3VnPT0KLS0t" +
	"LS1FTkQgUlNBIFBSSVZBVEUgS0VZLS0tLS0K"

var pubkey2b64 = "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUlJQ0lqQU5CZ2txaGtpRzl3MEJB" +
	"UUVGQUFPQ0FnOEFNSUlDQ2dLQ0FnRUEzMnIyenFpcHJqL2crSEh2RE9TTAo5d0F6" +
	"WGtOcXpoaW5PQ1BKcklvUytBWnduTmFMTHVVVDhMczlpUG1TYk5rTGcvbFltOWQ0" +
	"L29ZT0QvK1BBdCs2CndZcmRaUXA1ejdPRjRSYmcxRTBkdUEyQ2toVVYyZEZoT1l0" +
	"cm5DOThKL1dLRE1UVEpUUHNla0h2UU5lUUZTTnkKcktpb2lJa1BLRnVMcDVoZVVD" +
	"ZHF3WkdacUphRTcvTzNlemptVzIxSEo1Z01weWUzMWtoSHVFMXhYc3NoL1BNNgpK" +
	"QjlkL1VtdUVjQUNrU2NXY1Iya1kyS2JZdC91V3hKNWltWTQ0V0pHWDJnMUdXemJm" +
	"SVh3Mm9IMi9DWWNYTUVlCmU0Wk9WRzhPMUk5UmNLMFJJbmdxQWo4UzV6MSt2dkZU" +
	"ZUJXNXRjR0R2NHBTUUh1TENpTll3RU5TazVBU0RoVWQKbGRIUHkyWHJQMkVpY3M5" +
	"ZFM3YzRFQ1VIaTgvLzFkZVp1akllbmw2ZHRqcjR5TVdBNituaHFZcDZDOHlPYmJX" +
	"RQpLSVl4c0pQcm1QOTU2NzdERVdSY3B0ZW1jU1NJcUdlTFpLaVhCT2dSL21lUitw" +
	"ODRTOVdxVllUREJaUWR2ck5VCm5DaE9RVWxZUk12Nnlablg2Sm8vYVU3bnhnNDdw" +
	"THVMdEh6VHpMUW9OSHhVUjBlSEY2VGpGRFlLbHR0L0J1SkgKazAwazAyVTBRK01D" +
	"ZU5icVEvbCtxKzRTTEt4cGVZNDg4QlB6MU1aQXdheVpHQ1I1T1NzNW44NllKKytF" +
	"SXRuMgpuR2RZRFpFVVpYaHVlem1KeWp5Qk4veTNCeVZnMnFGNkFWZi91Zng4WWpj" +
	"MEltT1J2WkJ0NWViTDlnS2daZ1VrCnhlM2xxa0ZQOVNDRm11eEt2dXpyWTlrQ0F3" +
	"RUFBUT09Ci0tLS0tRU5EIFBVQkxJQyBLRVktLS0tLQo="

var privkey2b64 = "LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlKS1FJQkFBS0NBZ0VB" +
	"MzJyMnpxaXByai9nK0hIdkRPU0w5d0F6WGtOcXpoaW5PQ1BKcklvUytBWnduTmFM" +
	"Ckx1VVQ4THM5aVBtU2JOa0xnL2xZbTlkNC9vWU9ELytQQXQrNndZcmRaUXA1ejdP" +
	"RjRSYmcxRTBkdUEyQ2toVVYKMmRGaE9ZdHJuQzk4Si9XS0RNVFRKVFBzZWtIdlFO" +
	"ZVFGU055cktpb2lJa1BLRnVMcDVoZVVDZHF3WkdacUphRQo3L08zZXpqbVcyMUhK" +
	"NWdNcHllMzFraEh1RTF4WHNzaC9QTTZKQjlkL1VtdUVjQUNrU2NXY1Iya1kyS2JZ" +
	"dC91Cld4SjVpbVk0NFdKR1gyZzFHV3piZklYdzJvSDIvQ1ljWE1FZWU0Wk9WRzhP" +
	"MUk5UmNLMFJJbmdxQWo4UzV6MSsKdnZGVGVCVzV0Y0dEdjRwU1FIdUxDaU5Zd0VO" +
	"U2s1QVNEaFVkbGRIUHkyWHJQMkVpY3M5ZFM3YzRFQ1VIaTgvLwoxZGVadWpJZW5s" +
	"NmR0anI0eU1XQTYrbmhxWXA2Qzh5T2JiV0VLSVl4c0pQcm1QOTU2NzdERVdSY3B0" +
	"ZW1jU1NJCnFHZUxaS2lYQk9nUi9tZVIrcDg0UzlXcVZZVERCWlFkdnJOVW5DaE9R" +
	"VWxZUk12Nnlablg2Sm8vYVU3bnhnNDcKcEx1THRIelR6TFFvTkh4VVIwZUhGNlRq" +
	"RkRZS2x0dC9CdUpIazAwazAyVTBRK01DZU5icVEvbCtxKzRTTEt4cAplWTQ4OEJQ" +
	"ejFNWkF3YXlaR0NSNU9TczVuODZZSisrRUl0bjJuR2RZRFpFVVpYaHVlem1KeWp5" +
	"Qk4veTNCeVZnCjJxRjZBVmYvdWZ4OFlqYzBJbU9SdlpCdDVlYkw5Z0tnWmdVa3hl" +
	"M2xxa0ZQOVNDRm11eEt2dXpyWTlrQ0F3RUEKQVFLQ0FnRUExbnEyTXhDaHpHRVFs" +
	"Ukd6ZnJvTlQvTUdYVkQxUUtOVUxNWFdmdWdTYTc2cTd6WGJhZ3FLaVFrSApldTYw" +
	"VGdCVFdML1ArOVB3R05BU3dmTUJsSzI1ZU1IWjVuMFhFWGp6Wm5IekpueGRzbXB0" +
	"MWRXZUkzd3BEUGcyCk56c3l3cDJxaUxXUFNlQzkvV1E4emcvakJ4Zi9wNWRHSzhV" +
	"QUl5czNONDVEeEVrQzZJN2haNElRWHRhbVp3bnAKd3cxMlNLRmtURGdKK1JGc29K" +
	"YmY1ak8yRGtKbHAzWGhZaDRRbUlPdk15L0dFSk4zVTkyKzlEMnJjZ3liVjJ6bwo0" +
	"QjNiRnc1UUkzZG9nZk9IbEEwK1VUUTQrQ2FCSCs2QVZmaEQwWFZBcHAzT25EdWxn" +
	"SUNTb2lGY0F1eGp5QjE3CktleDdrUzYxNWFkMGhDZ0l0SklzdlBLTEVxL2N0RUxq" +
	"bzJCQlgvMVc1Z0w3ZmlhczVlTXRPVlhKazZ6azMxNkcKRTl4TVhCbU1DZ29aRmNT" +
	"elZGL2tFU3N5b3hMcXFTNlU0blBseGZqbVJtNXpkU0N3QUNVRjNTUUxMQjhHMnEr" +
	"LwpqVEdVVHhLMDg1dFNyU0x5b0lJMGRjYjFNc0gyZkZOaWdtbHgxU1d3Mkdaemlm" +
	"MHQwVDF5TjlHWVhuMGJtRlJ5CnBuRXk2eFJBVWhIWUhBYjFvSDRPemVlMkJmQjNz" +
	"QUFtbEZCU2FrYzNDWG1tdjFzRU1tV1ZyZkdVRlJiaGI1ZG0KNldWeGFyeERCdVlE" +
	"b3ZjcWNJeE1JYkV3OHZLQW92YURQalJaVkdQZmpDQlhkQTRQcERLUzEvSjdXRmox" +
	"UFpiTwpVUlNFcUg5YzdYT3BNOHo2TCt4VDNLL1NsMGVLTEFBRTU2SmxaQjB1bE5a" +
	"K0JEL2VhWEVDZ2dFQkFQVXpwSzQ3CmlsMmFMeDh5MFpNaGpVOVRYTnZRcmpXN25o" +
	"VnROQjJ0Z0txL1lxb0JoUXU2dnZ6WjdzQ3FlUXZxU0xHV0tJVjYKN1FpQ3k1bnNr" +
	"YmRiZFlYM1Y3TnFCS2tDcnE0ekJsa2pTQlMzQ0pzN2JEWkNCcGlQZ0luZFg3T2xj" +
	"Z0Z0d2RrRgordzA0TFdRNzgrN0IzeldLdElvSEIvUXNSYjU4Nm9Zd3JKRHFuYjJn" +
	"RzFtM1N5bVIyWWd6cG5UeGo0VXYzdDBuCjdqbVJuRlBkZ2hCYlFqZFUzd3NFYldQ" +
	"ZlRPVUNZaysyS1hKN3duSGdCaTYzL2pHL2ZvaFphZEkybkFicEZkUjcKTDNVNDRy" +
	"dWllUjAvVHE3K3gwMjFQMDMvSTNidkxaREhqdDJZbW00aHlrWEx3R1pzYklSbUpI" +
	"RFpOajVSOUE0NApHUjFxY2R6Z0U1VjVacTBDZ2dFQkFPbEJ1N3BueTRNcUhGMnht" +
	"VzA3VkRscStPS2FQSUFNVHRhbDB6VWI0RHk1ClhLUjBmRHdnK3JHb2xzTEZpZm5C" +
	"ekgxWkU2UEFJSlZEbFo3ZkJWbHlsV3A2OW4xY3FxRFFkTkNoTWplMEl2VVAKb1Jn" +
	"NHdEUWE5SUNmSHJNS2d0bVR4V1o3NDVTbDkrVThqSkFlc0xvbjRiRGovSlVRQXZj" +
	"Y0Y5M1RRN3p6RU8zOQpxSEI3alg2MjJGUTdJcVFUZzltQzZ3dmd4V3JHNjE2TzRC" +
	"MCtOVENpNEU0aE0rYTJ1c2dzenNkelc3dDJhblBICitxUk5TdG9MWlY5YVphNStX" +
	"QWNwWlBRRHZTNkEzZ2xIMFFzQVZkOWhONCs3Qy9Zc24yWGRnMHlpL3ViM0MweG0K" +
	"NTFSSFRyZDlyQ3VqTkVZS3UrN2JUZE1VU2ZTT2tlU3UvWkVLOXIyRFUxMENnZ0VC" +
	"QU1ZblAvUlZ4Wkd6SWxXUwpHZTlPOUFXaEZxL0ZTcU85eFJrSHNWQXlnSUo4Tzky" +
	"cmNMdHo1Umd0ZmxaUTdaV0ZkYzJkelkxaE0rRG00bWEzCjJXSldGUGw0VTNWNFBk" +
	"L2ZmUTdseVVHTDA1cDUrQWlLMHY1ZUNUcU03WkY4UnZURXhRY0dqZHMyakJXNHlt" +
	"WHcKVlVjamdhQ2hRUmt5YVdrWHhoMFVrZXB6dDJFOVdOQi9iTnJwMTJIMnJkYjE5" +
	"cFVYQ2FiV2NzSkNuTEFGVGxJdgo5Z1lGMmRNaFVVWjBBM3JzWUJYS0FXenRoejB2" +
	"YW9uZ1F0N0tiakFCMHQyWmRIMGZDS1JGQlJFN283U1ZqaFdZClRVd25kd3pRZEh3" +
	"Rjl1eXZQUmZHWGdwY1dzWVZwdWROZzBzNFIzbitNUXdtQjFqekVIVDRnY1JqN1Zm" +
	"cVI2MzUKbjVueUM5a0NnZ0VBZFc4OXQyeDRYcEg5OUFIdEw3eFYvQTVxUFpQUGI5" +
	"eENlUGpGckJCYnhkYkEySjg0eVFFRgpsaG85eE5PMVVvUUtrdlVjMlMxcWVodXJv" +
	"Vy9BL3JhY21SNU5LUEpWaVY5SjRKdTNiNm9HaTRDUjUyTHpDWWlrCm5uajkrTUFL" +
	"L3NYUjlYWGNMME9iMmRLeEpnSDlrY3R3YWlGdVVoSGNuRktOaFlYT3JidG1RNXVZ" +
	"aVFEN1ROZDcKZEhUTlRQUHlScmtONDA0SHRtbHRxSTZnTUxqWUNLT0g4RzN3OW0v" +
	"Nis3cnJaQ2trL3Uxd1ROaGF6UXVJNnR3Zgo3bkRSanBkWGRFdVg0dTVhK2FXeG85" +
	"Ui9YMTJNM2tqUXMxRkZoV3dUMkRJODM0R1VlZTNZeDE5cmlkZll2ckxMCjByQjVL" +
	"ZVpQbGNMZG5LNnpTU2ZhTmRzUUdFei95b3ZxbFFLQ0FRQTN4VHhvM3NBVlJQcmtr" +
	"bitnREpvR1lEZlAKOGNHYm1JS0lFcyttT1JxbyszUU5UNitaT3VaajFoV2UxcUt5" +
	"QmY4dWcrcCtQUHBrVEpZNnJqbVAxUGVLL25ocQoxM2tzRFVrM2RObksrRnJLN2VL" +
	"V2c0eXRSSVlRT1NkNEVTSjcxa0JGNE15ZzRQNnlkZVY3bytSWi83bW9ON3FrCndY" +
	"ZWtHejM4a21Tb3NPc09QaVhCTWpFTUtwMVRkMWkrUDlYUWJZdUlKS1JTK1hXbDNq" +
	"UjVQOVhUa2ZyRmlpY1MKdnlCOEN3Y083REh3ZmRKZ0dXenpweGRYcWk2YTlpYjNR" +
	"WDJOZnBuUzZlaDZzK3dzaHRabkVMTDZ2Q3B1dGJNYQphME56bUVSeWxtem5zYURQ" +
	"TlpUNXo5T3lNczhzMjRwbmdabmJWZ3J2UmgvMTk0aytTZGF4ck5EM2Zwc24KLS0t" +
	"LS1FTkQgUlNBIFBSSVZBVEUgS0VZLS0tLS0K"

var pubkey3b64 = "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUlJQ0lqQU5CZ2txaGtpRzl3MEJB" +
	"UUVGQUFPQ0FnOEFNSUlDQ2dLQ0FnRUErTlhtLytHZFE4d3pFNTR0N1dsYQpDa2VW" +
	"KzJrL0pPTVpBeWFRMms2VzYzb0pzN1VNaW4rVmphRVJPTEttM3V4Tk5WcUxyRzJ5" +
	"NlV2M3NsNUFtb2JzCjF3RS8rTUJQMmhLQVpodWFRTU9hUzloK2YzSmF1ak1WU2Q1" +
	"bHdlSFBrQmthWUtHSGpDMjRCMHl3cmlmbWdJM0QKTlVOS294d01IYi9QbGJVbzVB" +
	"Qml4M3A0SC9xTG1uRnB0SmppTVJUZTV0OGV5UFpwdElTZmNzeUgvU0h6Z0l3Zwow" +
	"ZitEOE9pNldub3VlZ0FaUC84cnIxUlFqdEFEQWxtSGhNWG1MMVphcmRaYi9FNzZE" +
	"RXdxa2NMVm9JY0NXT243Cisvdm50NUJBSHB5cFN4bHUxSGUvdktkRlZ0Z0t5dWNQ" +
	"V0F3OFF3ckFYWERwL005czZpUm9kdDRYNXhYeXNnMm8KWjRtSyt4TkQ1NURiYmhi" +
	"QUYydHpWMzgwWkxtL2lGMXFZa0Fwb1FFWkppbEpjYng1cUxyTzJzc0ZtVk11RWlD" +
	"MApGTkdqUjVTQlFleVd5b1V4N1NjNGw5NTZwMXBRNXVDcy9QV1BYTjYvM0hybzcx" +
	"WUkvb0dNd2c2MlE5dmF0NU9qCklyVG83UlA5Q2lRNldiY0hucWZlclFhVXlicHV5" +
	"TVdESVJra29EQVc5REpKNGtrL3VXS29wSlJQZUN5cHo1dHoKZVpheTF1ZkVJbU9h" +
	"MUY3eFJ6WlV2bUc2SlZPQ09KdU8veVIyazdtcVRBWFhycFdCMjJ0VUw2NjluV1ZS" +
	"UDJKUQpLM2RMTk5YMC8yY2dFaWtXbXAydTJ3aVBZSWhKRTF2aml5RFlWV2w2WUVQ" +
	"MHF6dUhMOTA0d05JUVdLOEtCeVp1ClFwM0JObW1CYTRNTFc2QnNhQ1Z0d0cwQ0F3" +
	"RUFBUT09Ci0tLS0tRU5EIFBVQkxJQyBLRVktLS0tLQo="

var privkey3b64 = "LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlKS2dJQkFBS0NBZ0VB" +
	"K05YbS8rR2RROHd6RTU0dDdXbGFDa2VWKzJrL0pPTVpBeWFRMms2VzYzb0pzN1VN" +
	"CmluK1ZqYUVST0xLbTN1eE5OVnFMckcyeTZVdjNzbDVBbW9iczF3RS8rTUJQMmhL" +
	"QVpodWFRTU9hUzloK2YzSmEKdWpNVlNkNWx3ZUhQa0JrYVlLR0hqQzI0QjB5d3Jp" +
	"Zm1nSTNETlVOS294d01IYi9QbGJVbzVBQml4M3A0SC9xTAptbkZwdEpqaU1SVGU1" +
	"dDhleVBacHRJU2Zjc3lIL1NIemdJd2cwZitEOE9pNldub3VlZ0FaUC84cnIxUlFq" +
	"dEFECkFsbUhoTVhtTDFaYXJkWmIvRTc2REV3cWtjTFZvSWNDV09uNysvdm50NUJB" +
	"SHB5cFN4bHUxSGUvdktkRlZ0Z0sKeXVjUFdBdzhRd3JBWFhEcC9NOXM2aVJvZHQ0" +
	"WDV4WHlzZzJvWjRtSyt4TkQ1NURiYmhiQUYydHpWMzgwWkxtLwppRjFxWWtBcG9R" +
	"RVpKaWxKY2J4NXFMck8yc3NGbVZNdUVpQzBGTkdqUjVTQlFleVd5b1V4N1NjNGw5" +
	"NTZwMXBRCjV1Q3MvUFdQWE42LzNIcm83MVlJL29HTXdnNjJROXZhdDVPaklyVG83" +
	"UlA5Q2lRNldiY0hucWZlclFhVXlicHUKeU1XRElSa2tvREFXOURKSjRray91V0tv" +
	"cEpSUGVDeXB6NXR6ZVpheTF1ZkVJbU9hMUY3eFJ6WlV2bUc2SlZPQwpPSnVPL3lS" +
	"Mms3bXFUQVhYcnBXQjIydFVMNjY5bldWUlAySlFLM2RMTk5YMC8yY2dFaWtXbXAy" +
	"dTJ3aVBZSWhKCkUxdmppeURZVldsNllFUDBxenVITDkwNHdOSVFXSzhLQnladVFw" +
	"M0JObW1CYTRNTFc2QnNhQ1Z0d0cwQ0F3RUEKQVFLQ0FnRUF1MVYyS2kvMWtNUWJz" +
	"K3BERTFoY0pCOE9xQTd0TGQwV3lJdHhSQmtrZjdVSnR0UlgwN0VIcTIrVwpJb1JG" +
	"SXREdHMzd3VhU3JSSmRnK2EzZVAxWVk4cWdWVDN2Y1JadERFLzVwS1AvWENwTlVo" +
	"THR1dHVENmJDVmk0CmJRV09tU0o4L0VDL1ptWkpCSjNVNmRnNkxaQU1aWDM2bzkr" +
	"S3M1N2pMZ2NMK05MZGl1WUZwN1dkQWpIZDdjdW4KaG1IN0NmN3lFME9JQXhKUlpF" +
	"RGRKRkk2R3czajY1VWRCUEtBMFhyb29JcVFkK0NvUjhBSFFlMFNSdU9XSmZ4RApO" +
	"bUlodEh3TUZtQnkyVzFDSXloMllmc2laa1FKcEFSYXg1Uis0VXo2R3dMVHNIdFN5" +
	"emwyOTFHQWxvN3J1MUxDCno0bzVsbDlhbVN5a0I0WlBheVg3QXN0QVFwSUx0Q01P" +
	"R2JRbUJ1dXR4L0pJKzEvVnAwOTZmcjVndFBvSXh1VGgKbE5MdUhmSXBZQWZ3Y2gy" +
	"WmlUb3JvWlMzSFFQR3M5eERFZ0JIQnBXME5kWWVIZTJ0SEQ2ZzZmWG5Ca2pvUk5G" +
	"bgppb2FQek9mckZ6SE5YMUMzTFhKNmpBbjJkSWxuVEc4amdvai91Sk9aaW1qeFda" +
	"YXRjK0p2bm9ZMEk1N3l3M3JNCjZBa045cDhsWDdVcVJEa2JFSjIzbktWTWhTSURX" +
	"T0hKQmdCQngzQm1KdG1tMGYwdzJpMlVkbUdNNjY2ejZ4c3AKc0dqK0JiZkl6T1ph" +
	"NWc0T3hNMnlnRFVoYjBVWkxmb3F3UFJGdkFXYk1uWmNqNkZRTTQ4UFlWOFNOUGNF" +
	"Z1lmNwozOUsyTmhUa3VleEpzaWlPSkxkclBQdHhtUmltdVVyTkVmWUQrdGVGeFp4" +
	"cmFkaGVIVlVDZ2dFQkFQeTBQd3N0CkRadG44c3pybkwwSDk0eVdwVzc3M1NVWVc2" +
	"MUJSTW5xdzl0eTlBVTJqaGtuZytTMVB1TkF2VlJmRTB6M3pXZkQKQW5VRWRpNTVZ" +
	"OW1SeFg2RjlHOEhmcWdpL1QxeXVvcTlzbUVWak5aU1FReWJsZS9Jd1hGdUd3K3l5" +
	"c0hpWEN1agpsUndtTUtkdXA0RDZlOUtZdkJ6OXd1djk0T0o4SmhUdURFZkJIQXVR" +
	"dy9EQnpXdmpBemVIS1pjMEEzOUtIS1d1CjZ5MDdRNlBNbE9Nc1AweGtiREZoY0li" +
	"dVVMeVdxWUdZeFUzTkVPdko0K2E5ZXNodFpoTWlrbXlvaG9OcVBYbHkKNzBzeEUy" +
	"ckdhSWVLSGM4WGRUY0ZjZWRLZ1U4ejUwbkpvZE1iT21mYU5vVnBKSnI2aWN2UkdX" +
	"QWRFOVpNV3lEdgpTb2pOMm9lcVBuTHNvSGNDZ2dFQkFQd1V2VXZiMzA5TVh6WnJl" +
	"cnZ5S2M4ck1vLy9vd0gvUTBib2FlUFk5YXRGCnVuT3dTRGt3TnliMWxJTG5RYjIw" +
	"OWVGaW1pSjhNL25LY1BlN3gyckR2NlBpVkFhZXppRmYwbjlDTWRiRHRrMm0KUDZS" +
	"NDJaUWFLcVRURzNDMGdIOXFUV0lSS0Z3cHB0TW5qTlJ5NU50WERlcUFTT0ZLRHJO" +
	"ekpwclpERm9ibVU3eQordjdxeXFjVlU1WXhQaVZPTGJZSy9WWXREZjdSc0EzR1Vh" +
	"ZUJtRk1pRU5ySWtBRFdJeU1jY01weGhQbkVTNmg5CkVwdWNwRUJwVis3bmNYNEVk" +
	"WXJyQUpUWE9MRFlWbHNQNHA1TFdYdndmRW15M0FEcEh1VExJdlhWK012WkJpYmQK" +
	"d044SHdhbTB4MUd2OVl5b2R4YnBsNzh3Yk9xZHR6Yyttb08vUXRaUG96c0NnZ0VC" +
	"QUlDSFdNMWhZOXRZR25rNQptU1hZQ0lPY0Y1YUVTZTFWSDBQV3Y1c0hhZ2lTeGlS" +
	"a3BBK05OcHM3eURtanN1aFgxeVE5b1Y2V0pBaktkU1djCkhqb0oxMjVMeVpBek9x" +
	"dGY2SGU1ZzhHUFRFdnV2d3cvRjlER3paTUJBOHFpbXViNEpBSkxGR0FwdW14dnpD" +
	"MU0KcmF0L096MVk2OHkvRU1ZaEFhR1FUWG8xdlU4OW0vc1I3V2JsMjRwMUV5eko5" +
	"VkJ2Wnh2MTRPNHdNbk4yQWlncwpwTW1LdHNNdkRJeHRKK09wRUsxcTM5b0hqc3JN" +
	"Y3ovS0ZyMUVRRS95dklrYysyYUNySy9vZUUzdU5HR0ZHNEFpClhBWWNMSjhwS2pn" +
	"SzcrMFl5djR5d3YrWHExUUpORUtnRFR6N3hsK0E4RmNYQ2hZRmxCSmFFYnVGbWQ1" +
	"TS9Vb2wKUDlBS0pHOENnZ0VCQU0vQ1VCbTdoNXNWdU10alhlNlZNZnc3QUJ0S2VC" +
	"RG5UNDJiYzlxRU1FWU8zQk1KVVIxQgpMZE9BUi94em1PMC92ZjhhZ3lxMDd5bUt2" +
	"ZnlQMlZXWEs5Vm9iaFJld2tramJwdlA5TCtxNGcreFczYTAyNjZDCnVtN0tSeTFt" +
	"dHhsTWhhYXp1VzNzTGtDTnNqWk8wMndyblo2T1NJTFZ4TFFGemVXRnlmWmlGTUxL" +
	"NUM5QlYvREoKVlBET2VRZktIVWFTWENXd1VINmFWOTJpZkIzd1k1ancxSzljNmNL" +
	"bXVxTHZoODV5TFVTbGtpMjFsNmFGUGFLUQpzQmFJempNc2Zhd1c3NDI4ckU2a250" +
	"ZkNpZVlvK0FGOFBST0l4R1pEdkdDWlE2RVZ6MDVDK2gwQ1d6bjJiSmp3CkUrNnhk" +
	"VkdPYjBpRkVicFRzUkRWRi9JQ09Oc293VzljaDQwQ2dnRUFhTFpLTGJ4c2ZUY0pj" +
	"VGw1WnQyRnVpbmsKUTc0S3J4NlhqM1h0TEJjbEJjenB1NTNtWTNQUHJMTE5zNHpl" +
	"ZXRsM3hqcGZoZUZIcUhDckVWZG1tQnk4WERxTgphRFBkTVlNSzQyUTNha21uVWYw" +
	"OU5VSFBpa2dyRWdZc0ROZFBGZW5FQXdydHV3dldWR1dXVlVNS25RYXE0ZllFCmlC" +
	"dzZSUWYxanRUc3V4Z283SnFUQ3FTYUgzaHVZbmdrK1JBQzFITHJSN3BISVpOUHBL" +
	"ZFpDdUU5Vk9IMGJsQWQKV01RamtPZVg4bEo1VVJnUWc3YU9iajJaZXpoUlhXYWhy" +
	"RnVkV3ZTWHBRWmNQQjV4cVNLSXpPWTBjYzNuNlBzbgphMTZxQWZwZE1sYkpaZ1F3" +
	"elMrZ3JsNW9hSW04MXUwT2NFZWRYZGljV2wremVqR21BZmZFYkVySEYzQ2VlUT09" +
	"Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg=="
*/
// ECC TEST KEYS
var pubprivkeyb64Ecc = "Tcksa18txiwMEocq7NXdeMwz6PPBD+nxCjb/WCtxq1+dln3M3IaOmg+YfTIbBpk+jIbZZZiT+4CoeFzaJGEWmg=="

var pubkeyb64Ecc = "Tcksa18txiwMEocq7NXdeMwz6PPBD+nxCjb/WCtxq18="

//var privkeyb64Ecc = "nZZ9zNyGjpoPmH0yGwaZPoyG2WWYk/uAqHhc2iRhFpo="
