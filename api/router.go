package api

// Router : defines an interface for a stateful Routing object
type Router interface {
	// Route : Determine what to do with the given message, and then have the node do it.
	Route(node Node, msg []byte) error
	// Patch : Add a mapping from an incoming channel to one or more destination channels
	Patch(patch Patch)
	// GetPatches : Returns an array with the mappings of incoming channels to destination channels
	GetPatches() []Patch

	JSON
}

// Patch : defines a mapping from an incoming channel to one or more destination channels.
type Patch struct {
	From string
	To   []string
}

const (
	// StreamHeaderFlag : this message is a stream header
	StreamHeaderFlag = 0x01
	// ChunkedFlag : this message is a chunked
	ChunkedFlag = 0x02
	// ChannelFlag : this message has a channel name prefix
	ChannelFlag = 0x04
)
