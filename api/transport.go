package api

// Transport - Interface to implement in a RatNet-compatable pluggable transport module
type Transport interface {
	Listen(listen string, adminMode bool)
	Name() string
	RPC(host string, method string, args ...interface{}) (interface{}, error)
	Stop()
	MarshalJSON() (b []byte, e error)

	ByteLimit() int64 // limit on bytes per bundle for this transport
	SetByteLimit(limit int64)
}
