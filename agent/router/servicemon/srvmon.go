package servicemon

import (
	"fmt"
	"net/netip"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/agent/router/peermon/routeselector"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

const pkgName = "ServiceMonitor. "

// ServiceMonitor monitors routes to configured services
// Does rerouting when PathSelector.BestPath() changes
// ServiceMonitor is explicitely used in Router and is always under main Router lock
// So no need for locking here
type ServiceMonitor struct {
	routes             map[netip.Prefix]*routeList
	routeMonitor       routeselector.PathSelector
	groupID            int
	activeConnectionID int
}

func New(ps routeselector.PathSelector, gid int) *ServiceMonitor {
	return &ServiceMonitor{
		routes:             make(map[netip.Prefix]*routeList),
		routeMonitor:       ps,
		groupID:            gid,
		activeConnectionID: 0,
	}
}

func (sm *ServiceMonitor) Add(netpath *common.SdnNetworkPath, ip netip.Prefix, disabled bool) error {
	// Keep a list of active SDN routes
	if sm.routes[ip] == nil {
		sm.routes[ip] = newRouteList(netpath.GroupID, disabled)
	}

	sm.routes[ip].Add(&routeEntry{
		ifname:       netpath.Ifname,
		publicKey:    netpath.PublicKey,
		gateway:      netpath.Gateway,
		connectionID: netpath.ConnectionID,
	})

	return nil
}

func (sm *ServiceMonitor) Del(netpath *common.SdnNetworkPath, ip netip.Prefix) error {
	// Keep a list of active SDN routes
	if sm.routes[ip] == nil {
		return fmt.Errorf("no such address %s", ip)
	}

	sm.routes[ip].MarkDel(netpath.Gateway)

	return nil
}

func (sm *ServiceMonitor) HasAddress(ip netip.Prefix) bool {
	rl, ok := sm.routes[ip]
	// ignore not applied addresses
	if ok && !rl.Disabled() {
		add, del := rl.Pending()
		// check and ignore services that will be deleted
		if add == 0 && del == rl.Count() {
			return false
		}
		return true
	}

	return false
}

func (sm *ServiceMonitor) Close() error {
	for ip, rl := range sm.routes {
		if rl.Disabled() {
			// no need to delete routes that were not added
			// conflicting IP was detected (and prevented)
			continue
		}

		rl.clearRoute(ip)
	}

	// delete map entries
	for ip := range sm.routes {
		delete(sm.routes, ip)
	}

	return nil
}

func (sm *ServiceMonitor) Flush() {
	var deleteIPs []netip.Prefix

	for ip, rl := range sm.routes {
		if rl.Disabled() {
			// no need to do smart routes delete and merge for routes that were not added
			// because conflicting IP was detected (and prevented)
			// instead flush them asap
			deleteIPs = append(deleteIPs, ip)
			continue
		}

		logger.Debug().Println(pkgName, "Flushing", ip)
		rl.Flush()
	}

	// now flush/delete the conflicting addresses
	for _, ip := range deleteIPs {
		logger.Debug().Println(pkgName, "Flushing (previously IP conflict)", ip)
		delete(sm.routes, ip)
	}
}

func (sm *ServiceMonitor) Dump() {
	for ip, rl := range sm.routes {
		logger.Debug().Println(pkgName, ip, rl)
	}
}
