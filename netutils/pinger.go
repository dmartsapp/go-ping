package netutils

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"strings"
	"time"
)

type ICMPPacket struct {
	Destination     *net.IPAddr `json:"destination"`
	PayloadSize     int         `json:"payload_size"`
	Sequence        int         `json:"sequence_number"`
	SentDateTime    time.Time   `json:"sent_datetime"`
	ReceiveDateTime time.Time   `json:"receive_datetime"`
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
	DestinationStr     string    `json:"destination"`
	Destination        *[]net.IP `json:"destination_ip_addresses"`
	TTL                int       `json:"ttl"`
	NameResolveTimeout int       `json:"name_resolve_timeout_ms"`
	Payload            string    `json:"payload"`
	Count              int       `json:"ping_count"`
	Stats              *Stats    `json:"stats"`
	IsSequential       bool      `json:"is_sequential_ping"`
}

var (
// pinger_wg sync.WaitGroup
)

const (
	_DEFAULT_COUNT   = 4
	_DEFAULT_NETWORK = "ip4"
)

func NewPinger(destination string) *Pinger {
	pinger := Pinger{
		DestinationStr:     destination,
		Payload:            strings.Repeat("d", _DEFAULT_COUNT),
		Count:              _DEFAULT_COUNT,
		Stats:              &Stats{},
		IsSequential:       true,
		NameResolveTimeout: 5000,
	}

	return &pinger
}

func (pinger *Pinger) Ping() error {
	if err := pinger.resolveName(pinger.DestinationStr); err != nil {
		return err
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

func (p *Pinger) ToString() string {
	if str, err := json.Marshal(p); err != nil {
		log.Fatal(err)
		return err.Error()
	} else {
		return string(str)
	}
}

func (pinger *Pinger) resolveName(destination string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(pinger.NameResolveTimeout))
	defer cancel()
	start := time.Now()
	addr, err := net.DefaultResolver.LookupIP(ctx, _DEFAULT_NETWORK, destination)
	if err != nil {
		pinger.Stats = &Stats{
			ResolveTime:     time.Duration(time.Since(start).Milliseconds()),
			ResolveTimedOut: true,
		}
		return err
	}
	pinger.Destination = &addr
	pinger.Stats = &Stats{ResolveTime: time.Duration(time.Since(start).Milliseconds()), ResolveTimedOut: false}
	return nil
}
