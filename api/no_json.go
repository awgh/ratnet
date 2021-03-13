// +build no_json

package api

// JSON - does not include the JSON serializer
type JSON interface{} // disabled

// ImportExport - makes a node exportable/importable when activated by build tag
type ImportExport interface{} // disabled
