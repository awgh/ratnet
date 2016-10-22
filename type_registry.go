package ratnet

import "github.com/awgh/ratnet/api"

// One map for each type of types:  Routers, Connection Policies, and Transports

var (
	// Routers : Registry of available Router modules by name
	Routers map[string]func(map[string]interface{}) api.Router
	// Policies : Registry of available Policy modules by name
	Policies map[string]func(api.Transport, api.Node, map[string]interface{}) api.Policy
	// Transports : Registry of available Transport modules by name
	Transports map[string]func(api.Node, map[string]interface{}) api.Transport
)

func init() {
	Routers = make(map[string]func(map[string]interface{}) api.Router)
	Policies = make(map[string]func(api.Transport, api.Node, map[string]interface{}) api.Policy)
	Transports = make(map[string]func(api.Node, map[string]interface{}) api.Transport)
}

// NewTransportFromMap : Create a new instance of a Transport from a map of arguments
func NewTransportFromMap(node api.Node, t map[string]interface{}) api.Transport {
	ttype := t["type"].(string)
	return Transports[ttype](node, t)
}

// NewRouterFromMap : Create a new instance of a Router from a map of arguments
func NewRouterFromMap(r map[string]interface{}) api.Router {
	rtype := r["type"].(string)
	return Routers[rtype](r)
}

// NewPolicyFromMap : Create a new instance of a Policy from a map of arguments
func NewPolicyFromMap(transport api.Transport, node api.Node, p map[string]interface{}) api.Policy {
	ptype := p["type"].(string)
	return Policies[ptype](transport, node, p)
}
