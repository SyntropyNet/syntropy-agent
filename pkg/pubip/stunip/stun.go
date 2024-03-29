// stunip gets public IP from public STUN servers
package stunip

import (
	"fmt"
	"net"

	"github.com/pion/stun"
)

var lastGoodIdx int

// PublicIP tries optimise STUN server lists.
// If server fails - it tries another one from list
// When server responds successfully - next time it will be tried first.
// Very simple and straightforward implementation.
func PublicIP() (net.IP, error) {
	for i := 0; i < len(stunServers); i++ {
		ip, err := checkStunServer(stunServers[lastGoodIdx])

		if err == nil && ip != nil {
			// Return IP address and stay on same server
			return ip, nil
		} else {
			// Server failed - try next one
			lastGoodIdx++
			if lastGoodIdx >= len(stunServers) {
				lastGoodIdx = 0
			}
		}
	}
	return net.IP{}, fmt.Errorf("could not get public ip address")
}

func checkStunServer(srv string) (net.IP, error) {
	var ipAddress net.IP
	var callbackError error

	callback := func(res stun.Event) {
		if res.Error != nil {
			callbackError = res.Error
			return
		}

		// Decoding XOR-MAPPED-ADDRESS attribute from message.
		var xorAddr stun.XORMappedAddress
		callbackError = xorAddr.GetFrom(res.Message)
		if callbackError != nil {
			return
		}
		if xorAddr.IP != nil {
			ipAddress = xorAddr.IP
		} else {
			callbackError = fmt.Errorf("could not parse STUN result")
		}
	}

	// Creating a "connection" to STUN server.
	// By default we want an IPv4, thus "udp4"
	c, err := stun.Dial("udp4", srv)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	// Building binding request with random transaction id.
	message, err := stun.Build(stun.TransactionID, stun.BindingRequest)
	if err != nil {
		return nil, err
	}
	// Sending request to STUN server, waiting for response message.
	err = c.Do(message, callback)

	if err != nil {
		return nil, err
	}

	// Don't get confused here. Callback is called async thus has its own callback error
	// Don't do optimisations here with stun.Do error and callbackError

	return ipAddress, callbackError
}
