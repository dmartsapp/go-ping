package netutils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Pinger sends ICMP echo requests to every address a hostname resolves to,
// dispatching each one over ICMPv4 or ICMPv6 depending on that address's own
// family. A dual-stack host (both A and AAAA records) is pinged over both
// protocols in the same run - see SetNetwork to restrict that.
type Pinger struct {
	DestinationStr string   `json:"destination"`
	Destination    []net.IP `json:"destination_ip_addresses"`

	// Network is the resolution family: "ip" (default, both v4 and v6),
	// "ip4", or "ip6". Matches the network argument net.Resolver.LookupIP
	// itself takes, so there's nothing new to learn here.
	Network string `json:"network"`

	// ReplyTimeoutMS is how long a single echo request waits for its reply
	// before being counted as lost. (In releases before v2.0.0 this was
	// misleadingly named TTL - it was never the IP-level hop limit.)
	ReplyTimeoutMS int `json:"reply_timeout_ms"`

	ResolveTimeoutMS   int    `json:"resolve_timeout_ms"`
	Payload            []byte `json:"-"`
	Count              int    `json:"ping_count"`
	Stats              *Stats `json:"stats"`
	IsSequential       bool   `json:"is_sequential_ping"`
	PingDelay          int    `json:"ping_delay_ms"`
	RandomizePingDelay bool   `json:"is_ping_delay_random"`

	// MTU bounds both the reply read buffer and (via SetPayloadSizeInBytes)
	// the largest payload that will fit in a single unfragmented packet.
	MTU int `json:"mtu"`

	packetCh  chan ICMPPacket
	logCh     chan string
	pingsDone int32 // atomic bool
}

// NewPinger resolves destination (per Network, "ip"/dual-stack by default)
// and returns a Pinger ready to be configured with the SetXxx methods and
// run with PingAll.
func NewPinger(destination string) (*Pinger, error) {
	pinger := &Pinger{
		DestinationStr:     destination,
		Network:            _DEFAULT_NETWORK,
		ReplyTimeoutMS:     _DEFAULT_REPLY_TIMEOUT_MS,
		Destination:        []net.IP{},
		Payload:            []byte(repeatByte('d', _DEFAULT_PAYLOAD_SIZE)),
		Count:              _DEFAULT_MIN_PING_COUNT,
		Stats:              &Stats{},
		IsSequential:       true,
		ResolveTimeoutMS:   _DEFAULT_RESOLVE_TIMEOUT_MS,
		PingDelay:          _DEFAULT_PING_DELAY_MS,
		MTU:                _DEFAULT_MTU,
		packetCh:           make(chan ICMPPacket, _DEFAULT_MAX_PING_COUNT),
		logCh:              make(chan string, _DEFAULT_MAX_PING_COUNT),
	}
	start := time.Now()
	if err := pinger.resolveName(pinger.DestinationStr); err != nil {
		pinger.Stats.TotalTime = time.Since(start)
		return nil, fmt.Errorf("unable to resolve %q: %w", destination, err)
	}
	pinger.Stats.ResolveTime = time.Since(start)
	return pinger, nil
}

