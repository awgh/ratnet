package api

import (
	"bytes"

	"github.com/awgh/bencrypt/bc"
)

// Msg : object that describes the messages passed between nodes
type Msg struct {
	Name         string
	Content      *bytes.Buffer
	IsChan       bool
	PubKey       bc.PubKey
	Chunked      bool
	StreamHeader bool
}
