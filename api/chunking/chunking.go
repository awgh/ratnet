package chunking

import (
	"bytes"
	"encoding/binary"
	"io/ioutil"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/ratnet/api"
	"github.com/awgh/ratnet/api/events"
)

// ChunkSize - calculates the minimum chunk size from all active transports
func ChunkSize(node api.Node) uint32 {
	var chunksize uint32 = 64 * 1024
	policies := node.GetPolicies()
	for _, p := range policies {
		limit := uint32(p.GetTransport().ByteLimit())
		if limit < chunksize {
			chunksize = limit
		}
	}
	if chunksize <= 132 {
		events.Critical(node, "Transport has invalid low byte limit")
	}

	return chunksize - 132 //todo: this is the overhead for ECC, what about RSA?
}

// SendChunked - utility function to break large messages into smaller ones for transports that can't handle arbitrarily large messages
func SendChunked(node api.Node, chunkSize uint32, msg api.Msg) (err error) {

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
		if err = node.SendMsg(api.Msg{Name: msg.Name, Content: b, IsChan: msg.IsChan, PubKey: msg.PubKey, Chunked: true, StreamHeader: true}); err != nil {
			return
		}
		for i := uint32(0); i < wholeLoops; i++ {
			b := bytes.NewBuffer(streamID)                  // StreamID
			binary.Write(b, binary.LittleEndian, uint32(i)) // ChunkNum
			b.Write(buf[i*chunkSizeMinusHeader : (i*chunkSizeMinusHeader)+chunkSizeMinusHeader])
			if err = node.SendMsg(api.Msg{Name: msg.Name, Content: b, IsChan: msg.IsChan, PubKey: msg.PubKey, Chunked: true}); err != nil {
				return
			}
		}
		if remainder > 0 {
			b := bytes.NewBuffer(streamID)                           // StreamID
			binary.Write(b, binary.LittleEndian, uint32(wholeLoops)) // ChunkNum
			b.Write(buf[wholeLoops*chunkSizeMinusHeader:])
			if err = node.SendMsg(api.Msg{Name: msg.Name, Content: b, IsChan: msg.IsChan, PubKey: msg.PubKey, Chunked: true}); err != nil {
				return
			}
		}
	}
	return
}

// HandleChunked - shared handler for Nodes that deals with chunks and stream headers
func HandleChunked(node api.Node, msg api.Msg) error {
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

		events.Debug(node, "adding chunk: %x  chunkNum: %x (%d)\n", streamID, chunkNum, chunkNum)
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
	events.Debug(node, "adding stream: %x  totalChunks: %x (%d)\n", streamID, totalChunks, totalChunks)
	return node.AddStream(streamID, totalChunks, channel)
}
