package peerlist

import "github.com/SyntropyNet/syntropy-agent/pkg/multiping/pingdata"

func (pl *PeerList) PingProcess(pr *pingdata.PingData) {
	for addr, peer := range pl.peers {
		// Ignore peers that are conflicting (pifDisabled)
		// or configuration is not yet applied (pifAddPending/pifDelPending)
		if peer.flags != PifNone {
			continue
		}

		val, ok := pr.Get(addr.Addr())
		if !ok {
			// NOTE: PeerMonitor is not creating its own ping list
			// It depends on other pingers and is an additional PingClient in their PingProces line
			// At first it may sound a bit complicate, but in fact it is not.
			// It just looks for its peers in other ping results. And it always founds its peers.
			// NOTE: Do not print error here - PeerMonitor always finds its peers. Just not all of them in one run.
			continue
		}
		peer.Add(val.Latency(), val.Loss())
	}
}
