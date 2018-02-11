package policy

import (
	"log"
	"runtime/debug"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/ratnet/api"
)

var peerTable map[string]*api.PeerInfo

func init() {
	peerTable = make(map[string]*api.PeerInfo)
}

// PollServer does a Push/Pull between a local and remote Node
func PollServer(transport api.Transport, node api.Node, host string, pubsrv bc.PubKey) (bool, error) {
	// make PeerInfo for this host if doesn't exist
	if _, ok := peerTable[host]; !ok {
		peerTable[host] = new(api.PeerInfo)
	}
	peer := peerTable[host]
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
	toRemote, err := node.Pickup(peer.RoutingPub, peer.LastPollLocal)
	if err != nil {
		debug.PrintStack()
		return false, err
	}
	//log.Println("pollServer Pickup Local result len: ", len(toRemote.Data))

	// Pickup Remote
	toLocalRaw, err := transport.RPC(host, "Pickup", pubsrv, peer.LastPollRemote)
	if err != nil {
		return false, err
	}
	toLocal, ok := toLocalRaw.(api.Bundle)
	if !ok {
		log.Printf("pollServer type assertion tolocalRaw failed")
		return false, err
	}
	//log.Printf("pollServer Pickup Remote len: %d ", len(toLocal.Data))

	peer.TotalBytesRX = peer.TotalBytesRX + int64(len(toLocal.Data))

	// only start tracking time once we start receiving data
	if peer.TotalBytesTX > 0 {
		peer.LastPollLocal = toRemote.Time
	}
	if peer.TotalBytesRX > 0 {
		peer.LastPollRemote = toLocal.Time
	}

	// Dropoff Remote
	if len(toRemote.Data) > 0 {
		if _, err := transport.RPC(host, "Dropoff", toRemote); err != nil {
			return false, err
		}
		peer.TotalBytesTX = peer.TotalBytesTX + int64(len(toRemote.Data))
	}
	// Dropoff Local
	if len(toLocal.Data) > 0 {
		if err := node.Dropoff(toLocal); err != nil {
			return false, err
		}
	}
	return true, nil
}
