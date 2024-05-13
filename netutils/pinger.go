package devnutils

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"time"
)

type Stats struct {
	Sent        int           `json:"sent"`
	Received    int           `json:"received"`
	Loss        int           `json:"loss"`
	Min         time.Duration `json:"min"`
	Max         time.Duration `json:"max"`
	Avg         time.Duration `json:"avg"`
	StdDev      time.Duration `json:"stddev"`
	ResolveTime time.Duration `json:"resolvetimems"`
}

type Pinger struct {
	Destination        *net.IPAddr  `json:"destination"`
	TTL                int          `json:"ttl"`
	NameResolveTimeout int          `json:"name_resolve_timeout"`
	Payload            string       `json:"payload"`
	Count              int          `json:"ping_count"`
	ResponseReceived   map[int]bool `json:"response_received"`
	Stats              Stats        `json:"stats"`
}

func NewPinger(destination_name string, resolvetimeout int) *Pinger {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(resolvetimeout))
	start := time.Now()
	addr, err := net.DefaultResolver.LookupIPAddr(ctx, destination_name)
	if err != nil {
		log.Fatal("Unable to resolve destination name")
	}
	defer cancel()
	return &Pinger{
		Destination:        &addr[0],
		NameResolveTimeout: resolvetimeout,
		ResponseReceived:   make(map[int]bool, 100),
		Count:              4,
		Payload:            "devn",
		Stats: Stats{
			Sent:        0,
			Received:    0,
			Loss:        0,
			Min:         0,
			Max:         0,
			Avg:         0,
			StdDev:      0,
			ResolveTime: time.Duration(time.Since(start).Milliseconds()),
		},
	}
}

func (p *Pinger) ToString() string {
	if str, err := json.Marshal(p); err != nil {
		log.Fatal(err)
		return err.Error()
	} else {
		return string(str)
	}
}
