package api

// LogLevel - Severity value for events
type LogLevel int

// Info is the lowest, Critical the highest
const (
	Info LogLevel = iota
	Debug
	Warning
	Error
	Critical
)

// EventType - type of event
type EventType int

//
const (
	Log EventType = iota
)

// Event - Ratnet Events
type Event struct {
	Severity LogLevel
	Type     EventType
	Data     []interface{}
}
