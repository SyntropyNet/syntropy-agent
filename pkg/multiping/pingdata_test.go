package multiping

import (
	"fmt"
	"net/netip"
	"testing"
)

func TestPingData(t *testing.T) {
	const maxCount = 222
	data := NewPingData()

	if data.Count() != 0 {
		t.Errorf("Invalid initial PingData count")
	}

	data.Add(netip.MustParseAddr("127.0.0.1"))
	data.Add(netip.MustParseAddr("127.0.0.1"))
	data.Add(netip.MustParseAddr("127.0.0.1"))
	if data.Count() != 1 {
		t.Errorf("Duplicate entries check failed")
	}

	for i := 2; i <= maxCount; i++ {
		data.Add(netip.MustParseAddr(fmt.Sprintf("127.0.0.%d", i)))
	}
	if data.Count() != maxCount {
		t.Errorf("Total count test failed")
	}

	data.Reset()
	if data.Count() != maxCount {
		t.Errorf("Data reset test failed")
	}
	data.Flush()
	if data.Count() != 0 {
		t.Errorf("Data flush test failed")
	}
}

func TestAppend(t *testing.T) {
	// Fake ping data #1
	data := NewPingData()
	data.entries[netip.MustParseAddr("192.168.1.1")] = &PingStats{
		tx:     1,
		rx:     1,
		rtt:    100,
		avgRtt: 100,
	}
	data.entries[netip.MustParseAddr("192.168.1.2")] = &PingStats{
		tx: 1,
	}

	// fake ping data #2
	more := NewPingData()
	more.entries[netip.MustParseAddr("192.168.1.1")] = &PingStats{
		tx:     2,
		rx:     2,
		rtt:    400,
		avgRtt: 40,
	}
	more.entries[netip.MustParseAddr("192.168.1.2")] = &PingStats{
		tx:     1,
		rx:     1,
		rtt:    111,
		avgRtt: 111,
	}
	more.entries[netip.MustParseAddr("10.10.0.2")] = &PingStats{
		tx:     1,
		rx:     1,
		rtt:    102,
		avgRtt: 102,
	}

	// Merge ping data results
	data.Append(more)

	// Test correct merge
	if data.Count() != 3 {
		t.Errorf("Incorrect append count")
	}
	val, _ := data.Get(netip.MustParseAddr("192.168.1.1"))
	if (val != PingStats{
		tx:     3,
		rx:     3,
		rtt:    400,
		avgRtt: 60,
	}) {
		t.Errorf("Entry 1 is not equal")
	}
	val, _ = data.Get(netip.MustParseAddr("192.168.1.2"))
	if (val != PingStats{
		tx:     2,
		rx:     1,
		rtt:    111,
		avgRtt: 111,
	}) {
		t.Errorf("Entry 2 is not equal")
	}

	val, _ = data.Get(netip.MustParseAddr("10.10.0.2"))
	if (val != PingStats{
		tx:     1,
		rx:     1,
		rtt:    102,
		avgRtt: 102,
	}) {
		t.Errorf("Entry 3 is not equal")
	}

	val, _ = data.Get(netip.MustParseAddr("10.200.200.200"))
	if (val != PingStats{
		tx:     0,
		rx:     0,
		rtt:    0,
		avgRtt: 0,
	}) {
		t.Errorf("Empty entry incorrect")
	}
}

const (
	UintSize = 32 << (^uint(0) >> 32 & 1) // 32 or 64
	MaxUint  = 1<<UintSize - 1            // 1<<32 - 1 or 1<<64 - 1
)

func TestLossValidation(t *testing.T) {
	// try listing corner cases here
	testEntries := []PingStats{
		{tx: 0, rx: 0}, // invalid values
		{tx: 1, rx: 1},
		{tx: 1, rx: 0},
		{tx: 0, rx: 1}, // invalid values
		{tx: 2, rx: 1},
		{tx: 1, rx: 2}, // invalid values
		{tx: 222, rx: 125},
		{tx: 111, rx: 225}, // invalid values
		{tx: MaxUint, rx: 0},
		{tx: MaxUint, rx: 10},
		{tx: 0, rx: MaxUint},           // invalid values
		{tx: 100, rx: MaxUint},         // invalid values
		{tx: MaxUint / 2, rx: MaxUint}, // invalid values
		{tx: MaxUint, rx: MaxUint / 2},
		{tx: MaxUint / 2, rx: MaxUint / 2},
		{tx: MaxUint - 1, rx: MaxUint}, // invalid values
		{tx: MaxUint, rx: MaxUint - 1},
		{tx: MaxUint - 1, rx: MaxUint - 1},
		{tx: MaxUint, rx: MaxUint},
	}

	for _, e := range testEntries {
		loss := e.Loss()
		if loss < 0 || loss > 1 {
			t.Errorf("Invalid loss %f (tx: %d, rx: %d)", loss, e.tx, e.rx)
		}
	}
}
