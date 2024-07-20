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
	Destination         net.IPAddr `json:"destination"`
	PayloadSize         int        `json:"payload_size"`
	Sequence            int        `json:"sequence_number"`
	SentDateTimeUNIX    int64      `json:"sent_datetime_unix_ms"`
	ReceiveDateTimeUNIX int64      `json:"receive_datetime_unix_ms"`
}
type Stats struct {
	Packets         []ICMPPacket  `json:"icmp_packets"`
	Loss            int           `json:"loss"`
	Min             time.Duration `json:"min_ms"`
	Max             time.Duration `json:"max_ms"`
	Avg             float64       `json:"avg_ms"`
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
	MTU                int      `json:"mtu"`
}

var (
	_pinger_wg      = sync.WaitGroup{}
	_pinger_channel = make(chan ICMPPacket, 1)
	// _pinger_mutux = sync.Mutex{}
)

const (
	_DEFAULT_PING_COUNT         = 4
	_DEFAULT_PAYLOAD_SIZE       = 4
	_DEFAULT_MTU                = 1500
	_DEFAULT_MAX_PAYLOAD_SIZE   = _DEFAULT_MTU - 16 - 20 - 16 // MTU - 16 bytes_of_src_datetime - 20 bytes_of_icmp_header - 16 bytes_of_checksum
	_DEFAULT_NETWORK            = "ip4"
	_DEFAULT_RESOLVE_TIMEOUT_MS = 5000
	_DEFAULT_PING_DELAY_MS      = 1000
	_DEFAULT_MAX_DELAY          = 10000
)

func NewPinger(destination string) *Pinger {
	pinger := Pinger{
		DestinationStr: destination,
		Destination:    []net.IP{},
		Payload:        strings.Repeat("d", _DEFAULT_PAYLOAD_SIZE),
		Count:          _DEFAULT_PING_COUNT,
		Stats:          &Stats{},
		IsSequential:   true,
		ResolveTimeout: _DEFAULT_RESOLVE_TIMEOUT_MS,
		PingDelay:      _DEFAULT_PING_DELAY_MS,
		MTU:            _DEFAULT_MTU,
	}

	return &pinger
}

func (pinger *Pinger) Ping() error {
	// resolve the name first to populate pinger object properties
	if err := pinger.resolveName(pinger.DestinationStr); err != nil {
		return err
	}
	_pinger_wg.Add(1)
	// start monitoring the pinger channel for incoming data from completed pings
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		for packet := range _pinger_channel {
			pinger.Stats.Packets = append(pinger.Stats.Packets, packet)
		}
		fmt.Println("Calculate the stats, now that pingers have stopped sending packets")
	}(&_pinger_wg)

	if pinger.IsSequential {
		for seq := range pinger.Count {
			for _, ip := range pinger.Destination {
				pinger.sendicmp(ip, seq)
				if pinger.RandomizePingDelay {
					pinger.PingDelay = rand.Intn(_DEFAULT_MAX_DELAY)
				}
				time.Sleep(time.Millisecond * time.Duration(pinger.PingDelay))
			}
		}
		close(_pinger_channel)
	} else {
		for seq := range pinger.Count {
			for _, ip := range pinger.Destination {
				_pinger_wg.Add(1)
				go func(wg *sync.WaitGroup) {
					defer wg.Done()
					pinger.sendicmp(ip, seq)
				}(&_pinger_wg)

				if pinger.RandomizePingDelay {
					pinger.PingDelay = rand.Intn(_DEFAULT_MAX_DELAY)
				}
				time.Sleep(time.Millisecond * time.Duration(pinger.PingDelay))
			}
		}
		_pinger_wg.Wait()
		close(_pinger_channel)
	}
	return nil
}

func (pinger *Pinger) EnableParallelPing() {
	pinger.IsSequential = false
}

func (pinger *Pinger) SetPayloadSizeInBytes(payload_size int) {
	// explicitly sets the size of the ping requests within boundary of _DEFAULT_MAX_PAYLOAD_SIZE
	// returns nil
	pinger.Payload = strings.Repeat("d", payload_size%_DEFAULT_MAX_PAYLOAD_SIZE)
}

func (pinger *Pinger) SetPingCount(count int) {
	// explicitly set ping count. Checks if set below 0, then converts to absolute
	// default is usually 4 as defined in _DEFAULT_PING_COUNT
	// returns nil
	if count < 0 {
		count *= -1
	}
	pinger.Count = count
}

func (pinger *Pinger) SetResolveTimeout(timeout int) error {
	// explicitly set ping delay. Checks for timeout less than 0ms
	// default is usually 5000ms as defined in _DEFAULT_RESOLVE_TIMEOUT_MS
	// returns error if timeout < 0
	if timeout < 0 {
		return fmt.Errorf("timeout must not be less than 0ms")
	}
	pinger.ResolveTimeout = timeout
	return nil
}

func (pinger *Pinger) SetPingDelayInMS(delay int) error {
	// explicitly set ping delay. Checks for delay
	// default is usually 1000ms as defined in _DEFAULT_PING_DELAY_MS
	// returns error if delay <= 0
	if delay <= 0 {
		return fmt.Errorf("delay cannot be less than 1ms. Set randomized ping delay to achieve delay between pings")
	}
	pinger.PingDelay = delay
	return nil
}

func (pinger *Pinger) SetRandomizedPingDelay(random bool) {
	// explicitly set if ping delay should be randomized
	// returns nil
	pinger.RandomizePingDelay = random
}

func (p *Pinger) String() string {
	// returns json representation of the pinger object
	if str, err := json.Marshal(p); err != nil {
		log.Fatal(err)
		return err.Error()
	} else {
		return string(str)
	}
}

func (pinger *Pinger) resolveName(destination string) error {
	// method resolves the name against a timeout defined in ResolveTimeout
	// also populates basic properties like
	// - resolved addresses and
	// - time taken to resolve
	// - if timed out to resolve, marks resolvedtimedout to true
	// - returns error if error encountered while resolve execution
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

func (pinger *Pinger) sendicmp(ip net.IP, seq int) {
	time.Sleep(time.Millisecond * time.Duration(pinger.PingDelay))
	packet := ICMPPacket{
		Destination: net.IPAddr{
			IP: ip,
		},
		Sequence:         seq,
		PayloadSize:      len(pinger.Payload),
		SentDateTimeUNIX: time.Now().UnixMilli(),
	}
	sendReq()
	_pinger_channel <- packet
}
