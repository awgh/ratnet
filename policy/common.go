package policy

import (
	"sync"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/ratnet/api"
	"github.com/awgh/ratnet/api/events"
)

var peerTable map[string]*api.PeerInfo
var lock = sync.RWMutex{}

func readPeerTable(key string) (*api.PeerInfo, bool) {
	lock.RLock()
	defer lock.RUnlock()
	v, ok := peerTable[key]
	return v, ok
}
func writePeerTable(key string, val *api.PeerInfo) {
	lock.Lock()
	defer lock.Unlock()
	peerTable[key] = val
}

func init() {
	peerTable = make(map[string]*api.PeerInfo)
}

// PollServer does a Push/Pull between a local and remote Node
func PollServer(transport api.Transport, node api.Node, host string, pubsrv bc.PubKey) (bool, error) {
	// make PeerInfo for this host if doesn't exist
	if _, ok := readPeerTable(host); !ok {
		writePeerTable(host, new(api.PeerInfo))
	}
	peer, _ := readPeerTable(host)

	if peer.RoutingPub == nil {
		rpubkey, err := transport.RPC(host, "ID")
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
	toRemote, err := node.Pickup(peer.RoutingPub, peer.LastPollLocal, transport.ByteLimit())
	if err != nil {
		events.Error(node, "local pickup error: "+err.Error())
		return false, err
	}
	events.Debug(node, "pollServer Pickup Local result len: ", len(toRemote.Data))

	// Pickup Remote
	toLocalRaw, err := transport.RPC(host, "Pickup", pubsrv, peer.LastPollRemote)
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
		if _, err := transport.RPC(host, "Dropoff", toRemote); err != nil {
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
