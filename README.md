# go-ping

[![CI](https://github.com/dmartsapp/go-ping/actions/workflows/ci.yaml/badge.svg)](https://github.com/dmartsapp/go-ping/actions/workflows/ci.yaml)
[![Go Reference](https://pkg.go.dev/badge/github.com/dmartsapp/go-ping.svg)](https://pkg.go.dev/github.com/dmartsapp/go-ping)

A small Go library for sending ICMP echo ("ping") requests, over **IPv4 and IPv6**, without shelling out to the system `ping` binary. It's the ICMP engine behind [shint](https://github.com/dmartsapp/shint)'s `ping` command.

## Install

```bash
go get github.com/dmartsapp/go-ping@v2
```

Requires Go 1.27.1 or newer (see `go.mod`).

## Quick start

```go
pinger, err := netutils.NewPinger("google.com") // resolves both A and AAAA by default
if err != nil {
    log.Fatal(err)
}

pinger.
    SetPingCount(4).
    SetPayloadSizeInBytes(16).
    SetPingDelayInMS(500).
    SetParallelPing(true)

if err := pinger.PingAll(); err != nil {
    log.Fatal(err)
}

pinger.MeasureStats()
fmt.Println(pinger.Stats) // JSON: loss, min/max/avg/stddev in ms, per-packet detail
```

Each resolved address is pinged with the ICMP protocol matching its own family - a dual-stack hostname (both A and AAAA records) is pinged over **both** IPv4 and IPv6 in the same run. Restrict that with `SetNetwork`:

```go
pinger.SetNetwork("ip4") // or "ip6"; "ip" (both) is the default
```

Stream human-readable progress while pinging runs:

```go
go func() {
    for line := range pinger.StreamLog() {
        fmt.Println(line)
    }
}()
pinger.PingAll()
```

## API

`NewPinger(destination string) (*Pinger, error)` resolves `destination` (DNS name or IP literal) and returns a `*Pinger` configured with sensible defaults, ready to run.

| Setter | Default | Meaning |
|---|---|---|
| `SetNetwork(string)` | `"ip"` | `"ip"` (both v4 and v6), `"ip4"`, or `"ip6"` - passed straight through to the same resolver `net.Resolver.LookupIP` itself uses |
| `SetPingCount(int)` | `4` | Number of echo requests per destination; negative values are made positive, values above 1000 are clamped to 1000 |
| `SetParallelPing(bool)` | `false` (sequential) | Send all destinations' requests for an iteration concurrently instead of one at a time |
| `SetPayloadSizeInBytes(int)` | `4` | Echo payload size, clamped to the largest size that fits an unfragmented default-MTU packet |
| `SetPingDelayInMS(int)` | `1000` | Delay between iterations (not applied after the last one) |
| `SetRandomizedPingDelay(bool)` | `false` | Randomize that delay (0-10s) instead of using a fixed value |
| `SetReplyTimeoutInMS(int)` | `1000` | How long a single echo request waits for its reply before being counted as lost |
| `SetResolveTimeout(int)` | `5000` | DNS resolution timeout in milliseconds |

`PingAll() error` sends every configured request and blocks until all replies/timeouts are collected into `Stats.Packets`. `MeasureStats() *Stats` then computes min/max/avg/stddev round-trip time over the successful packets.

## Privileges

Ping uses unprivileged ICMP datagram sockets by default on every platform except Windows:

- **macOS / BSD:** works for a regular user out of the box.
- **Linux:** gated by the `net.ipv4.ping_group_range` / `net.ipv6.ping_group_range` sysctls. Most desktop distributions ship these open to all users; a hardened or minimal distro may restrict them to root - widen the range if `PingAll` reports every packet lost with a permission error: `sudo sysctl -w net.ipv4.ping_group_range="0 2147483647"` (and the `net.ipv6.` equivalent for IPv6).
- **Windows:** always uses a raw ICMP socket, which requires Administrator.

An environment with IPv6 disabled entirely (common in some minimal containers) will simply fail to resolve or ping `AAAA`/`ip6` destinations with a clear error - IPv4 pinging is unaffected.

## Changelog

### v2.0.0

- **Added IPv6 support.** `Pinger.Destination` can now hold a mix of IPv4 and IPv6 addresses (from a dual-stack resolution, or set manually), and each is pinged with the ICMP protocol matching its own family. `NewPinger` resolves both families by default (`SetNetwork` to restrict to one).
- Fixed a data race in `PingAll` (two goroutines wrote the same `error` variable with no synchronization).
- Fixed `SetPayloadSizeInBytes` using `size % max` to clamp: requesting exactly `max` (or a multiple of it) silently produced an *empty* payload instead of a full one. Now clamps properly.
- Fixed sequence-number matching: replies were matched by re-marshaling the reply body and reading its 4th byte, i.e. only the **low byte** of the 16-bit sequence number - pings past #255 could be attributed to the wrong request. Now matches on the actual decoded `Echo.Seq`.
- Fixed `MeasureStats` dividing by a zero success count (all packets lost, or a resolve timeout) and silently producing `NaN`/`Inf` in the JSON output; it now leaves every derived field at zero instead.
- Fixed `Stats.MarshalJSON`: `ResolveTime`/`TotalTime` are `time.Duration` (nanoseconds) and marshaled as such by default, under field names (`resolve_time_ms`, `total_time_taken_ms`) that promised milliseconds. Added a custom `MarshalJSON` so the wire format matches the field names.
- Renamed the misleadingly-named `TTL` field/`SetTTL` method to `ReplyTimeoutMS`/`SetReplyTimeoutInMS`: despite the name, this was never the IP-level hop limit, only a per-echo reply wait time.
- Removed a redundant per-packet delay inside the send path that doubled up with the intentional between-iterations delay, and removed dead platform-stub files (`unix.go`/`windows.go`) that were never called.
- Upgraded to Go 1.27.1 and the latest `golang.org/x/net`/`golang.org/x/sys`.
- Added a real test suite (unit tests plus loopback IPv4/IPv6 integration tests) and this README - there was previously no documentation and no tests at all.

This is a breaking change from v1.x (field/method rename, `Payload` is now `[]byte` instead of `string`), hence the major version bump.

## License

MIT - see [LICENSE](LICENSE).
