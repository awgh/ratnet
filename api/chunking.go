package api

import (
	"bytes"
	"encoding/binary"

	"github.com/awgh/bencrypt/bc"
)

const (
//hdrMagic   uint32 = 0xF113
//chunkMagic uint32 = 0xF114
)

// StreamHeader manifest for a chunked transfer (database version)
type StreamHeader struct {
	StreamID    uint32 `db:"streamid"`
	NumChunks   uint32 `db:"parts"`
	ChannelName string `db:"channel"`
	Pubkey      string `db:"pubkey"`
}

// Chunk header for each chunk
type Chunk struct {
	StreamID uint32 `db:"streamid"`
	ChunkNum uint32 `db:"chunknum"`
	Data     []byte `db:"data"`
}

// ChunkSize - calculates the minimum chunk size from all active transports
func ChunkSize(node Node) uint32 {
	var chunksize uint32 = 64 * 1024
	policies := node.GetPolicies()
	for _, p := range policies {
		limit := uint32(p.GetTransport().ByteLimit())
		if limit < chunksize {
			chunksize = limit
		}
	}
	return chunksize
}

// SendChunked - utility function to break large messages into smaller ones for transports that can't handle arbitrarily large messages
func SendChunked(node Node, chunkSize uint32, msg Msg) (err error) {

	buf := msg.Content.Bytes()
	buflen := uint32(len(buf))
	chunkSizeMinusHeader := chunkSize - 8 // chunk header is two uint32's -> 8 bytes

	wholeLoops := buflen / chunkSizeMinusHeader
	remainder := buflen % chunkSizeMinusHeader
	totalChunks := wholeLoops
	if remainder != 0 {
		totalChunks++
	}

	var streamID []byte
	if wholeLoops+remainder != 0 { // we're sending something, send stream header
		streamID, err = bc.GenerateRandomBytes(4)
		if err != nil {
			return
		}
		b := bytes.NewBuffer(streamID)                            // StreamID
		binary.Write(b, binary.LittleEndian, uint32(totalChunks)) // NumChunks
		if err = node.SendMsg(Msg{Name: msg.Name, Content: b, IsChan: msg.IsChan, PubKey: msg.PubKey, Chunked: true, StreamHeader: true}); err != nil {
			return
		}
		for i := uint32(0); i < wholeLoops; i++ {
			b := bytes.NewBuffer(streamID)                  // StreamID
			binary.Write(b, binary.LittleEndian, uint32(i)) // ChunkNum
			b.Write(buf[i*chunkSizeMinusHeader : (i*chunkSizeMinusHeader)+chunkSizeMinusHeader])
			//log.Println("chunk loop", i, buflen, len(tbuf))
			if err = node.SendMsg(Msg{Name: msg.Name, Content: b, IsChan: msg.IsChan, PubKey: msg.PubKey, Chunked: true}); err != nil {
				return
			}
		}
		if remainder > 0 {
			b := bytes.NewBuffer(streamID)                           // StreamID
			binary.Write(b, binary.LittleEndian, uint32(wholeLoops)) // ChunkNum
			b.Write(buf[wholeLoops*chunkSizeMinusHeader:])
			//log.Println("chunk remainder", len(buf[wholeLoops*chunkSize:]))
			if err = node.SendMsg(Msg{Name: msg.Name, Content: b, IsChan: msg.IsChan, PubKey: msg.PubKey, Chunked: true}); err != nil {
				return
			}
		}
	}
	return
}

/*
// ReadChunked - utility function to rebuild chunks into the original buffer
// 					returns: ended (bool) - true when last chunk has been read
//							 Msg - set to the reconstructed message, only set when ended is true
func ReadChunked(chunks *map[int][]byte, rxMsgBuffer *api.Msg) (bool, *Msg) {
	rxMsg := new(Msg)
	var chunkId uint32
	binary.Read(rxMsgBuffer.Content, binary.LittleEndian, &chunkId)
	log.Printf("chunkId read at server: %x  len(rx): %d\n", chunkId, rxMsgBuffer.Content.Len())

	if chunkId != 0xFFFFFFFF {
		(*chunks)[int(chunkId)] = rxMsgBuffer.Content.Bytes()
		log.Println("chunkId cached", chunkId)
		return false, nil
	}
	// re-assemble johnny 5
	var buf bytes.Buffer
	var keys []int
	for k, _ := range *chunks {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		log.Println("reassembled chunk", k)
		buf.Write((*chunks)[k])
	}
	log.Println("reassembled remainder")
	buf.Write(rxMsgBuffer.Content.Bytes())

	//Use default gob decoder
	dec := gob.NewDecoder(bytes.NewReader(buf.Bytes()))
	if err := dec.Decode(&rxMsg); err != nil {
		log.Println("[INCOMING MSG ERR c2 224]:", err.Error(), rxMsg, rxMsgBuffer)
	}
	//chunks = new(map[int][]byte)
	return true, rxMsg
}
*/
