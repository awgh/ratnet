package api

// Router : defines an interface for a stateful Routing object
type Router interface {
	// Route : Determine what to do with the given message, and then have the node do it.
	Route(node Node, msg []byte) error
	// Patch : Add a mapping from an incoming channel to one or more destination channels
	Patch(from string, to ...string)
	// MarshalJSON : Serialize this type to JSON
	MarshalJSON() (b []byte, e error)
}
