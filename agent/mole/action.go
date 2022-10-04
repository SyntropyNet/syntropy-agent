package mole

import (
	"github.com/SyntropyNet/syntropy-agent/agent/peeradata"
	"github.com/SyntropyNet/syntropy-agent/agent/router"
	"github.com/SyntropyNet/syntropy-agent/agent/routestatus"
	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/netcfg"
)

// Some compatibility layer to directly access apprentice
func (m *Mole) Wireguard() *swireguard.Wireguard {
	return m.wg
}

// Some compatibility layer to directly access apprentice
func (m *Mole) Router() *router.Router {
	return m.router
}

// Close and cleanup
func (m *Mole) Close() error {
	m.Lock()
	defer m.Unlock()

	// Delete host routes to peers.
	// These are routes added to connected WG peers via original default gateway.
	// NOTE: other routes will be deleted on interface destroy
	if config.CleanupOnExit() {
		for _, entry := range m.cache.peers {
			if entry.gwIfname != "" {
				logger.Debug().Println(pkgName, "Cleanup peer host route", entry.destIP,
					"on", entry.gwIfname)
				err := netcfg.RouteDel(entry.gwIfname, &entry.destIP)
				if err != nil {
					// Warning and try to continue.
					logger.Warning().Println(pkgName, "peer host route cleanup", err)
				}
			}
		}

		err := m.filter.Close()
		if err != nil {
			logger.Error().Println(pkgName, "iptables close", err)
		}
	}

	m.cleanupControllerRoutes()

	err := m.router.Close()
	if err != nil {
		logger.Error().Println(pkgName, "Router close", err)
	}

	err = m.wg.Close()
	if err != nil {
		logger.Error().Println(pkgName, "Wireguard close", err)
	}

	return nil
}

// Flush old cache (prepare to build new cache)
func (m *Mole) Flush() {
	m.filter.Flush()
	m.wg.Flush()
	m.router.Flush()
}

// Apply pending results (sync cache to reality)
// Send some messages to controller (Writter), if needed
func (m *Mole) Apply() {
	routeStatusMessage := routestatus.New()
	peersActiveDataMessage := peeradata.NewMessage()

	delRoutes, err := m.wg.Apply()
	if err != nil {
		logger.Error().Println(pkgName, "wireguard apply", err)
	}

	// store initial new peers counters values
	// Note: PeerStatsInit is quite smart and does not reset existing peers stats
	m.wg.PeerStatsInit()

	// check and delete routes
	for _, r := range delRoutes {
		if m.router.HasRoute(r) {
			// do not delete routes, if router is still dealing with them
			logger.Warning().Println(pkgName, "Old route should be deleted, but router still has it", r.String())
			continue
		}

		found, ifname := netcfg.RouteSearch(&r)
		if found {
			logger.Debug().Println(pkgName, "Deleting leftover route", r, ifname)
			netcfg.RouteDel(ifname, &r)
		}
	}

	routeRes, peersData := m.router.Apply()

	routeStatusMessage.Add(routeRes...)
	peersActiveDataMessage.Add(peersData...)

	routeStatusMessage.Send(m.writer)
	peersActiveDataMessage.Send(m.writer)
}
