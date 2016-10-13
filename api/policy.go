package api

// Policy : defines a "Connection Policy" object
type Policy interface {
	RunPolicy() error
	Stop()
}

//transport transports.Transport, node Node, args ...string