// PingAll sends Count echo requests to every resolved destination address
// (sequentially or in parallel per SetParallelPing) and blocks until every
// reply or timeout is collected into Stats.Packets.
func (pinger *Pinger) PingAll() error {
	if pinger.Count <= 0 {
		return errors.New("invalid ping count")
	}
	if len(pinger.Destination) == 0 {
		return errors.New("no resolved destination addresses to ping")
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := pinger.produce(); err != nil {
			errCh <- err
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		pinger.consume()
	}()

	wg.Wait()
	close(errCh)
	return <-errCh // nil if the channel was never written to and is now closed
}

// produce sends every request (sequentially or fanned out in parallel,
// per IsSequential) and closes both channels once the last one has been
// sent, so consume's range loop terminates.
func (pinger *Pinger) produce() error {
	start := time.Now()
	defer func() {
		pinger.setPingerComplete()
		pinger.Stats.TotalTime = time.Since(start)
		close(pinger.logCh)
		close(pinger.packetCh)
	}()

	for iteration := 1; iteration <= pinger.Count; iteration++ {
		if pinger.IsSequential {
			for _, ip := range pinger.Destination {
				pinger.sendICMP(ip, iteration)
			}
		} else {
			var wg sync.WaitGroup
			for _, ip := range pinger.Destination {
				wg.Add(1)
				go func(ip net.IP) {
					defer wg.Done()
					pinger.sendICMP(ip, iteration)
				}(ip)
			}
			wg.Wait()
		}

		if iteration == pinger.Count {
			break // no delay after the very last iteration
		}
		if pinger.RandomizePingDelay {
			time.Sleep(time.Millisecond * time.Duration(rand.IntN(_DEFAULT_MAX_DELAY_MS)))
		} else {
			time.Sleep(time.Millisecond * time.Duration(pinger.PingDelay))
		}
	}
	return nil
}

func (pinger *Pinger) consume() {
	for packet := range pinger.packetCh {
		pinger.Stats.Packets = append(pinger.Stats.Packets, packet)
	}
}

func (pinger *Pinger) IsPingComplete() bool {
	return atomic.LoadInt32(&pinger.pingsDone) == 1
}

func (pinger *Pinger) setPingerComplete() {
	atomic.StoreInt32(&pinger.pingsDone, 1)
}

func (pinger *Pinger) logToStreamChannel(line string) {
	pinger.logCh <- line
}

// StreamLog returns a channel of human-readable progress lines, closed once
// PingAll finishes sending every request.
func (pinger *Pinger) StreamLog() <-chan string {
	return pinger.logCh
}

// SetNetwork restricts (or, with "ip", restores) which address families are
// resolved and pinged: "ip" for both v4 and v6 (the default), "ip4" for
// IPv4 only, "ip6" for IPv6 only. Must be called before the addresses are
// used (i.e. right after NewPinger, before PingAll) since it doesn't
// re-resolve on its own.
func (pinger *Pinger) SetNetwork(network string) *Pinger {
	switch network {
	case "ip", "ip4", "ip6":
		pinger.Network = network
	}
	return pinger
}

func (pinger *Pinger) SetParallelPing(parallel bool) *Pinger {
	pinger.IsSequential = !parallel
	return pinger
}

// SetPayloadSizeInBytes sets the echo payload size, clamped to
// _DEFAULT_MAX_PAYLOAD_SIZE rather than wrapped: requesting exactly (or a
// multiple of) the max used to silently produce an empty payload instead.
func (pinger *Pinger) SetPayloadSizeInBytes(payloadSize int) *Pinger {
	if payloadSize < 0 {
		payloadSize = 0
	}
	if payloadSize > _DEFAULT_MAX_PAYLOAD_SIZE {
		payloadSize = _DEFAULT_MAX_PAYLOAD_SIZE
	}
	pinger.Payload = []byte(repeatByte('d', payloadSize))
	return pinger
}

func (pinger *Pinger) SetPingCount(count int) *Pinger {
	if count < 0 {
		count = -count
	}
	if count > _DEFAULT_MAX_PING_COUNT {
		count = _DEFAULT_MAX_PING_COUNT
	}
	pinger.Count = count
	return pinger
}

func (pinger *Pinger) SetResolveTimeout(timeoutMS int) *Pinger {
	if timeoutMS < 0 {
		timeoutMS = _DEFAULT_RESOLVE_TIMEOUT_MS
	}
	pinger.ResolveTimeoutMS = timeoutMS
	return pinger
}

func (pinger *Pinger) SetPingDelayInMS(delayMS int) *Pinger {
	if delayMS < 0 {
		delayMS = _DEFAULT_PING_DELAY_MS
	}
	pinger.PingDelay = delayMS
	return pinger
}

// SetReplyTimeoutInMS sets how long a single echo request waits for its
// reply before being counted as lost. Replaces the pre-2.0 SetTTL, which
// set this same value under a name that implied it was the IP-level hop
// limit; it never was.
func (pinger *Pinger) SetReplyTimeoutInMS(timeoutMS int) *Pinger {
	if timeoutMS <= 0 {
		timeoutMS = _DEFAULT_REPLY_TIMEOUT_MS
	}
	pinger.ReplyTimeoutMS = timeoutMS
	return pinger
}

func (pinger *Pinger) SetRandomizedPingDelay(random bool) *Pinger {
	pinger.RandomizePingDelay = random
	return pinger
}

func (p *Pinger) String() string {
	str, err := json.Marshal(p)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// resolveName resolves DestinationStr per Network and records the outcome
// (address list, timing, or timeout) on Stats.
func (pinger *Pinger) resolveName(destination string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(pinger.ResolveTimeoutMS))
	defer cancel()
	start := time.Now()
	addrs, err := net.DefaultResolver.LookupIP(ctx, pinger.Network, destination)
	if err != nil {
		pinger.Stats.ResolveTime = time.Since(start)
		pinger.Stats.ResolveTimedOut = true
		return err
	}
	pinger.Destination = addrs
	pinger.Stats.ResolveTime = time.Since(start)
	pinger.Stats.ResolveTimedOut = false
	return nil
}

func repeatByte(b byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}
