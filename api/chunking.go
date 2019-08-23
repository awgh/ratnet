package api

import (
	"bytes"
	"encoding/binary"
	"io/ioutil"
	"log"

	"github.com/awgh/bencrypt/bc"
)

// StreamHeader manifest for a chunked transfer (database version)
type StreamHeader struct {
	StreamID    uint32 `db:"streamid"`
	NumChunks   uint32 `db:"parts"`
	ChannelName string `db:"channel"`
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

// HandleChunked - shared handler for Nodes that deals with chunks and stream headers
func HandleChunked(node Node, msg Msg) error {
	if !msg.StreamHeader {
		// save chunk
		var streamID, chunkNum uint32
		data, err := ioutil.ReadAll(msg.Content)
		if err != nil {
			return err
		}
		tmpb := bytes.NewBuffer(data[:8])
		binary.Read(tmpb, binary.LittleEndian, &streamID)
		binary.Read(tmpb, binary.LittleEndian, &chunkNum)

		log.Printf("adding chunk: %x  chunkNum: %x (%d)\n", streamID, chunkNum, chunkNum)
		return node.AddChunk(streamID, chunkNum, data[8:])
	}
	// save totalChunks by streamID
	var streamID, totalChunks uint32
	tmpb := bytes.NewBuffer(msg.Content.Bytes()[:8])
	binary.Read(tmpb, binary.LittleEndian, &streamID)
	binary.Read(tmpb, binary.LittleEndian, &totalChunks)
	channel := ""
	if msg.IsChan {
		channel = msg.Name
	}
	log.Printf("adding stream: %x  totalChunks: %x (%d)\n", streamID, totalChunks, totalChunks)
	return node.AddStream(streamID, totalChunks, channel)
}
