// +build !no_json

package api

// JSON - includes the JSON serializer
type JSON interface {
	// MarshalJSON : Serialize this type to JSON
	MarshalJSON() (b []byte, e error)
}

// ImportExport - makes a node exportable/importable when activated by build tag
type ImportExport interface {
	// Import node from JSON
	Import(jsonConfig []byte) error
	// Export node to JSON
	Export() ([]byte, error)
}

// ExportedNode - Node Config structure for export
type ExportedNode struct {
	ContentKey  string
	ContentType string
	RoutingKey  string
	RoutingType string
	Policies    []Policy

	Profiles []ProfilePrivB64
	Channels []ChannelPrivB64
	Peers    []Peer
	Contacts []Contact
	Router   Router
}

// ImportedNode - Node Config structure for import
type ImportedNode struct {
	ContentKey  string
	ContentType string
	RoutingKey  string
	RoutingType string
	Policies    []map[string]interface{}

	Profiles []ProfilePrivB64
	Channels []ChannelPrivB64
	Peers    []Peer
	Contacts []Contact
	Router   map[string]interface{}
}
