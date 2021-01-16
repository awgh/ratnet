// +build debug

package events

import (
	"github.com/awgh/ratnet/api"
)

// Info - Informational messages (1)
func Info(node api.Node, args ...interface{}) {
	if !node.IsRunning() {
		return
	}
	node.Events() <- api.Event{Severity: api.Info, Type: api.Log, Data: args}
}

// Debug - Debug messages (2)
func Debug(node api.Node, args ...interface{}) {
	if !node.IsRunning() {
		return
	}
	node.Events() <- api.Event{Severity: api.Debug, Type: api.Log, Data: args}
}

// Warning - Warning messages (3)
func Warning(node api.Node, args ...interface{}) {
	if !node.IsRunning() {
		return
	}
	node.Events() <- api.Event{Severity: api.Warning, Type: api.Log, Data: args}
}

// Error - Error messages (4)
func Error(node api.Node, args ...interface{}) {
	if !node.IsRunning() {
		return
	}
	node.Events() <- api.Event{Severity: api.Error, Type: api.Log, Data: args}
}

// Critical - Critical error messages (5)
func Critical(node api.Node, args ...interface{}) {
	if !node.IsRunning() {
		return
	}
	node.Events() <- api.Event{Severity: api.Critical, Type: api.Log, Data: args}
}
