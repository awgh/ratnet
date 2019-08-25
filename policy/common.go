package policy

import (
	"log"
	"sync"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/ratnet/api"
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
	//log.Printf("pollServer using peer %+v\n", peer)

	if peer.RoutingPub == nil {
		rpubkey, err := transport.RPC(host, "ID")
		if err != nil {
			log.Println(err.Error())
			return false, err
		}
		rpk, ok := rpubkey.(bc.PubKey)
		if !ok {
			log.Println("type assertion failed to bc.PubKey in p2p pollServer")
			return false, err
		}
		peer.RoutingPub = rpk
	}

	// Pickup Local
	toRemote, err := node.Pickup(peer.RoutingPub, peer.LastPollLocal, transport.ByteLimit())
	if err != nil {
		log.Println("local pickup error: " + err.Error())
		return false, err
	}
	//log.Println("pollServer Pickup Local result len: ", len(toRemote.Data))

	// Pickup Remote
	toLocalRaw, err := transport.RPC(host, "Pickup", pubsrv, peer.LastPollRemote)
	if err != nil {
		log.Println("remote pickup error: " + err.Error())
		return false, err
	}
	var toLocal api.Bundle
	var ok bool
	if toLocalRaw != nil {
		toLocal, ok = toLocalRaw.(api.Bundle)
		if !ok {
			log.Printf("pollServer type assertion tolocalRaw failed")
			return false, err
		}
		//log.Printf("pollServer Pickup Remote len: %d ", len(toLocal.Data))
		peer.TotalBytesRX = peer.TotalBytesRX + int64(len(toLocal.Data))
	}
	// Dropoff Remote
	if len(toRemote.Data) > 0 {
		if _, err := transport.RPC(host, "Dropoff", toRemote); err != nil {
			log.Println("remote dropoff error: " + err.Error())
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
