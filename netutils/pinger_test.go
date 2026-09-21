package netutils

import (
	"net"
	"strings"
	"testing"
	"time"
)

func TestNewPingerResolvesLiteralIPv4(t *testing.T) {
	pinger, err := NewPinger("127.0.0.1")
	if err != nil {
		t.Fatalf("NewPinger error: %v", err)
	}
	if len(pinger.Destination) != 1 || !pinger.Destination[0].Equal(net.ParseIP("127.0.0.1")) {
		t.Errorf("Destination = %v, want [127.0.0.1]", pinger.Destination)
	}
}

func TestNewPingerResolvesLiteralIPv6(t *testing.T) {
	pinger, err := NewPinger("::1")
	if err != nil {
		t.Fatalf("NewPinger error: %v", err)
	}
	if len(pinger.Destination) != 1 || !pinger.Destination[0].Equal(net.ParseIP("::1")) {
		t.Errorf("Destination = %v, want [::1]", pinger.Destination)
	}
}

func TestNewPingerInvalidHost(t *testing.T) {
	_, err := NewPinger("this-host-should-not-exist.invalid")
	if err == nil {
		t.Fatal("expected an error for an unresolvable host, got nil")
	}
}

func TestSetPingCountClampsNegativeToPositive(t *testing.T) {
	pinger := &Pinger{}
	pinger.SetPingCount(-5)
	if pinger.Count != 5 {
		t.Errorf("Count = %d, want 5 (absolute value of -5)", pinger.Count)
	}
}

func TestSetPingCountClampsAboveMax(t *testing.T) {
	pinger := &Pinger{}
	pinger.SetPingCount(_DEFAULT_MAX_PING_COUNT + 1000)
	if pinger.Count != _DEFAULT_MAX_PING_COUNT {
		t.Errorf("Count = %d, want %d", pinger.Count, _DEFAULT_MAX_PING_COUNT)
	}
}

func TestSetPayloadSizeInBytesClampsAtExactMax(t *testing.T) {
	// Regression test: the pre-2.0 implementation used payload_size % max,
	// so requesting exactly _DEFAULT_MAX_PAYLOAD_SIZE (or any multiple of
	// it) silently produced an EMPTY payload instead of a full one.
	pinger := &Pinger{}
	pinger.SetPayloadSizeInBytes(_DEFAULT_MAX_PAYLOAD_SIZE)
	if len(pinger.Payload) != _DEFAULT_MAX_PAYLOAD_SIZE {
		t.Errorf("len(Payload) = %d, want %d (exact max should clamp, not wrap to 0)", len(pinger.Payload), _DEFAULT_MAX_PAYLOAD_SIZE)
	}
}

func TestSetPayloadSizeInBytesClampsAboveMax(t *testing.T) {
	pinger := &Pinger{}
	pinger.SetPayloadSizeInBytes(_DEFAULT_MAX_PAYLOAD_SIZE * 3)
	if len(pinger.Payload) != _DEFAULT_MAX_PAYLOAD_SIZE {
		t.Errorf("len(Payload) = %d, want %d", len(pinger.Payload), _DEFAULT_MAX_PAYLOAD_SIZE)
	}
}

func TestSetPayloadSizeInBytesNegativeBecomesZero(t *testing.T) {
	pinger := &Pinger{}
	pinger.SetPayloadSizeInBytes(-1)
	if len(pinger.Payload) != 0 {
		t.Errorf("len(Payload) = %d, want 0 for a negative request", len(pinger.Payload))
	}
}

func TestSetNetworkAcceptsOnlyValidValues(t *testing.T) {
	pinger := &Pinger{Network: "ip"}
	pinger.SetNetwork("ip4")
	if pinger.Network != "ip4" {
		t.Errorf("Network = %q, want ip4", pinger.Network)
	}
	pinger.SetNetwork("bogus")
	if pinger.Network != "ip4" {
		t.Errorf("Network = %q, want unchanged ip4 after an invalid SetNetwork call", pinger.Network)
	}
}

