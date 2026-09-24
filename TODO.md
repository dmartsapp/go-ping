# TODO - planned for shint v4.4.0

A tracking page, not a design doc: what shint's `ping` command needs from this
library that it does not have yet, gathered here so the rewiring shint's
v4.4.0 milestone already calls for happens once, deliberately, instead of
piecemeal. Each item below is verified against the current source (this
branch's base, `main` at `d13489e`), not assumed - file:line references are
real, not recalled. Nothing here is implemented; this branch exists to hold
the list until work starts.

## 1. `--timeout` cannot cover the DNS lookup (shint issue #14)

`NewPinger` resolves the destination *inside the constructor*
(`netutils/pinger.go:70`, `pinger.resolveName(pinger.DestinationStr)`),
using `pinger.ResolveTimeoutMS`, which is set from `_DEFAULT_RESOLVE_TIMEOUT_MS`
in the same struct literal a few lines above (`pinger.go:63`) - **before**
the caller has a `*Pinger` to call `SetResolveTimeout` on. The setter exists
(`pinger.go:219`) and the field is real and plumbed through to `Stats`
(`json:"resolve_timeout_ms"`), but nothing a caller does after `NewPinger`
returns can affect the resolve that already happened inside it. shint's
`ping --timeout` cannot bound the lookup as a result - confirmed by reading
the call site, not assumed from shint's side.

**Fix shape**: `NewPinger` needs to accept the resolve timeout (a variadic
option, a second constructor, or splitting resolution out of the
constructor entirely) so it can be set *before* resolution happens, not after.

## 2. `PingAll` cannot be cancelled (shint issue #25)

`func (pinger *Pinger) PingAll() error` (`pinger.go:81`) takes no
`context.Context` and no other cancellation input at all. Once called, it
runs to completion - `produce`/`consume` on two goroutines, `wg.Wait()` -
with no way to stop it early. shint's Ctrl+C handling works for every other
command (`telnet`, `web`, `udp`, `nmap`, `ntp`, `wol`, `rdns`, `dns` all
watch `ctx.Done()` mid-run and print a partial summary); `ping` alone cannot,
because there is nothing here to cancel.

**Fix shape**: `PingAll(ctx context.Context) error`, checking `ctx.Done()` in
the `produce` loop between requests (and ideally aborting an in-flight read
in `sendICMP` too, not just skipping the next one).

## 3. Timing resolution is milliseconds, not sub-millisecond

`packet.SentDateTimeUNIX`/`ReceiveDateTimeUNIX` are `time.Now().UnixMilli()`
(`packet.go:76`, `94`, `99`, `136`), and `MeasureStats` computes round-trip
as `int(packet.ReceiveDateTimeUNIX - packet.SentDateTimeUNIX)` (`stats.go:91`)
- integer milliseconds. A loopback ping, which genuinely completes in well
under a millisecond, is reported as `0ms`, which reads as suspicious rather
than fast. Switching to `time.Now().UnixMicro()` (or storing `time.Time`
directly and deriving both) is a small, mostly mechanical change, but touches
every place that currently assumes millisecond units, including the JSON
field names (`_ms` suffixes) - a compatibility question to settle deliberately
rather than let slide.

## 4. An ICMP error reply and a plain timeout are the same field

`ErrorEncountered bool` (`packet.go:46`) is set by the single `fail()` helper
(`packet.go:54`) from whatever Go error `sendICMP` produced - a read-deadline
timeout and an actual ICMP error message received back (Destination
Unreachable, Time Exceeded, ...) both just become `ErrorEncountered: true`,
`ErrorStr: err.Error()`. The original structure of what came back (a real
ICMP message with a type/code, versus nothing arriving at all before the
deadline) is discarded at the point `fail()` is called. shint's `--json`
callers (and `isLostPing` in `shint/lib/handlers/icmp.go`, which pattern-matches
the *log text* "no reply" as a stand-in for this exact distinction, because
nothing more structured exists) cannot currently tell "the host is down" from
"something between here and there said it is unreachable."

**Fix shape**: `sendICMP`'s read path needs to distinguish a read-deadline
timeout (`net.Error.Timeout()`) from a successfully-read ICMP message that is
not the expected echo reply (type/code exposed on `ICMPPacket`, not just a
flattened error string).

## 5. No source address / interface binding

Raised while discussing whether shint could route outgoing checks through a
specific interface (2026-09-23): nothing in `Pinger` sets a local address for
outgoing ICMP - it always uses whatever the OS default-routes each resolved
destination through. A `--interface`/`--source` flag on shint's `ping` needs
this added here first; shint's other commands (TCP via `net.Dialer.LocalAddr`,
UDP the same way) can do this without any library change, so `ping` catching
up here is what unblocks a *shint-wide* version of the flag, not just
`ping`'s own.

**Fix shape**: resolve the interface name to its own address (the portable,
unprivileged path - see the shint-side discussion for why this is preferred
over `SO_BINDTODEVICE`/`IP_BOUND_IF`), then bind the ICMP socket to it before
sending. Needs checking per-platform: the raw-socket path (macOS, Windows,
privileged Linux) and the unprivileged "ping socket" path
(`kernelOwnsEchoID`, `packet.go:36`) may need this wired in differently.

## 6. Reply TTL / hop limit is not exposed

Also part of the original v4.4.0 "richer ping" plan: the reply's IP-level
TTL (IPv4) / hop limit (IPv6) is read off the wire by the OS but not
surfaced on `ICMPPacket` today. Real ping tools show it (`ttl=54` and
similar) as a rough, useful hint about the path (a changed TTL between
runs can mean a routing change). Needs checking whether Go's `icmp`/raw
socket read path on each platform actually exposes it without extra work
(IP_RECVTTL and friends) before assuming it is cheap to add.

---

Order isn't fixed, but #1 and #2 block real shint issues (#14, #25) and are
the most contained; #3 and #4 are related (both touch `ICMPPacket`/`fail`,
worth doing together); #5 and #6 are newer, less scoped, and should probably
get their own design pass before code, the way #5 got some real discussion
on the shint side already.
