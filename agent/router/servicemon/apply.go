package servicemon

import (
	"fmt"

	"github.com/SyntropyNet/syntropy-agent/agent/peeradata"
	"github.com/SyntropyNet/syntropy-agent/agent/router/peermon"
	"github.com/SyntropyNet/syntropy-agent/agent/routestatus"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/netcfg"
)

func (sm *ServiceMonitor) Apply() ([]*routestatus.Connection, []*peeradata.Entry) {
	var routeStatusCons []*routestatus.Connection
	var peersActiveData []*peeradata.Entry
	var deleteIPs []string

	sm.Lock()
	defer sm.Unlock()

	bestRoute := sm.routeMonitor.BestPath()

	for ip, rl := range sm.routes {
		add, del := rl.Pending()
		if add == 0 && del == 0 {
			// nothing to do for this group
			continue
		}
		count := rl.Count()
		logger.Info().Printf("%s Apply: add:%d, del:%d, count:%d\n",
			pkgName, add, del, count)
		rl.Dump()

		if add == count && del == 0 {
			routeStatus, _ := rl.SetRoute(ip)
			if routeStatus != nil {
				routeStatusCons = append(routeStatusCons, routeStatus)
			}
		} else if del == count && add == 0 {
			rl.ClearRoute(ip)
			// It is dangerous to delete map entry while iterating.
			// Put a mark for later deletion
			deleteIPs = append(deleteIPs, ip)
		} else {
			if bestRoute != nil {
				rl.MergeRoutes(ip, bestRoute.IP)
			} else {
				rl.MergeRoutes(ip, peermon.NoRoute)
			}

		}

	}

	// Safely remove deleted entries
	for _, ip := range deleteIPs {
		delete(sm.routes, ip)
	}

	newConnID := 0
	// Format response message
	if len(sm.routes) > 0 && bestRoute != nil {
		newConnID = bestRoute.ID
	}

	if sm.connectionID != newConnID {
		peersActiveData = append(peersActiveData,
			peeradata.NewEntry(sm.connectionID, newConnID, 0)) // TODO: GroupID
		sm.connectionID = newConnID
	}
	return routeStatusCons, peersActiveData
}

func (rl *routeList) SetRoute(destination string) (*routestatus.Connection, error) {
	defer rl.resetPending()

	routeConflict, conflictIfName := netcfg.RouteConflict(destination)
	logger.Debug().Println(pkgName, "Apply/SetRoute ", destination)

	if !routeConflict {
		// clean case - no route conflict. Simply add the route
		route := rl.GetDefault()
		if route == nil {
			logger.Error().Println(pkgName, "No new route found to", destination)
			return nil, nil
		}
		// mark route as active
		route.SetFlag(rfActive)
		logger.Info().Println(pkgName, "Route add ", destination, " via ", route.gateway, "/", route.ifname)
		err := netcfg.RouteAdd(route.ifname, "", destination)
		routeRes := routestatus.NewEntry(destination, err)

		if err != nil {
			logger.Error().Println(pkgName, "route add error:", err)
		}
		return routestatus.NewConnection(route.connectionID, route.groupID, routeRes),
			nil
	}

	// Route already exists. Check if it was configured earlier and is valid
	for _, route := range rl.list {
		if route.ifname == conflictIfName {
			// Mark active route to keep cache in sync
			active := rl.GetActive()
			if active != nil {
				active.ClearFlags(rfActive)
			}
			route.SetFlag(rfActive)
			// Return route added OK
			return routestatus.NewConnection(route.connectionID, route.groupID,
					routestatus.NewEntry(destination, nil)),
				nil
		}
	}

	// Route exists but is unknown - inform error
	err := fmt.Errorf("route to %s exists on %s", destination, conflictIfName)
	logger.Error().Println(pkgName, "route add error:", err)
	return nil, err
}

func (rl *routeList) ClearRoute(destination string) error {
	defer rl.resetPending()

	logger.Debug().Println(pkgName, "Apply/ClearRoute ", destination)

	route := rl.GetActive()
	if route == nil {
		return nil
	}

	err := netcfg.RouteDel(route.ifname, destination)
	if err != nil {
		logger.Error().Println(pkgName, destination, "route delete error", err)
	}
	route.ClearFlags(rfActive)

	return nil
}

func (rl *routeList) MergeRoutes(destination string, newgw string) error {
	logger.Debug().Println(pkgName, "Apply/MergeRoute ", destination)

	activeRoute := rl.GetActive()
	var newRoute *routeEntry
	if newgw != peermon.NoRoute {
		newRoute = rl.Find(newgw)
		// check if route change is needed
		// I think this both cases should never happen
		if newRoute == nil {
			logger.Error().Println(pkgName, "New route ", newgw, "not found.")
		} else if newRoute.CheckFlag(rfPendingDel) {
			logger.Error().Println(pkgName, "New active route marked for deletion.", newgw)
			newRoute = nil
		}
	}

	// Build new list of new and old, but not deleted entries
	newList := []*routeEntry{}
	for _, e := range rl.list {
		if !e.CheckFlag(rfPendingDel) {
			newList = append(newList, e)
		}
	}
	// drop old list and keep updated list.
	rl.list = newList
	rl.resetPending()

	// Reuse reroute function to do actual job
	return rl.Reroute(newRoute, activeRoute, destination)
}