func TestSetReplyTimeoutInMSRejectsNonPositive(t *testing.T) {
	pinger := &Pinger{}
	pinger.SetReplyTimeoutInMS(0)
	if pinger.ReplyTimeoutMS != _DEFAULT_REPLY_TIMEOUT_MS {
		t.Errorf("ReplyTimeoutMS = %d, want default %d for a zero request", pinger.ReplyTimeoutMS, _DEFAULT_REPLY_TIMEOUT_MS)
	}
	pinger.SetReplyTimeoutInMS(250)
	if pinger.ReplyTimeoutMS != 250 {
		t.Errorf("ReplyTimeoutMS = %d, want 250", pinger.ReplyTimeoutMS)
	}
}

// pingLoopbackOrSkip runs a small PingAll against host (expected to be a
// loopback address) and skips the test - rather than failing it - if every
// packet failed for a permission-shaped reason: unprivileged ICMP sockets
// depend on OS configuration (net.ipv4.ping_group_range /
// net.ipv6.ping_group_range on Linux) that a sandboxed test environment may
// not grant.
func pingLoopbackOrSkip(t *testing.T, host string) *Pinger {
	t.Helper()
	pinger, err := NewPinger(host)
	if err != nil {
		t.Fatalf("NewPinger(%q) error: %v", host, err)
	}
	pinger.SetPingCount(2).SetParallelPing(false).SetPingDelayInMS(10).SetReplyTimeoutInMS(2000)
	if err := pinger.PingAll(); err != nil {
		t.Fatalf("PingAll error: %v", err)
	}
	pinger.MeasureStats()

	if pinger.Stats.Loss == len(pinger.Stats.Packets) {
		for _, p := range pinger.Stats.Packets {
			if strings.Contains(strings.ToLower(p.ErrorStr), "permission") || strings.Contains(strings.ToLower(p.ErrorStr), "not permitted") || strings.Contains(strings.ToLower(p.ErrorStr), "address family not supported") {
				t.Skipf("unprivileged ICMP not available for %s in this environment: %s", host, p.ErrorStr)
			}
		}
	}
	return pinger
}

func TestPingAllIPv4Loopback(t *testing.T) {
	pinger := pingLoopbackOrSkip(t, "127.0.0.1")
	if pinger.Stats.Loss > 0 {
		t.Errorf("Loss = %d, want 0 pinging IPv4 loopback; errors: %v", pinger.Stats.Loss, packetErrors(pinger))
	}
	if len(pinger.Stats.Packets) != 2 {
		t.Errorf("got %d packets, want 2", len(pinger.Stats.Packets))
	}
}

func TestPingAllIPv6Loopback(t *testing.T) {
	pinger := pingLoopbackOrSkip(t, "::1")
	if pinger.Stats.Loss > 0 {
		t.Errorf("Loss = %d, want 0 pinging IPv6 loopback; errors: %v", pinger.Stats.Loss, packetErrors(pinger))
	}
	if len(pinger.Stats.Packets) != 2 {
		t.Errorf("got %d packets, want 2", len(pinger.Stats.Packets))
	}
}

func packetErrors(pinger *Pinger) []string {
	var errs []string
	for _, p := range pinger.Stats.Packets {
		if p.ErrorEncountered {
			errs = append(errs, p.ErrorStr)
		}
	}
	return errs
}

func TestPingAllParallelModeNoRace(t *testing.T) {
	// Exercises the concurrent sendICMP fan-out; run with -race to catch
	// any data races in Stats/packet channel handling.
	pinger := mustNewPinger(t, "127.0.0.1")
	pinger.SetPingCount(8).SetParallelPing(true).SetPingDelayInMS(5).SetReplyTimeoutInMS(2000)
	if err := pinger.PingAll(); err != nil {
		t.Fatalf("PingAll error: %v", err)
	}
	if len(pinger.Stats.Packets) != 8 {
		t.Errorf("got %d packets, want 8", len(pinger.Stats.Packets))
	}
}

func TestPingAllRejectsZeroCount(t *testing.T) {
	pinger := mustNewPinger(t, "127.0.0.1")
	pinger.Count = 0
	if err := pinger.PingAll(); err == nil {
		t.Error("expected an error for Count=0, got nil")
	}
}

