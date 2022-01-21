package peermon

import (
	"fmt"
)

// peerInfo collects stores and calculates moving average of last [SYNTROPY_PEERCHECK_WINDOW] link measurement
type peerInfo struct {
	endpoint string
	gateway  string
	latency  [valuesCount]float32
	loss     [valuesCount]float32
	index    int
}

func (node *peerInfo) Add(latency, loss float32) {
	node.latency[node.index] = latency
	node.loss[node.index] = loss
	node.index++
	if node.index >= valuesCount {
		node.index = 0
	}
}

func (node *peerInfo) Latency() float32 {
	count := 0
	var sum float32
	for _, val := range node.latency {
		if val > 0 {
			sum = sum + val
			count++
		}
	}
	if count > 0 {
		return sum / float32(count)
	}
	return 0
}

func (node *peerInfo) Loss() float32 {
	count := 0
	var sum float32
	for _, val := range node.loss {
		sum = sum + val
		count++
	}
	if count > 0 {
		return sum / float32(count)
	}
	return 0
}

func (node *peerInfo) StatsIncomplete() bool {
	count := 0
	for _, val := range node.latency {
		if val > 0 {
			count++
		}
	}
	return count != valuesCount
}

func (node *peerInfo) String() string {
	return fmt.Sprintf("%s via %s loss: %f latency %f",
		node.endpoint, node.gateway, node.Loss(), node.Latency())
}
