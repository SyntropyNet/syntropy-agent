package peermon

import (
	"testing"

	"github.com/SyntropyNet/syntropy-agent/internal/config"
)

func TestPeerMonitor(t *testing.T) {
	count := 24
	pm := New(uint(count))

	addNode := func(gw, ip string) {
		pm.AddNode("ifname", "PublicKey", gw, ip, 0)
	}

	fillStats := func(endpoint string, latency, loss float32) {
		for _, peer := range pm.peerList {
			if peer.endpoint == endpoint {
				for i := 0; i < count; i++ {
					peer.Add(latency, loss)
				}
			}
		}

	}

	addNode("1.1.1.1", "1.1.1.2")
	addNode("2.2.2.1", "2.2.2.2")
	addNode("3.3.3.1", "3.3.3.2")
	addNode("4.4.4.1", "4.4.4.2")
	pm.lastBest = 0

	// Lower loss is always must
	fillStats("1.1.1.2", 100, 0.02)
	fillStats("2.2.2.2", 145, 0.11)
	fillStats("3.3.3.2", 500, 0)
	fillStats("4.4.4.2", 105, 0.3)
	best := pm.BestPath()
	if best != "3.3.3.1" {
		t.Errorf("Lowest loss test failed %s", best)
	}

	// Test without thresholds
	config.SetRerouteThresholds(0, 1)
	pm.lastBest = 0
	fillStats("1.1.1.2", 100, 0)
	fillStats("2.2.2.2", 145, 0)
	fillStats("3.3.3.2", 250, 0)
	fillStats("4.4.4.2", 95, 0)
	best = pm.BestPath()
	if best != "4.4.4.1" {
		t.Errorf("Test without threshold %s", best)
	}

	// Set thresholds and test
	config.SetRerouteThresholds(10, 1.05)
	pm.lastBest = 0
	best = pm.BestPath()
	if best != "1.1.1.1" {
		t.Errorf("Test with too big threshold %s", best)
	}

	config.SetRerouteThresholds(5, 1.05)
	pm.lastBest = 0
	best = pm.BestPath()
	if best != "4.4.4.1" {
		t.Errorf("Test with correct threshold %s", best)
	}

	// test incomplete statistics
	pm.lastBest = 0
	pm.peerList[3].Add(0, 0)
	best = pm.BestPath()
	if best != "1.1.1.1" {
		t.Errorf("Test with incomplete statistics %s", best)
	}
}
