package router

import (
	"net/netip"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/agent/peeradata"
	"github.com/SyntropyNet/syntropy-agent/agent/routestatus"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/netcfg"
)

func (r *Router) RouteAdd(netpath *common.SdnNetworkPath, dest ...netip.Prefix) error {
	r.Lock()
	defer r.Unlock()

	for idx, ip := range dest {
		// A very dumb protection from "bricking" servers by adding default routes
		// Allow add default routes only for configured VPN_CLIENT
		// TODO: there are dosens other ways to act as default route, without 0.0.0.0/0 IP
		if !config.IsVPNClient() && netcfg.IsDefaultRoute(&ip) {
			logger.Warning().Println(pkgName, "ignored default route for non configured VPN client")
			continue
		}

		// Some hidden business logic here:
		// Controller sends Allowed_IPs as follows:
		// first entry (index=0) is its WG tunnel peers internal ip ==> need to add host route
		// all other entries are peers LANs (docker, etc) services IPs, that should have SDN routing on them
		// I don't need to send IP address to PeerAdd, because it is the same as netpath.Gateway
		if idx == 0 {
			r.PeerAdd(netpath)
		} else {
			r.ServiceAdd(netpath, ip)
		}
	}

	return nil
}

func (r *Router) RouteDel(netpath *common.SdnNetworkPath, ips ...netip.Prefix) error {
	r.Lock()
	defer r.Unlock()

	for idx, ip := range ips {
		// Some hidden business logic here:
		// Controller sends Allowed_IPs as follows:
		// first entry (index=0) is its WG tunnel peers internal ip ==> need to add host route
		// all other entries are peers LANs (docker, etc) services IPs, that should have SDN routing on them
		// I don't need to send IP address to PeerDel, because it is the same as netpath.Gateway
		if idx == 0 {
			r.PeerDel(netpath)
		} else {
			r.ServiceDel(netpath, ip)
		}
	}

	return nil
}

func (r *Router) Apply() ([]*routestatus.Connection, []*peeradata.Entry) {
	r.Lock()
	defer r.Unlock()

	routeStatusCons := []*routestatus.Connection{}
	peersActiveData := []*peeradata.Entry{}

	// Avoid endless execution, thus limit iteration to 3 times
	for iteration := 3; iteration > 0; iteration-- {
		r.peersApply()
		rsc, pad := r.serviceApply()
		routeStatusCons = append(routeStatusCons, rsc...)
		peersActiveData = append(peersActiveData, pad...)

		// Check and mark as resolved IP conflicting addresses
		// If any IP conflict was resolved - try smart services reconfiguration
		if count := r.ipConflictResolve(); count == 0 {
			// No need to retry if no IP conflict was resolved
			break
		} else {
			logger.Info().Println(pkgName, count, "IP conflicts resolved")
		}
	}

	return routeStatusCons, peersActiveData
}

func (r *Router) HasRoute(ip netip.Prefix) bool {
	r.Lock()
	defer r.Unlock()

	for _, route := range r.routes {
		if route.peerMonitor.HasNode(ip) || route.serviceMonitor.HasAddress(ip) {
			return true
		}
	}

	return false
}

func (r *Router) HasIpConflict(addr netip.Prefix, groupID int) bool {
	for gid, routesGroup := range r.routes {
		if routesGroup.peerMonitor.HasNode(addr) ||
			routesGroup.serviceMonitor.HasAddress(addr) {
			if groupID != gid {
				logger.Error().Println(pkgName, addr.String(), "IP conflict. Connection GIDs:", groupID, gid)
				return true
			}
		}
	}

	return false
}

func (r *Router) ipConflictResolve() (count int) {
	for _, routeGroup := range r.routes {
		count += routeGroup.peerMonitor.ResolveIpConflict(r.HasIpConflict)
		count += routeGroup.serviceMonitor.ResolveIpConflict(r.HasIpConflict)
	}
	return count
}

func (r *Router) Close() error {
	if !config.CleanupOnExit() {
		return nil
	}

	r.Lock()
	defer r.Unlock()

	var err error
	for id, route := range r.routes {
		err = route.peerMonitor.Close()
		if err != nil {
			logger.Error().Println(pkgName, "peer monitor", id, "Close", err)
		}

		err = route.serviceMonitor.Close()
		if err != nil {
			logger.Error().Println(pkgName, "service monitor", id, "Close", err)
		}
	}

	return nil
}

func (r *Router) Flush() {
	r.Lock()
	defer r.Unlock()

	for _, route := range r.routes {
		route.peerMonitor.Flush()
		route.serviceMonitor.Flush()
	}

}
