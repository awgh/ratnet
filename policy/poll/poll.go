package poll

import (
	"crypto/rand"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/awgh/ratnet/api"
	"github.com/awgh/ratnet/api/events"
	"github.com/awgh/ratnet/policy"
)

// Poll : defines a Polling Connection Policy, which will periodically connect to each remote Peer
type Poll struct {
	// internal
	wg        sync.WaitGroup
	isRunning uint32

	// last poll times
	lastPollLocal, lastPollRemote int64

	Transport api.Transport
	node      api.Node
	pt        *policy.PeerTable

	interval int32
	jitter   int32

	Groups        []string
	curGroupIndex int

	RetryForever  bool
	RetryAttempts int
}

// New : Returns a new instance of a Poll Connection Policy
func New(transport api.Transport, node api.Node, interval, jitter int, group ...string) *Poll {
	p := new(Poll)
	if len(group) > 0 {
		p.Groups = group
	} else {
		p.Groups = []string{""} // if we don't have a specified group, it's ""
	}
	p.Transport = transport
	p.node = node
	p.interval = int32(interval)
	p.jitter = int32(jitter)

	p.RetryForever = true
	p.RetryAttempts = 3
	p.curGroupIndex = 0
	p.pt = policy.NewPeerTable()

	return p
}

// GetInterval : Get the interval at which this policy sends/receives messages
func (p *Poll) GetInterval() int {
	return int(atomic.LoadInt32(&p.interval))
}

// SetInterval : Set the interval at which this policy sends/receives messages
func (p *Poll) SetInterval(newInterval int) {
	atomic.StoreInt32(&p.interval, int32(newInterval))
}

// GetJitter : Get the percentage of which the interval with be randomly skewed
func (p *Poll) GetJitter() int {
	return int(atomic.LoadInt32(&p.jitter))
}

// SetJitter : Set the percentage of which the interval with be randomly skewed
func (p *Poll) SetJitter(newJitter int) {
	atomic.StoreInt32(&p.jitter, int32(newJitter))
}

// RunPolicy : Poll
func (p *Poll) RunPolicy() error {
	if p.IsRunning() {
		return errors.New("Policy is already running")
	}

	p.lastPollLocal = 0
	p.lastPollRemote = 0

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		if p.IsRunning() {
			return
		}
		p.setIsRunning(true)

		pubsrv, err := p.node.ID()
		if err != nil {
			events.Critical(p.node, "Couldn't get routing key in Poll.RunPolicy:\n"+err.Error())
		}

		fails := make(map[string]int)
		b := make([]byte, 1)
		counter := 0
		for {
			// check if we should still be running
			if !p.IsRunning() {
				break
			}

			if interval := p.GetInterval(); interval > 0 {
				delay := time.Duration(interval) * time.Millisecond
				rand.Read(b)
				var sleep time.Duration
				jit := p.GetJitter()
				if jit == 0 { // no jitter discount, no divide by zero
					sleep = delay
				} else { // discount a jitter amount within the given percentage
					sleep = (time.Duration((float64(100-(int(b[0])%jit)) / 100) * float64(delay)))
				}
				time.Sleep(sleep) // update interval
			}

			// Get Server List for this Poll's assigned Group
			peers, err := p.node.GetPeers(p.Groups[p.curGroupIndex])
			if err != nil {
				events.Warning(p.node, "Poll.RunPolicy error in loop: ", err)
				continue
			}
			tries := 0
			for _, element := range peers {
				if _, ok := fails[element.URI]; !ok {
					fails[element.URI] = 0
				}
				if element.Enabled && fails[element.URI] < p.RetryAttempts {
					tries++

					_, err := p.pt.PollServer(p.Transport, p.node, element.URI, pubsrv)
					if err != nil {
						events.Warning(p.node, "pollServer error: ", err.Error())
						fails[element.URI]++
					} else {
						fails[element.URI] = 0
					}
				}
			}
			if tries == 0 {
				if p.curGroupIndex < len(p.Groups)-1 {
					fails = make(map[string]int)
					p.curGroupIndex++
				} else if p.RetryForever {
					fails = make(map[string]int)
					p.curGroupIndex = 0
				} else {
					events.Warning(p.node, "pollServer error: All Peers have been disabled or hit retry limits")
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

// Stop : Stops this instance of Poll from running
func (p *Poll) Stop() {
	p.setIsRunning(false)
	p.wg.Wait()
	p.Transport.Stop()
}

// GetTransport : Returns the transports associated with this policy
//
func (p *Poll) GetTransport() api.Transport {
	return p.Transport
}

// IsRunning - returns true if this policy is running
func (p *Poll) IsRunning() bool {
	return atomic.LoadUint32(&p.isRunning) == 1
}

func (p *Poll) setIsRunning(b bool) {
	var running uint32 = 0
	if b {
		running = 1
	}
	atomic.StoreUint32(&p.isRunning, running)
}
