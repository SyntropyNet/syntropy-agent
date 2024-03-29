// swireguard package is wireguard on steroids
// super-wireguard, smart-wireguar, Syntropy-wireguard
// This package is a helper for agent to configure
// (kernel or userspace) wireguard tunnels
// It also collects peer status, monitores latency, and other releated work
package swireguard

import (
	"net/netip"
	"strings"
	"sync"

	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/env"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"golang.zx2c4.com/wireguard/wgctrl"
)

const pkgName = "Wireguard. "

type Wireguard struct {
	// If true - remove resident non-syntropy created tunnels
	RemoveNonSyntropyInterfaces bool

	sync.RWMutex
	wgc *wgctrl.Client
	// NOTE: caching wireguard setup may sound like an overhead at first.
	// But in future we may need to add checking/syncing/recreating delete interfaces
	devices []*InterfaceInfo
}

// New creates new instance of Wireguard configurer and monitor
func New() (*Wireguard, error) {
	wgc, err := wgctrl.New()
	if err != nil {
		return nil, err
	}

	wg := Wireguard{
		wgc:                         wgc,
		RemoveNonSyntropyInterfaces: false,
	}

	loadKernelModule()

	return &wg, nil
}

func (wg *Wireguard) Devices() []*InterfaceInfo {
	wg.RLock()
	defer wg.RUnlock()

	rv := []*InterfaceInfo{}

	rv = append(rv, wg.devices...)

	return rv
}

func (wg *Wireguard) Close() error {
	// If configured - cleanup created interfaces on exit.
	if config.CleanupOnExit() {
		logger.Info().Println(pkgName, "Deleting interfaces.")
		for _, dev := range wg.devices {
			wg.RemoveInterface(dev)
		}
	}

	return wg.wgc.Close()
}

// Flush clears all WG local cache
func (wg *Wireguard) Flush() {
	wg.Lock()
	defer wg.Unlock()

	wg.devices = wg.devices[:0]
}

// Apply function setups cached WG configuration,
// and cleans up resident configuration
func (wg *Wireguard) Apply() (allowedIPs []netip.Prefix, err error) {
	wg.RLock()
	defer wg.RUnlock()

	osDevs, err := wg.wgc.Devices()
	if err != nil {
		return
	}

	// remove resident devices (created by already terminated agent)
	for _, osDev := range osDevs {
		found := false
		for _, agentDev := range wg.devices {
			if osDev.Name == agentDev.IfName {
				found = true
				break
			}
		}

		if !found {
			if strings.HasPrefix(osDev.Name, env.InterfaceNamePrefix) ||
				wg.RemoveNonSyntropyInterfaces {
				for _, peer := range osDev.Peers {
					for _, aIP := range peer.AllowedIPs {
						addr, ok := netip.AddrFromSlice(aIP.IP)
						if !ok {
							continue
						}
						bitlen, _ := aIP.Mask.Size()
						allowedIPs = append(allowedIPs, netip.PrefixFrom(addr, bitlen))
					}
				}
				wg.RemoveInterface(&InterfaceInfo{
					IfName: osDev.Name,
				})
			}
		}
	}

	// reread OS setup - it may has changed
	osDevs, err = wg.wgc.Devices()
	if err != nil {
		return
	}
	// create missing devices
	for _, agentDev := range wg.devices {
		found := false
		for _, osDev := range osDevs {
			if osDev.Name == agentDev.IfName {
				found = true
			}
		}

		if !found {
			wg.CreateInterface(agentDev)
		}
		ips, err := wg.applyPeers(agentDev)
		if err != nil {
			return allowedIPs, err
		}
		allowedIPs = append(allowedIPs, ips...)
	}

	return allowedIPs, err
}
