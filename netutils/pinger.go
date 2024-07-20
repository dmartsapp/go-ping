package netutils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"
)

type ICMPPacket struct {
	Destination     net.IPAddr `json:"destination"`
	PayloadSize     int        `json:"payload_size"`
	Sequence        int        `json:"sequence_number"`
	SentDateTime    time.Time  `json:"sent_datetime"`
	ReceiveDateTime time.Time  `json:"receive_datetime"`
}
type Stats struct {
	Packets         []ICMPPacket  `json:"icmp_packets"`
	Loss            int           `json:"loss"`
	Min             time.Duration `json:"min"`
	Max             time.Duration `json:"max"`
	Avg             float64       `json:"avg"`
	StdDev          float64       `json:"stddev"`
	ResolveTime     time.Duration `json:"resolve_time_ms"`
	ResolveTimedOut bool          `json:"is_resolve_timed_out"`
}

type Pinger struct {
	DestinationStr     string   `json:"destination"`
	Destination        []net.IP `json:"destination_ip_addresses"`
	TTL                int      `json:"ttl"`
	ResolveTimeout     int      `json:"resolve_timeout_ms"`
	Payload            string   `json:"payload_data"`
	Count              int      `json:"ping_count"`
	Stats              *Stats   `json:"stats"`
	IsSequential       bool     `json:"is_sequential_ping"`
	PingDelay          int      `json:"ping_delay_ms"`
	RandomizePingDelay bool     `json:"is_ping_delay_random"`
}

var (
	_pinger_wg = sync.WaitGroup{}
)

const (
	_DEFAULT_COUNT              = 4
	_DEFAULT_NETWORK            = "ip4"
	_DEFAULT_RESOLVE_TIMEOUT_MS = 5000
	_DEFAULT_PING_DELAY_MS      = 1000
	_DEFAULT_MAX_DELAY          = 10000
)

func NewPinger(destination string) *Pinger {
	pinger := Pinger{
		DestinationStr: destination,
		Destination:    []net.IP{},
		Payload:        strings.Repeat("d", _DEFAULT_COUNT),
		Count:          _DEFAULT_COUNT,
		Stats:          &Stats{},
		IsSequential:   true,
		ResolveTimeout: _DEFAULT_RESOLVE_TIMEOUT_MS,
		PingDelay:      _DEFAULT_PING_DELAY_MS,
	}

	return &pinger
}

func (pinger *Pinger) Ping() error {
	if err := pinger.resolveName(pinger.DestinationStr); err != nil {
		return err
	}
	if pinger.IsSequential {
		for range pinger.Count {
			for _, ip := range pinger.Destination {
				time.Sleep(time.Millisecond * time.Duration(pinger.PingDelay))
				pinger.sendicmp(ip)
				if pinger.RandomizePingDelay {
					pinger.PingDelay = rand.Intn(_DEFAULT_MAX_DELAY)
				}
			}
		}

	} else {
		for range pinger.Count {
			for _, ip := range pinger.Destination {
				time.Sleep(time.Millisecond * time.Duration(pinger.PingDelay))
				_pinger_wg.Add(1)
				go func(wg *sync.WaitGroup) {
					defer wg.Done()
					pinger.sendicmp(ip)
				}(&_pinger_wg)

				if pinger.RandomizePingDelay {
					pinger.PingDelay = rand.Intn(_DEFAULT_MAX_DELAY)
				}
			}
		}
		_pinger_wg.Wait()
	}
	return nil
}

func (pinger *Pinger) EnableParallelPing() {
	pinger.IsSequential = false
}

func (pinger *Pinger) SetPingerPayloadSize(payload_size int) {
	pinger.Payload = strings.Repeat("d", payload_size)
}

func (pinger *Pinger) SetPingCount(count int) {
	pinger.Count = count
}

func (pinger *Pinger) SetResolveTimeout(timeout int) {
	pinger.ResolveTimeout = timeout
}

func (pinger *Pinger) SetPingDelayInMS(delay int) {
	if delay <= 0 {
		return
	}
	pinger.PingDelay = delay
}

func (pinger *Pinger) SetRandomizedPingDelay(random bool) {
	pinger.RandomizePingDelay = random
}

func (p *Pinger) String() string {
	if str, err := json.Marshal(p); err != nil {
		log.Fatal(err)
		return err.Error()
	} else {
		return string(str)
	}
}

func (pinger *Pinger) resolveName(destination string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(pinger.ResolveTimeout))
	defer cancel()
	start := time.Now()
	addr, err := net.DefaultResolver.LookupIP(ctx, _DEFAULT_NETWORK, destination)
	if err != nil {
		pinger.Stats.ResolveTime = time.Duration(time.Since(start).Milliseconds())
		pinger.Stats.ResolveTimedOut = true
		return err
	}
	pinger.Destination = addr
	pinger.Stats.ResolveTime = time.Duration(time.Since(start).Milliseconds())
	pinger.Stats.ResolveTimedOut = false

	return nil
}

func (pinger *Pinger) sendicmp(ip net.IP) {
	fmt.Println(ip)

}
