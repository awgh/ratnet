package qldb

import (
	"database/sql"
	"log"
	"os"
	"os/signal"
)

// GetChannelPrivKey : Return the private key of a given channel
func (node *Node) GetChannelPrivKey(name string) (string, error) {
	c := node.db()
	r := transactQueryRow(c, "SELECT privkey FROM channels WHERE name==$1;", name)
	var privkey string
	if err := r.Scan(&privkey); err == sql.ErrNoRows {
		return "", nil
	} else if err != nil {
		return "", err
	} else {
		return privkey, nil
	}
}

func (node *Node) refreshChannels(c *sql.DB) { // todo: this could be selective or somehow less heavy
	// refresh the channelKeys map
	rc := transactQuery(c, "SELECT name,privkey FROM channels;")
	for rc.Next() {
		var n, s string
		rc.Scan(&n, &s)
		cc := node.contentKey.Clone()
		if err := cc.FromB64(s); err == nil {
			node.channelKeys[n] = cc
		}
	}
}

func (node *Node) seenRecently(hdr []byte) bool {

	shdr := string(hdr)
	_, aok := node.recentPage1[shdr]
	_, bok := node.recentPage2[shdr]
	retval := aok || bok

	switch node.recentPageIdx {
	case 1:
		if len(node.recentPage1) >= 50 {
			if len(node.recentPage2) >= 50 {
				node.recentPage2 = nil
				node.recentPage2 = make(map[string]byte)
			}
			node.recentPageIdx = 2
			node.recentPage2[shdr] = 1
		} else {
			node.recentPage1[shdr] = 1
		}
	case 2:
		if len(node.recentPage2) >= 50 {
			if len(node.recentPage1) >= 50 {
				node.recentPage1 = nil
				node.recentPage1 = make(map[string]byte)
			}
			node.recentPageIdx = 1
			node.recentPage1[shdr] = 1
		} else {
			node.recentPage2[shdr] = 1
		}
	}
	return retval
}

/*
func (node *Node) handleErr(err error) {
	errMsg := Msg{Name: "[ERROR]"}
	errMsg.Content = bytes.NewBufferString(err.Error())
	node.Err <- errMsg
}
*/
func (node *Node) signalMonitor() {
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, nil)
	go func() {
		defer node.Stop()
		for {
			switch <-sigChannel {
			case os.Kill:
				break
			}
		}
	}()
}

func (node *Node) debugMsg(msg string) {
	if node.debugMode {
		log.Println("[DEBUG] => " + msg)
	}
}
