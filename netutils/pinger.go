package devnutils

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type Stats struct {
	Sent        int           `json:"sent"`
	Received    int           `json:"received"`
	Loss        int           `json:"loss"`
	Min         time.Duration `json:"min"`
	Max         time.Duration `json:"max"`
	Avg         float64       `json:"avg"`
	StdDev      float64       `json:"stddev"`
	ResolveTime time.Duration `json:"resolve_time_ms"`
}

type Pinger struct {
	Destination        *[]net.IPAddr `json:"destination"`
	TTL                int           `json:"ttl"`
	NameResolveTimeout int           `json:"name_resolve_timeout"`
	Payload            string        `json:"payload"`
	Count              int           `json:"ping_count"`
	ResponsesReceived  map[int]bool  `json:"response_received"`
	Stats              *Stats        `json:"stats"`
	IsSequential       bool          `json:"is_sequential_ping"`
}

var (
// pinger_wg sync.WaitGroup
)

func (p *Pinger) ToString() string {
	if str, err := json.Marshal(p); err != nil {
		log.Fatal(err)
		return err.Error()
	} else {
		return string(str)
	}
}

func (pinger *Pinger) SetPingerPayloadSize(payload_size int) {
	pinger.Payload = strings.Repeat("d", payload_size)
}

func (pinger *Pinger) ResolveName(destination string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(pinger.NameResolveTimeout))
	defer cancel()
	start := time.Now()
	addr, err := net.DefaultResolver.LookupIPAddr(ctx, destination)
	if err != nil {
		return err
	}
	pinger.Destination = &addr
	pinger.Stats = &Stats{ResolveTime: time.Duration(time.Since(start).Milliseconds())}
	return nil
}

func (pinger *Pinger) Ping(pinger_wg *sync.WaitGroup, pinger_channel chan *Pinger) {
	defer pinger_wg.Done()
	time.Sleep(time.Second)
	pinger_channel <- pinger
}

func NewPinger(resolvetimeout int, issequential bool, ping_count int) *Pinger {
	return &Pinger{
		Destination:        nil,
		NameResolveTimeout: resolvetimeout,
		ResponsesReceived:  make(map[int]bool, ping_count),
		Count:              ping_count,
		Payload:            "devn",
		IsSequential:       issequential,
		Stats:              nil,
	}
}

func NewPingerNameResolved(destination string, resolvetimeout int, issequential bool, ping_count int) (*Pinger, error) {
	pinger := NewPinger(resolvetimeout, issequential, ping_count)
	if err := (pinger).ResolveName(destination); err != nil {
		return nil, err
	} else {
		return pinger, nil
	}
}
