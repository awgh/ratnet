package api

// Router : defines an interface for a stateful Routing object
type Router interface {
	Route(node Node, msg []byte) error
	Patch(from string, to ...string)
	MarshalJSON() (b []byte, e error)
}
