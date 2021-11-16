package stunip

import (
	"net"
	"testing"
)

func TestStunServerList(t *testing.T) {
	prevIP := net.ParseIP("0.0.0.0")
	for _, srv := range stunServers {
		ip, err := checkStunServer(srv)
		if err != nil {
			t.Errorf("STUN server %s failed %s", srv, err)
		}
		if prevIP.IsUnspecified() {
			prevIP = ip
		} else if !ip.Equal(prevIP) {
			// Thinking globally in some cases this test MAY fail
			// But this is not common and generally should not happen.
			t.Errorf("Public IP change. Is %s, expected %s", ip.String(), prevIP.String())
		}
	}

	_, err := checkStunServer("some.invalid.really.bad.server")
	if err == nil {
		t.Errorf("Invalid STUN server test failed")
	}
}
