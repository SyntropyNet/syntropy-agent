package router

import (
	"github.com/SyntropyNet/syntropy-agent/agent/peeradata"
	"github.com/SyntropyNet/syntropy-agent/agent/routestatus"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

// Apply actually executes router configuration changes
// Several iterations are done in order to check and resolve possible IP conflicts
func (r *Router) Apply() ([]*routestatus.Connection, []*peeradata.Entry) {
	r.Lock()
	defer r.Unlock()

	routeStatusCons := []*routestatus.Connection{}
	peersActiveData := []*peeradata.Entry{}

	// Avoid endless execution, thus limit iteration to 3 times
	for iteration := 3; iteration > 0; iteration-- {
		rsc, pad := r.applyChanges()
		routeStatusCons = append(routeStatusCons, rsc...)
		peersActiveData = append(peersActiveData, pad...)

		// Check and mark as resolved IP conflicting addresses
		// If any IP conflict was resolved - try smart services reconfiguration
		if count := r.resolveIpConflict(); count == 0 {
			// No need to retry if no IP conflict was resolved
			break
		} else {
			logger.Info().Println(pkgName, count, "IP conflicts resolved")
		}
	}

	return routeStatusCons, peersActiveData
}

// applyChanges iterates all GIDs and executes configuration changes
// for peers and monitored services
// The caller is responsible for locking
func (r *Router) applyChanges() ([]*routestatus.Connection, []*peeradata.Entry) {
	deleteGIDs := []int{}

	routeStatusCons := []*routestatus.Connection{}
	peersActiveData := []*peeradata.Entry{}

	for gid, route := range r.routes {
		// Apply peers
		err := route.peerMonitor.Apply()
		if err != nil {
			logger.Error().Println(pkgName, "Apply peers for GID", gid, "error", err)
		}

		// Apply services and report route adding
		rsc, pad := route.serviceMonitor.Apply()
		routeStatusCons = append(routeStatusCons, rsc...)
		peersActiveData = append(peersActiveData, pad...)

		// Check and delete empty GIDs
		if route.peerMonitor.Count() == 0 {
			deleteGIDs = append(deleteGIDs, gid)
			if servicesCount := route.serviceMonitor.Count(); servicesCount > 0 {
				logger.Warning().Println(pkgName, "Invalid configuration for GID", gid,
					" Has 0 peers and", servicesCount, "services left. Forcing deletion of services.")
			}
		}
	}

	// remove deleted gids
	for _, gid := range deleteGIDs {
		delete(r.routes, gid)
	}

	return routeStatusCons, peersActiveData
}
