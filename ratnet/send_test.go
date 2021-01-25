package main

import (
	"log"
	"sync"
	"testing"
	"time"

	"github.com/awgh/bencrypt/ecc"
	"github.com/awgh/ratnet/api"
	"github.com/awgh/ratnet/api/events/defaultlogger"
	"github.com/awgh/ratnet/nodes/ram"
	"github.com/awgh/ratnet/policy/poll"
	"github.com/awgh/ratnet/policy/server"
	"github.com/awgh/ratnet/transports/udp"
)

const URI = "127.0.0.1:20005"

func Test_fast_sending(t *testing.T) {
	receivingNode := ram.New(new(ecc.KeyPair), new(ecc.KeyPair))
	receivingNode.SetPolicy(server.New(udp.New(receivingNode), URI, false))
	defaultlogger.StartDefaultLogger(receivingNode, api.Info)

	sendingNode := ram.New(new(ecc.KeyPair), new(ecc.KeyPair))
	sendingNode.SetPolicy(poll.New(udp.New(sendingNode), sendingNode, 100, 0))
	sendingNode.AddPeer("rc", true, URI)
	key, _ := receivingNode.CID()
	sendingNode.AddContact("rc", key.ToB64())
	defaultlogger.StartDefaultLogger(sendingNode, api.Info)

	var err error
	if err = receivingNode.Start(); err != nil {
		t.Fatalf("receiving node failed to start: %v", err)
	}
	if err = sendingNode.Start(); err != nil {
		t.Fatalf("sending node failed to start: %v", err)
	}
	var mutex sync.Mutex

	var msgCounter int
	go func() {
		for {
			msg := <-receivingNode.Out()
			log.Printf("received message: %s", msg.Content.String())
			mutex.Lock()
			msgCounter++
			mutex.Unlock()
		}
	}()

	for i := 0; i < 3; i++ {
		if err := sendingNode.Send("rc", []byte("i hope this works...")); err != nil {
			t.Fatalf("error sending message to receiver: %v", err)
		}
		log.Println("sent message to receiver")
	}

	time.Sleep(3 * time.Second)
	// sendingNode.Stop()
	// receivingNode.Stop()

	mutex.Lock()
	defer mutex.Unlock()
	if msgCounter != 3 {
		t.Fatalf("expected receiving 3 messages, got %d", msgCounter)
	}
}
