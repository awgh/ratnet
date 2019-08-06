package ram

import (
	"bytes"
	"log"
	"sync"
	"time"
)

// Message Queue / Outbox

type outboxMsg struct {
	channel   string
	msg       []byte
	timeStamp int64
}

type outboxQueue struct {
	mux    sync.Mutex
	outbox []*outboxMsg
}

// Append : Adds an outbound message to queue
func (o *outboxQueue) Append(msg *outboxMsg) {
	o.mux.Lock()
	o.outbox = append(o.outbox, msg)
	o.mux.Unlock()
}

// MsgExists : Returns true iff a matching message is already in the outbound queue
func (o *outboxQueue) MsgExists(channelName string, message []byte) bool {
	o.mux.Lock()
	for _, mail := range o.outbox {
		if mail.channel == channelName && bytes.Equal(mail.msg, message) {
			return true // already have a copy... //todo: do we really need this check? or can it be more efficient?
		}
	}
	o.mux.Unlock()
	return false
}

// Flush : Deletes outbound messages older than maxAgeSeconds seconds
func (o *outboxQueue) Flush(maxAgeSeconds int64) {
	c := (time.Now().UnixNano()) - (maxAgeSeconds * 1000000000)
	o.mux.Lock()
	for index, mail := range o.outbox {
		if mail.timeStamp < c {
			if len(o.outbox) > index {
				o.outbox = append(o.outbox[:index], o.outbox[index+1:]...)
			} else {
				o.outbox = []*outboxMsg{}
			}
		}
	}
	o.mux.Unlock()
}

// MsgsSince : Get messages after the given timestamp
func (o *outboxQueue) MsgsSince(lastTime int64, maxBytes int64, channelNames ...string) ([][]byte, int64) {
	var msgs [][]byte
	retvalTime := lastTime
	o.mux.Lock()
	for _, mail := range o.outbox {
		if lastTime < mail.timeStamp {
			pickupMsg := false
			if len(channelNames) > 0 {
				for _, channelName := range channelNames {
					if channelName == mail.channel {
						pickupMsg = true
					}
				}
			} else {
				pickupMsg = true
			}
			if pickupMsg {
				msgsSize := 0
				for i := range msgs {
					msgsSize += len(msgs[i])
				}
				proposedSize := len(mail.msg) + msgsSize

				if maxBytes > 0 && int64(proposedSize) > maxBytes {
					if msgsSize == 0 {
						log.Fatal("Bailing with zero return results!", proposedSize, len(mail.msg), msgsSize, maxBytes)
					}
					break
				}
				retvalTime = mail.timeStamp
				msgs = append(msgs, mail.msg)
			}
		}
	}
	o.mux.Unlock()
	return msgs, retvalTime
}
