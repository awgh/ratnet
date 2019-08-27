package fs

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/awgh/ratnet/api"
	"github.com/awgh/ratnet/api/events"
)

// GetChannelPrivKey : Return the private key of a given channel
func (node *Node) GetChannelPrivKey(name string) (string, error) {
	c, ok := node.channels[name]
	if !ok {
		return "", errors.New("Channel not found")
	}
	return c.Privkey.ToB64(), nil
}

// Forward - Add an already-encrypted message to the outbound message queue (forward it along)
func (node *Node) Forward(msg api.Msg) error {

	flags := uint8(0)
	if msg.IsChan {
		flags |= api.ChannelFlag
	}
	if msg.Chunked {
		flags |= api.ChunkedFlag
	}
	if msg.StreamHeader {
		flags |= api.StreamHeaderFlag
	}
	rxsum := []byte{flags} // prepend flags byte
	m := new(outboxMsg)
	path := node.basePath
	if msg.IsChan {
		// prepend a uint16 of channel name length, little-endian
		t := uint16(len(msg.Name))
		rxsum = append(rxsum, byte(t>>8), byte(t&0xFF))
		rxsum = append(rxsum, []byte(msg.Name)...)
		m.channel = msg.Name
		// create channel dir if not exist
		path = filepath.Join(path, msg.Name)
		os.Mkdir(path, os.FileMode(int(0700)))
	}
	message := append(rxsum, msg.Content.Bytes()...)
	/*
		for _, mail := range node.outbox {
			if mail.channel == channelName && bytes.Equal(mail.msg, message) {
				return nil // already have a copy... //todo: do we really need this check?
			}
		}
	*/

	f, err := os.Create(filepath.Join(path, hex(node.outboxIndex)))
	if err != nil {
		return err
	}
	node.outboxIndex++
	defer f.Close()
	w := bufio.NewWriter(f)
	w.Write(message)
	w.Flush()

	return nil
}

// Handle - Decrypt and handle an encrypted message
func (node *Node) Handle(msg api.Msg) (bool, error) {
	var clear []byte
	var err error
	var tagOK bool
	var clearMsg api.Msg // msg to out channel

	if msg.IsChan {
		v, ok := node.channels[msg.Name]
		if !ok || v.Privkey == nil {
			return tagOK, errors.New("Cannot Handle message for Unknown Channel")
		}
		clearMsg = api.Msg{Name: msg.Name, IsChan: true, Chunked: msg.Chunked, StreamHeader: msg.StreamHeader}
		tagOK, clear, err = v.Privkey.DecryptMessage(msg.Content.Bytes())
	} else {
		clearMsg = api.Msg{Name: "[content]", IsChan: false, Chunked: msg.Chunked, StreamHeader: msg.StreamHeader}
		tagOK, clear, err = node.contentKey.DecryptMessage(msg.Content.Bytes())
	}
	// DecryptMessage will return !tagOK if the quick-check fails, which is common
	if !tagOK || err != nil {
		return tagOK, err
	}
	clearMsg.Content = bytes.NewBuffer(clear)

	if msg.Chunked {
		err = api.HandleChunked(node, clearMsg)
		if err != nil {
			return false, err
		}
		return true, err
	}

	select {
	case node.Out() <- clearMsg:
		events.Debug(node, "Sent message "+fmt.Sprint(msg.Content.Bytes()))
	default:
		events.Debug(node, "No message sent")
	}
	return tagOK, nil
}

func (node *Node) signalMonitor() {
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, nil)
	go func() {
		defer node.Stop()
		for {
			switch <-sigChannel {
			case os.Kill:
				return
			}
		}
	}()
}

// AddStream - adds a partial message header to internal storage
func (node *Node) AddStream(streamID uint32, totalChunks uint32, channelName string) error {
	stream := new(api.StreamHeader)
	stream.StreamID = streamID
	stream.NumChunks = totalChunks
	stream.ChannelName = channelName
	node.streams[streamID] = stream
	return nil
}

// AddChunk - adds a chunk of a partial message to internal storage
func (node *Node) AddChunk(streamID uint32, chunkNum uint32, data []byte) error {
	chunk := new(api.Chunk)
	chunk.StreamID = streamID
	chunk.ChunkNum = chunkNum
	chunk.Data = data
	if node.chunks[streamID] == nil {
		node.chunks[streamID] = make(map[uint32]*api.Chunk)
	}
	node.chunks[streamID][chunkNum] = chunk
	return nil
}
