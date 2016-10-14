package policy

import (
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/ratnet/api"
)

// Poll : defines a Polling Connection Policy, which will periodically connect to each remote Peer
type Poll struct {
	// internal
	wg        sync.WaitGroup
	isRunning bool

	// last poll times
	lastPollLocal, lastPollRemote int64
	transport                     api.Transport
	node                          api.Node
}

// RunPolicy : Poll
func (p *Poll) RunPolicy() error {
	if p.isRunning {
		return errors.New("Policy is already running")
	}

	p.lastPollLocal = 0
	p.lastPollRemote = 0

	go func() {
		p.wg.Add(1)
		defer p.wg.Done()
		if p.isRunning {
			return
		}
		p.isRunning = true

		pubsrv, err := p.node.ID()
		if err != nil {
			log.Fatal("Couldn't get routing key in Poll.RunPolicy:\n" + err.Error())
		}
		counter := 0
		for {
			// check if we should still be running
			if p.isRunning == false {
				break
			}
			time.Sleep(500 * time.Millisecond) // update interval

			// Get Server List
			peers, err := p.node.GetPeers()
			if err != nil {
				log.Println("Poll.RunPolicy error in loop: ", err)
				continue
			}
			for _, element := range peers {
				if element.Enabled {
					_, err := p.pollServer(p.transport, p.node, element.URI, pubsrv)
					if err != nil {
						log.Println("pollServer error: ", err.Error())
					}
				}
			}

			if counter%500 == 0 {
				p.node.FlushOutbox(300) // seconds to cache
			}
			counter++
		}
	}()

	return nil
}

// pollServer will keep trying until either we get a result or the timeout expires
func (p *Poll) pollServer(transport api.Transport, node api.Node, host string, pubsrv bc.PubKey) (bool, error) {

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

// Stop : Stops this instance of Poll from running
func (p *Poll) Stop() {
	p.isRunning = false
	p.wg.Wait()
	p.transport.Stop()
}

// NewPoll : Returns a new instance of a Poll Connection Policy
func NewPoll(transport api.Transport, node api.Node) *Poll {
	p := new(Poll)
	p.transport = transport
	p.node = node
	return p
}
