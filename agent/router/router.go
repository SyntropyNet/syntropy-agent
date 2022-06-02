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

	return r.serviceApply()
}

func (r *Router) HasRoute(ip netip.Prefix) bool {
	r.Lock()
	defer r.Unlock()

	for _, route := range r.routes {
		if route.peerMonitor.HasNode(ip.Addr()) || route.serviceMonitor.Has(ip) {
			return true
		}
	}

	return false
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
