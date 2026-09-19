package netutils

import (
	"net"
	"runtime"
	"testing"
)

func TestFamilyForIPv4(t *testing.T) {
	f := familyFor(net.ParseIP("127.0.0.1"))
	if f != icmpv4Family {
		t.Errorf("familyFor(127.0.0.1) = %s, want ipv4", f.name)
	}
}

func TestFamilyForIPv4MappedDottedQuad(t *testing.T) {
	f := familyFor(net.ParseIP("8.8.8.8"))
	if f != icmpv4Family {
		t.Errorf("familyFor(8.8.8.8) = %s, want ipv4", f.name)
	}
}

func TestFamilyForIPv6(t *testing.T) {
	f := familyFor(net.ParseIP("::1"))
	if f != icmpv6Family {
		t.Errorf("familyFor(::1) = %s, want ipv6", f.name)
	}
}

func TestFamilyForIPv6GlobalUnicast(t *testing.T) {
	f := familyFor(net.ParseIP("2001:4860:4860::8888"))
	if f != icmpv6Family {
		t.Errorf("familyFor(2001:4860:4860::8888) = %s, want ipv6", f.name)
	}
}

func TestFamilyNetworkPicksPlatformVariant(t *testing.T) {
	want := icmpv4Family.unprivilegedNet
	if runtime.GOOS == "windows" {
		want = icmpv4Family.privilegedNet
	}
	if got := icmpv4Family.network(); got != want {
		t.Errorf("icmpv4Family.network() = %q, want %q (GOOS=%s)", got, want, runtime.GOOS)
	}
}

func TestFamiliesHaveDistinctProtocolNumbers(t *testing.T) {
	if icmpv4Family.protocolNumber == icmpv6Family.protocolNumber {
		t.Error("icmpv4Family and icmpv6Family must not share a protocol number")
	}
	// 1 = ICMP, 58 = IPv6-ICMP per IANA; pinned here since packet.go's use
	// of icmp.ParseMessage depends on these exact values.
	if icmpv4Family.protocolNumber != 1 {
		t.Errorf("icmpv4Family.protocolNumber = %d, want 1", icmpv4Family.protocolNumber)
	}
	if icmpv6Family.protocolNumber != 58 {
		t.Errorf("icmpv6Family.protocolNumber = %d, want 58", icmpv6Family.protocolNumber)
	}
}
