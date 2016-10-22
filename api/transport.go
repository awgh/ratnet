package api

// Transport - Interface to implement in a RatNet-compatable pluggable transport module
type Transport interface {
	Listen(listen string, adminMode bool)
	Name() string
	RPC(host string, method string, args ...string) ([]byte, error)
	Stop()
	MarshalJSON() (b []byte, e error)
}

// RemoteCall : defines a Remote Procedure Call
type RemoteCall struct {
	Action string
	Args   []string
}
