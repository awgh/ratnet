package fs

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/ratnet/api"
)

// ID : Return routing key
func (node *Node) ID() (bc.PubKey, error) {
	return node.routingKey.GetPubKey(), nil
}

// Dropoff : Deliver a batch of  messages to a remote node
func (node *Node) Dropoff(bundle api.Bundle) error {
	node.debugMsg("Dropoff called")
	if len(bundle.Data) < 1 { // todo: correct min length
		return errors.New("Dropoff called with no data")
	}
	tagOK, data, err := node.routingKey.DecryptMessage(bundle.Data)
	if err != nil {
		return err
	} else if !tagOK {
		return errors.New("Luggage Tag Check Failed in Dropoff")
	}

	var msgs [][]byte

	//Use default gob decoder
	reader := bytes.NewReader(data)
	dec := gob.NewDecoder(reader)
	if err := dec.Decode(&msgs); err != nil {
		log.Printf("dropoff gob decode failed, len %d\n", len(data))
		return err
	}
	for i := 0; i < len(msgs); i++ {
		if len(msgs[i]) < 16 { // aes.BlockSize == 16
			continue //todo: remove padding before here?
		}
		err = node.router.Route(node, msgs[i])
		if err != nil {
			log.Println("error in dropoff: " + err.Error())
			continue // we don't want to return routing errors back out the remote public interface
		}
	}

	node.debugMsg("Dropoff returned")
	return nil
}

// Pickup : Get messages from a remote node
func (node *Node) Pickup(rpub bc.PubKey, lastTime int64, maxBytes int64, channelNames ...string) (api.Bundle, error) {
	node.debugMsg("Pickup called")
	var retval api.Bundle
	var msgs [][]byte
	retval.Time = lastTime
	var bytesRead int64

	err := filepath.Walk(node.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Pickup failure accessing a path %q: %v\n", path, err)
			return err
		}
		fileTime := info.ModTime().UnixNano()
		//if !info.IsDir() {
		//	log.Printf("pickup t1: %d  t2: %d\n", fileTime, lastTime)
		//}
		if !info.IsDir() && fileTime > lastTime {
			b, err := ioutil.ReadFile(path) //filepath.Join(node.basePath, path))
			if err != nil {
				log.Printf("prevent panic by handling failure reading a file %q: %v\n", path, err)
				return err
			}

			if fileTime == retval.Time {
				log.Println("Identical filetimes, attempting to exceed recommended protocol buffer size")
			} else if bytesRead+int64(len(b)) >= maxBytes { // no room for next msg
				return io.EOF
			}

			msgs = append(msgs, b)
			bytesRead += int64(len(b))
			if fileTime > retval.Time {
				retval.Time = fileTime
			}
		}
		//fmt.Printf("visited file: %q\n", path)
		return nil
	})
	if err != nil && err != io.EOF {
		return retval, err
	}

	// transmit
	if len(msgs) > 0 {

		//use default gob encoder
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		if err := enc.Encode(msgs); err != nil {
			return retval, err
		}
		cipher, err := node.routingKey.EncryptMessage(buf.Bytes(), rpub)
		if err != nil {
			return retval, err
		}
		retval.Data = cipher
		return retval, err
	}
	node.debugMsg("Pickup returned")
	return retval, nil
}
