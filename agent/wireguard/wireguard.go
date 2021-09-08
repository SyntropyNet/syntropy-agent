// wireguard package is a helper for agent to configure
// (kernel or userspace) wireguard tunnels
package wireguard

import (
	"github.com/SyntropyNet/syntropy-agent-go/internal/peermon"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/common"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/multiping"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const pkgName = "Wireguard. "

// TODO: I'm trying to embed anonymous entry in my wireguard implementation/wrapper
// Hope I will get a good mic of stock wgctl and my extentions.
type Wireguard struct {
	wgc         *wgctrl.Client
	peerMonitor *peermon.PeerMonitor
	router      common.SdnRouter
}

// TODO: review and redesign Wireguard implementation.
// Maybe it should be an object, containing WG interface data and separate objects per interface ?
func New(r common.SdnRouter, pm *peermon.PeerMonitor) (*Wireguard, error) {
	wgc, err := wgctrl.New()
	if err != nil {
		return nil, err
	}

	wg := Wireguard{
		wgc:         wgc,
		peerMonitor: pm,
		router:      r,
	}

	return &wg, nil
}

func (wg *Wireguard) PeersMonitor() multiping.PingClient {
	return wg.peerMonitor
}

//func (wg *Wireguard) Router() common.Router {
//	return wg.router
//}

func (wg *Wireguard) Devices() ([]*wgtypes.Device, error) {
	return wg.wgc.Devices()
}

func (wg *Wireguard) Close() error {
	return wg.wgc.Close()
}