func TestPingAllDualStackMixesFamilies(t *testing.T) {
	// A Pinger can be handed a manually mixed destination list (as if a
	// dual-stack host resolved to both); each address should be pinged with
	// its own matching ICMP family without one family's failure affecting
	// the other.
	pinger := mustNewPinger(t, "127.0.0.1")
	pinger.Destination = []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}
	pinger.SetPingCount(1).SetParallelPing(true).SetReplyTimeoutInMS(2000)
	if err := pinger.PingAll(); err != nil {
		t.Fatalf("PingAll error: %v", err)
	}
	if len(pinger.Stats.Packets) != 2 {
		t.Fatalf("got %d packets, want 2 (one v4 + one v6)", len(pinger.Stats.Packets))
	}
	sawV4, sawV6 := false, false
	for _, p := range pinger.Stats.Packets {
		if p.Destination.IP.To4() != nil {
			sawV4 = true
		} else {
			sawV6 = true
		}
	}
	if !sawV4 || !sawV6 {
		t.Errorf("expected one v4 and one v6 packet, got sawV4=%v sawV6=%v", sawV4, sawV6)
	}
}

func mustNewPinger(t *testing.T, host string) *Pinger {
	t.Helper()
	pinger, err := NewPinger(host)
	if err != nil {
		t.Fatalf("NewPinger(%q) error: %v", host, err)
	}
	return pinger
}

// A reply that belongs to another ping must never satisfy this one. Every ICMP
// socket on macOS (and every raw socket) is handed every echo reply the host
// receives, and requests used to carry identifier seq & 0xffff - the same number
// for the first request of every ping - so a ping to an address nobody answers
// (TEST-NET-1, RFC 5737) succeeded whenever another ping got a reply.
func TestReplyToAnotherPingIsNotAccepted(t *testing.T) {
	// Six rounds: before the fix a round was fooled about nine times in ten.
	for round := 1; round <= 6; round++ {
		victim := mustNewPinger(t, "192.0.2.1")
		victim.SetPingCount(1).SetReplyTimeoutInMS(1200)
		neighbour := mustNewPinger(t, "127.0.0.1")
		neighbour.SetPingCount(1).SetReplyTimeoutInMS(1200)

		done := make(chan struct{})
		go func() {
			defer close(done)
			time.Sleep(200 * time.Millisecond) // its reply lands while the victim is still waiting
			_ = neighbour.PingAll()
		}()
		if err := victim.PingAll(); err != nil {
			t.Fatalf("PingAll error: %v", err)
		}
		<-done
		victim.MeasureStats()
		neighbour.MeasureStats()
		if neighbour.Stats.Loss == 1 {
			t.Skipf("unprivileged ICMP not available here: %v", packetErrors(neighbour))
		}
		if victim.Stats.Loss != 1 {
			t.Fatalf("round %d: a ping to an address nobody answers succeeded (loss %d): it took another ping's reply as its own", round, victim.Stats.Loss)
		}
	}
}

// Two destinations of one Pinger, pinged in parallel with the same sequence
// number: the dead one must not adopt the live one's reply.
func TestParallelDestinationsDoNotAdoptEachOthersReplies(t *testing.T) {
	for round := 1; round <= 3; round++ {
		pinger := mustNewPinger(t, "127.0.0.1")
		pinger.Destination = []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("192.0.2.1")}
		pinger.SetPingCount(1).SetParallelPing(true).SetReplyTimeoutInMS(1000)
		if err := pinger.PingAll(); err != nil {
			t.Fatalf("PingAll error: %v", err)
		}
		pinger.MeasureStats()
		answered := map[string]bool{}
		for _, p := range pinger.Stats.Packets {
			answered[p.Destination.IP.String()] = !p.ErrorEncountered
		}
		if !answered["127.0.0.1"] {
			t.Skipf("unprivileged ICMP not available here: %v", packetErrors(pinger))
		}
		if answered["192.0.2.1"] {
			t.Fatalf("round %d: 192.0.2.1 was reported as answering, with the reply that belonged to 127.0.0.1", round)
		}
	}
}

func TestEchoIDsAreUniquePerRequest(t *testing.T) {
	seen := map[int]bool{}
	for i := 0; i < 1000; i++ {
		id := nextEchoID()
		if id < 0 || id > 0xffff {
			t.Fatalf("id %d does not fit the 16-bit ICMP field", id)
		}
		if seen[id] {
			t.Fatalf("id %d handed out twice within 1000 requests", id)
		}
		seen[id] = true
	}
}
