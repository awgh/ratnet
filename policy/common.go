package policy

import (
	"sync"
	"sync/atomic"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/ratnet/api"
	"github.com/awgh/ratnet/api/events"
)

type PeerTable struct {
	peerTable map[string]*api.PeerInfo
	lock      sync.RWMutex
}

func NewPeerTable() *PeerTable {
	pt := new(PeerTable)
	pt.peerTable = make(map[string]*api.PeerInfo)
	return pt
}

func (pt *PeerTable) readPeerTable(key string) (*api.PeerInfo, bool) {
	v, ok := pt.peerTable[key]
	return v, ok
}

func (pt *PeerTable) writePeerTable(key string, val *api.PeerInfo) {
	pt.peerTable[key] = val
}

// PollServer does a Push/Pull between a local and remote Node
func (pt *PeerTable) PollServer(transport api.Transport, node api.Node, host string, pubsrv bc.PubKey) (bool, error) {
	pt.lock.Lock() // PollServer should be non-reentrant
	defer pt.lock.Unlock()

	// make PeerInfo for this host if doesn't exist
	if _, ok := pt.readPeerTable(host); !ok {
		pt.writePeerTable(host, new(api.PeerInfo))
	}
	peer, _ := pt.readPeerTable(host)

	if peer.RoutingPub == nil {
		rpubkey, err := transport.RPC(host, api.ID)
		if err != nil {
			events.Error(node, err.Error())
			return false, err
		}
		rpk, ok := rpubkey.(bc.PubKey)
		if !ok {
			events.Error(node, "type assertion failed to bc.PubKey in p2p pollServer")
			return false, err
		}
		peer.RoutingPub = rpk
	}

	// Pickup Local
	toRemote, err := node.Pickup(peer.RoutingPub, atomic.LoadInt64(&peer.LastPollLocal), transport.ByteLimit())
	if err != nil {
		events.Error(node, "local pickup error: "+err.Error())
		return false, err
	}
	events.Debug(node, "pollServer Pickup Local result len: ", len(toRemote.Data))

	// Pickup Remote
	toLocalRaw, err := transport.RPC(host, api.Pickup, pubsrv, peer.LastPollRemote)
	if err != nil {
		events.Error(node, "remote pickup error: "+err.Error())
		return false, err
	}
	var toLocal api.Bundle
	var ok bool
	if toLocalRaw != nil {
		toLocal, ok = toLocalRaw.(api.Bundle)
		if !ok {
			events.Error(node, "pollServer type assertion tolocalRaw failed")
			return false, err
		}
		events.Debug(node, "pollServer Pickup Remote len: %d ", len(toLocal.Data))

		peer.TotalBytesRX = peer.TotalBytesRX + int64(len(toLocal.Data))
	}
	// Dropoff Remote
	if len(toRemote.Data) > 0 {
		if _, err := transport.RPC(host, api.Dropoff, toRemote); err != nil {
			events.Error(node, "remote dropoff error: "+err.Error())
			return false, err
		}
		// only start tracking time once we start receiving data
		if peer.TotalBytesTX > 0 {
			peer.LastPollLocal = toRemote.Time
		}
		peer.TotalBytesTX = peer.TotalBytesTX + int64(len(toRemote.Data))
	}
	// Dropoff Local
	if toLocalRaw != nil && len(toLocal.Data) > 0 {
		if err := node.Dropoff(toLocal); err != nil {
			return false, err
		}
		if peer.TotalBytesRX > 0 {
			peer.LastPollRemote = toLocal.Time
		}
	}
	return true, nil
}
