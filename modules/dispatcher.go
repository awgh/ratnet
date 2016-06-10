package modules

import "errors"

// Dispatchers - every message handling module must register types handled here
//               and implement DispatchHandler
var Dispatchers = make(map[uint16]DispatchHandler)

// DispatchHandler Main Interface for Message Handling - implement this in plugin
type DispatchHandler interface {
	HandleDispatch(msg []byte) error
	GetName() string
}

// Dispatch - called by ratnet when a message needs to be handled
func Dispatch(typeID uint16, buf []byte) error {

	//	log.Println("Dispatch:", typeID)

	d, found := Dispatchers[typeID]
	if !found {
		return errors.New("No Dispatcher found for that type")
	}

	return d.HandleDispatch(buf[:])
}
