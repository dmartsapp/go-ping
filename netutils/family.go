package netutils

import (
	"net"
	"runtime"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// icmpFamily bundles everything that differs between ICMPv4 and ICMPv6 echo
// handling behind one small, swappable descriptor, so the send/receive path
// in packet.go never has to branch on address family itself beyond picking
// the right one of these two values. Adding support for another ICMP-like
// protocol later is a matter of adding one more *icmpFamily, not touching
// the logic that uses it.
type icmpFamily struct {
	name string

	// unprivilegedNet is the golang.org/x/net/icmp network for a regular,
	// non-root ICMP datagram socket ("udp4"/"udp6"). Linux gates this via
	// the net.ipv4.ping_group_range / net.ipv6.ping_group_range sysctls;
	// macOS and the BSDs allow it unconditionally.
	unprivilegedNet string

	// privilegedNet is the network for a raw ICMP socket. Required on
	// Windows (which has no unprivileged ICMP datagram socket concept) and
	// usable as a fallback anywhere unprivileged sockets are disallowed,
	// given appropriate capabilities/Administrator rights.
	privilegedNet string

	listenAddr      string
	echoRequestType icmp.Type
	echoReplyType   icmp.Type

	// protocolNumber is the IANA protocol number icmp.ParseMessage needs to
	// pick the right type table when decoding a reply (1 = ICMP, 58 =
	// IPv6-ICMP). golang.org/x/net/internal/iana holds the named constants
	// but isn't importable from outside the module, so these are the same
	// literals every other consumer of the package ends up using.
	protocolNumber int
}

var icmpv4Family = &icmpFamily{
	name:            "ipv4",
	unprivilegedNet: "udp4",
	privilegedNet:   "ip4:icmp",
	listenAddr:      _DEFAULT_LISTEN_ADDRESS_V4,
	echoRequestType: ipv4.ICMPTypeEcho,
	echoReplyType:   ipv4.ICMPTypeEchoReply,
	protocolNumber:  1,
}

var icmpv6Family = &icmpFamily{
	name:            "ipv6",
	unprivilegedNet: "udp6",
	privilegedNet:   "ip6:ipv6-icmp",
	listenAddr:      _DEFAULT_LISTEN_ADDRESS_V6,
	echoRequestType: ipv6.ICMPTypeEchoRequest,
	echoReplyType:   ipv6.ICMPTypeEchoReply,
	protocolNumber:  58,
}

// familyFor picks the ICMP family for a resolved destination address. Note
// this is the address's own family, not the Pinger's configured network
// filter - a dual-stack ("ip") resolution can return a mix of v4 and v6
// addresses, and each is pinged with its own matching protocol.
func familyFor(destination net.IP) *icmpFamily {
	if destination.To4() != nil {
		return icmpv4Family
	}
	return icmpv6Family
}

// network returns the golang.org/x/net/icmp network name to use for this
// family on the current platform.
func (f *icmpFamily) network() string {
	if runtime.GOOS == "windows" {
		return f.privilegedNet
	}
	return f.unprivilegedNet
}

// writeAddr returns the net.Addr PacketConn.WriteTo needs for a given
// destination on this platform: a raw ICMP socket (Windows, always) expects
// a *net.IPAddr, while an unprivileged ICMP datagram socket (everywhere
// else) expects a *net.UDPAddr - passing the wrong one doesn't error
// immediately, it just results in every request silently going unanswered.
func (f *icmpFamily) writeAddr(destination net.IP) net.Addr {
	if runtime.GOOS == "windows" {
		return &net.IPAddr{IP: destination}
	}
	return &net.UDPAddr{IP: destination}
}
