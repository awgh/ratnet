package api

// Transport - Interface to implement in a RatNet-compatable pluggable transport module
type Transport interface {
	Listen(listen string, adminMode bool)
	Name() string
	RPC(host string, method Action, args ...interface{}) (interface{}, error)
	Stop()
	MarshalJSON() (b []byte, e error)

	ByteLimit() int64 // limit on bytes per bundle for this transport
	SetByteLimit(limit int64)
}

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
