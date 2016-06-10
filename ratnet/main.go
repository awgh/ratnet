package main

import (
	"flag"
	"fmt"
	"log"
	"github.com/awgh/ratnet/transports"
)

// usage: ./ratnet -dbfile=ratnet2.ql -p=20003

func serve(database string, listenPublic string, listenAdmin string, certfile string, keyfile string) {

	transports.NewServer("https", listenPublic, certfile, keyfile, database, false)
	log.Println("Public Server started: ", listenPublic)

	transports.NewServer("https", listenAdmin, certfile, keyfile, database, true)
	log.Println("Control Server started: ", listenAdmin)
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

	serve(dbFile, publicString, adminString, "cert.pem", "key.pem")
}
