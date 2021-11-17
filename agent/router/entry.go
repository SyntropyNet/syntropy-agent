package router

import (
	"github.com/SyntropyNet/syntropy-agent-go/agent/router/peermon"
	"github.com/SyntropyNet/syntropy-agent-go/agent/router/servicemon"
)

type routerGroupEntry struct {
	peerMonitor    *peermon.PeerMonitor
	serviceMonitor *servicemon.ServiceMonitor
}

func newRouteGroupEntry() *routerGroupEntry {
	return &routerGroupEntry{
		peerMonitor:    &peermon.PeerMonitor{},
		serviceMonitor: servicemon.New(),
	}
}

func (r *Router) findOrCreate(groupID int) *routerGroupEntry {
	routesGroup, ok := r.routes[groupID]
	if !ok {
		routesGroup = newRouteGroupEntry()
		r.routes[groupID] = routesGroup
	}
	return routesGroup
}

func (r *Router) find(groupID int) (*routerGroupEntry, bool) {
	routesGroup, ok := r.routes[groupID]
	return routesGroup, ok
}
