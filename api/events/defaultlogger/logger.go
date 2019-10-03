package defaultlogger

import (
	"github.com/awgh/ratnet/api"
	"github.com/fatih/color"
)

// StartDefaultLogger - Prints events above the given threshold using log
func StartDefaultLogger(node api.Node, logLevel api.LogLevel) {
	go func() {
		event := <-node.Events()
		if event.Severity >= logLevel && event.Type == api.Log {
			switch event.Severity {
			case api.Info:
				c := color.New(color.FgCyan)
				c.Printf("%+v\n", event.Data)
			case api.Debug:
				c := color.New(color.FgBlue)
				c.Printf("%+v\n", event.Data)
			case api.Warning:
				c := color.New(color.FgYellow)
				c.Printf("%+v\n", event.Data)
			case api.Error:
				c := color.New(color.FgRed)
				c.Printf("%+v\n", event.Data)
			case api.Critical:
				c := color.New(color.FgRed).Add(color.Bold).Add(color.Underline)
				c.Printf("%+v\n", event.Data)
			}
		}
	}()
}
