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

// RemoteCall : defines a Remote Procedure Call
type RemoteCall struct {
	Action string
	Args   []interface{}
}

// RemoteResponse : defines a response returned from a Remote Procedure Call
type RemoteResponse struct {
	Error string
	Value interface{}
}

// IsNil - is this response Nil?
func (r *RemoteResponse) IsNil() bool { return r.Value == nil }

// IsErr - is this response an error?
func (r *RemoteResponse) IsErr() bool { return r.Error != "" }
