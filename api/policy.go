package api

// Policy : defines a "Connection Policy" object
type Policy interface {
	RunPolicy() error
	Stop()
	MarshalJSON() (b []byte, e error)
}
