package multiping

import (
	"math/rand"
	"net/netip"
	"testing"

	"golang.org/x/net/icmp"
)

func TestPingPacket(t *testing.T) {
	ip := netip.MustParseAddr("127.0.0.1")

	var seq uint16 = 0
	for {
		seq = seq<<1 + 1
		id := uint16(rand.Intn(0xffff))

		pinger := NewPinger("ip", "icmp", id)
		if pinger == nil {
			t.Fatalf("Could not create pinger")
		}
		pinger.SetIPAddr(&ip)

		msgBytes, err := pinger.prepareICMP(seq)
		if err != nil {
			t.Fatalf("Icmp prepare %s", err)
		}

		packet, err := icmp.ParseMessage(protocolICMP, msgBytes)
		if err != nil {
			t.Fatalf("Icmp parse %s", err)
		}

		data, ok := packet.Body.((*icmp.Echo))
		if !ok {
			t.Fatalf("Invalid packet body")
		}

		if data.Seq != int(seq) {
			t.Fatalf("Invalid sequence")
		}

		if data.ID != int(id) {
			t.Fatalf("Invalid ID")
		}

		if seq == 0xffff {
			break
		}
	}
}
