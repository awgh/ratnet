package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/awgh/bencrypt/ecc"
	"github.com/awgh/ratnet/api"
	"github.com/awgh/ratnet/nodes/qldb"
	"github.com/awgh/ratnet/policy"
	"github.com/awgh/ratnet/transports/https"
)

// usage: ./ratnet -dbfile=ratnet2.ql -p=20003

func serve(transportPublic api.Transport, transportAdmin api.Transport, node api.Node, listenPublic string, listenAdmin string) {

	node.SetPolicy(
		policy.NewServer(transportPublic, listenPublic, false),
		policy.NewServer(transportAdmin, listenAdmin, true))

	log.Println("Public Server starting: ", listenPublic)
	log.Println("Control Server starting: ", listenAdmin)

	node.Start()
}

func p2p(transportPublic api.Transport, transportAdmin api.Transport, node api.Node, listenPublic string, listenAdmin string) {
	node.SetPolicy(
		policy.NewP2P(transportPublic, listenPublic, node, false, 500, 500),
		policy.NewServer(transportAdmin, listenAdmin, true))

	log.Println("Public Server starting: ", listenPublic)
	log.Println("Control Server starting: ", listenAdmin)

	node.Start()
}

func main() {

	var dbFile string
	var publicPort, adminPort int

	flag.StringVar(&dbFile, "dbfile", "ratnet.ql", "QL Database File")
	flag.IntVar(&publicPort, "p", 20001, "HTTPS Public Port (*)")
	flag.IntVar(&adminPort, "ap", 20002, "HTTPS Admin Port (localhost)")
	flag.Parse()

	publicString := fmt.Sprintf(":%d", publicPort)
	adminString := fmt.Sprintf("localhost:%d", adminPort)

	// QLDB Node Mode
	node := qldb.New(new(ecc.KeyPair), new(ecc.KeyPair))
	node.BootstrapDB(dbFile)

	// RamNode Mode:
	//node := ram.New(new(ecc.KeyPair), new(ecc.KeyPair))

	serve(https.New("cert.pem", "key.pem", node, true), https.New("cert.pem", "key.pem", node, true), node, publicString, adminString)
}
