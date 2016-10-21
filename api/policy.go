package api

// Policy : defines a "Connection Policy" object
type Policy interface {
	RunPolicy() error
	Stop()
	MarshalJSON() (b []byte, e error)
}

//transport transports.Transport, node Node, args ...string
