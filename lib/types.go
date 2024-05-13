package lib

import (
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
	Destination        *[]net.IPAddr `json:"destination"`
	TTL                int           `json:"ttl"`
	NameResolveTimeout int           `json:"name_resolve_timeout"`
	Payload            string        `json:"payload"`
	Count              int           `json:"ping_count"`
	ResponseReceived   map[int]bool  `json:"response_received"`
	Stats              *Stats        `json:"stats"`
}

func (p *Pinger) ToString() string {
	if str, err := json.Marshal(p); err != nil {
		log.Fatal(err)
		return err.Error()
	} else {
		return string(str)
	}
}
