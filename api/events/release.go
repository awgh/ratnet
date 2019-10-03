// +build !debug

package events

import "github.com/awgh/ratnet/api"

// Info - Informational messages (1)
func Info(node api.Node, args ...interface{}) {}

// Debug - Debug messages (2)
func Debug(node api.Node, args ...interface{}) {}

// Warning - Warning messages (3)
func Warning(node api.Node, args ...interface{}) {}

// Error - Error messages (4)
func Error(node api.Node, args ...interface{}) {}

// Critical - Critical error messages (5)
func Critical(node api.Node, args ...interface{}) {
	panic(args)
}
