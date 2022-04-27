package servicemon

import (
	"github.com/SyntropyNet/syntropy-agent/agent/peeradata"
	"github.com/SyntropyNet/syntropy-agent/agent/router/peermon"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/netcfg"
)

func (sm *ServiceMonitor) Reroute(selroute *peermon.SelectedRoute) []*peeradata.Entry {
	peersActiveData := []*peeradata.Entry{}

	sm.Lock()
	defer sm.Unlock()

	for dest, routeList := range sm.routes {
		currRoute := routeList.GetActive()
		var newRoute *routeEntry = nil
		if selroute != nil {
			newRoute = routeList.Find(selroute.IP)
			if newRoute == nil {
				logger.Error().Println(pkgName, "New route ", selroute.IP, "not found.")
			}
		}

		ret := routeList.Reroute(newRoute, currRoute, dest)
		if ret != nil {
			peersActiveData = append(peersActiveData, ret)
		}
	}

	return peersActiveData
}

// Reroute one routeList (aka Service Group)
func (rl *routeList) Reroute(newRoute, oldRoute *routeEntry, destination string) *peeradata.Entry {
	switch {
	case newRoute == oldRoute:
		// Nothing to change
		return nil

	case newRoute == nil:
		// Delete active route
		logger.Info().Println(pkgName, "remove route", destination, oldRoute.ifname)
		err := netcfg.RouteDel(oldRoute.ifname, destination)
		if err != nil {
			logger.Error().Println(pkgName, "could not remove route to", destination, "via", oldRoute.ifname)
		}
		// reset active flags
		oldRoute.ClearFlags(rfActive)

		// Return route change
		return peeradata.NewEntry(oldRoute.connectionID, 0, 0)

	case oldRoute == nil:
		// No previous active route was present. Set new route
		logger.Info().Println(pkgName, "add route", destination, newRoute.ifname)
		err := netcfg.RouteAdd(newRoute.ifname, "", destination)
		if err != nil {
			logger.Error().Println(pkgName, "could not add route to", destination, "via", newRoute.ifname)
		}
		// set active flags
		newRoute.SetFlag(rfActive)

		// Return route change
		return peeradata.NewEntry(0, newRoute.connectionID, newRoute.groupID)

	default:
		// Change the route to new active
		logger.Info().Println(pkgName, "replace route", destination, oldRoute.ifname, "->", newRoute.ifname)
		err := netcfg.RouteReplace(newRoute.ifname, "", destination)
		if err != nil {
			logger.Error().Println(pkgName, "could not change routes to", destination, "via", newRoute.ifname)
		}
		// change active flags
		oldRoute.ClearFlags(rfActive)
		newRoute.SetFlag(rfActive)

		// Return route change
		return peeradata.NewEntry(oldRoute.connectionID, newRoute.connectionID, newRoute.groupID)
	}
}
