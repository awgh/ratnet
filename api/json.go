// +build !no_json

package api

// JSON - includes the JSON serializer
type JSON interface {
	// MarshalJSON : Serialize this type to JSON
	MarshalJSON() (b []byte, e error)
}
