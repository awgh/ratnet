package defaultlogger

import (
	"log"

	"github.com/awgh/ratnet/api"
)

// StartDefaultLogger - Prints events above the given threshold using log
func StartDefaultLogger(node api.Node, logLevel api.LogLevel) {
	go func() {
		event := <-node.Events()
		if event.Severity >= logLevel {
			log.Printf("logger>  %+v\n", event)
		}
	}()
}
