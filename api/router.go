package api

// Router : defines an interface for a stateful Routing object
type Router interface {
	// Route - TBD
	Route(node api.Node, msg *[]byte)
}

// DefaultRouter - The Default router makes no changes at all,
//                 every message is sent out on the same channel it came in on,
//                 and non-channel messages are consumed but not forwarded
type DefaultRouter struct{}

// Route - Router that does default behavior
func (*DefaultRouter) Route(node api.Node, msg *[]byte) {
	/*
			checkMessageForMe := true

		var channelLen uint16 // beginning uint16 of message is channel name length
		channelLen = (uint16(msg[0]) << 8) | uint16(msg[1])

		if len(msg) < int(channelLen)+2+16+16 { // uint16 + nonce + hash //todo
			log.Println("Incorrect channel name length")
			continue
		}
		var crypt bc.KeyPair
		channelName := ""
		if channelLen == 0 { // private message (zero length channel)
			crypt = node.contentKey
		} else { // channel message
			channelName = string(msg[2 : 2+channelLen])
			var ok bool
			crypt, ok = node.channelKeys[channelName]
			if !ok { // we are not listening to this channel
				checkMessageForMe = false
			}
		}
		msg = msg[2+channelLen:] //skip over the channel name
		forward := true

		if node.seenRecently(msg[:16]) { // LOOP PREVENTION before handling or forwarding
			forward = false
			checkMessageForMe = false
		}
		if checkMessageForMe { // check to see if this is a msg for me
			pubkey := crypt.GetPubKey()
			hash, err := bc.DestHash(pubkey, msg[:16])
			if err != nil {
				continue
			}
			if bytes.Equal(hash, msg[16:16+len(hash)]) {
				if channelLen == 0 {
					forward = false
				}
				clear, err := crypt.DecryptMessage(msg[16+len(hash):])
				if err != nil {
					continue
				}
				var clearMsg api.Msg // write msg to out channel
				if channelLen == 0 {
					clearMsg = api.Msg{Name: "[content]", IsChan: false}
				} else {
					clearMsg = api.Msg{Name: channelName, IsChan: true}
				}
				clearMsg.Content = bytes.NewBuffer(clear)

				select {
				case node.Out() <- clearMsg:
					node.debugMsg("Sent message " + fmt.Sprint(msg))
				default:
					node.debugMsg("No message sent")
				}
			}
		}
		if forward {
			// save message in my outbox, if not already present
			r1 := transactQueryRow(c, "SELECT channel FROM outbox WHERE channel==$1 AND msg==$2;", channelName, lines[i])
			var rc string
			err := r1.Scan(&rc)
			if err == sql.ErrNoRows {
				// we don't have this yet, so add it
				t := time.Now().UnixNano()
				transactExec(c, "INSERT INTO outbox(channel,msg,timestamp) VALUES($1,$2,$3);",
					channelName, lines[i], t)
			} else if err != nil {
				return err
			}
		}

*/
}
