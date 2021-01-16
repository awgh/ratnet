package fs

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/ratnet/api"
	"github.com/awgh/ratnet/api/events"
)

// ID : Return routing key
func (node *Node) ID() (bc.PubKey, error) {
	return node.routingKey.GetPubKey(), nil
}

// Dropoff : Deliver a batch of  messages to a remote node
func (node *Node) Dropoff(bundle api.Bundle) error {
	events.Debug(node, "Dropoff called")
	if len(bundle.Data) < 1 { // todo: correct min length
		return errors.New("Dropoff called with no data")
	}
	tagOK, data, err := node.routingKey.DecryptMessage(bundle.Data)
	if err != nil {
		return err
	} else if !tagOK {
		return errors.New("Luggage Tag Check Failed in FSNode Dropoff")
	}
	msgs, err := api.BytesBytesFromBytes(&data)
	if err != nil {
		events.Warning(node, "dropoff decode failed, len %d\n", len(data))
		return err
	}
	for i := 0; i < len(*msgs); i++ {
		if len((*msgs)[i]) < 16 { // aes.BlockSize == 16
			continue // todo: remove padding before here?
		}
		err = node.router.Route(node, (*msgs)[i])
		if err != nil {
			events.Error(node, "error in dropoff: "+err.Error())
			continue // we don't want to return routing errors back out the remote public interface
		}
	}

	events.Debug(node, "Dropoff returned")
	return nil
}

// Pickup : Get messages from a remote node
func (node *Node) Pickup(rpub bc.PubKey, lastTime int64, maxBytes int64, channelNames ...string) (api.Bundle, error) {
	events.Debug(node, "Pickup called")
	var retval api.Bundle
	var msgs [][]byte
	retval.Time = lastTime
	var bytesRead int64

	err := filepath.Walk(node.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			events.Error(node, "Pickup failure accessing a path:", path, err)
			return err
		}
		fileTime := info.ModTime().UnixNano()
		if !info.IsDir() && fileTime > lastTime {
			b, err := ioutil.ReadFile(path) // filepath.Join(node.basePath, path))
			if err != nil {
				events.Error(node, "prevent panic by handling failure reading a file:", path, err)
				return err
			}

			if fileTime == retval.Time {
				events.Warning(node, "Identical filetimes, attempting to exceed recommended protocol buffer size")
			} else if bytesRead+int64(len(b)) >= maxBytes { // no room for next msg
				events.Warning(node, "Result too big to be fetched on this transport! Flush and rechunk")
				return io.EOF
			}

			msgs = append(msgs, b)
			bytesRead += int64(len(b))
			if fileTime > retval.Time {
				retval.Time = fileTime
			}
		}
		return nil
	})
	if err != nil && err != io.EOF {
		return retval, err
	}

	// transmit
	if len(msgs) > 0 {
		buf := api.BytesBytesToBytes(&msgs)
		cipher, err := node.routingKey.EncryptMessage(*buf, rpub)
		if err != nil {
			return retval, err
		}
		retval.Data = cipher
		return retval, err
	}
	events.Debug(node, "Pickup returned")
	return retval, nil
}
