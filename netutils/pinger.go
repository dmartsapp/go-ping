package devnutils

import (
	"context"
	"log"
	"net"
	"time"

	lib "github.com/farhansabbir/goping/lib"
)

func NewPinger(destination_name string, resolvetimeout int) *lib.Pinger {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(resolvetimeout))
	start := time.Now()
	addr, err := net.DefaultResolver.LookupIPAddr(ctx, destination_name)
	if err != nil {
		log.Fatal("Unable to resolve destination name")
	}
	defer cancel()
	return &lib.Pinger{
		Destination:        &addr,
		NameResolveTimeout: resolvetimeout,
		ResponseReceived:   make(map[int]bool, 100),
		Count:              4,
		Payload:            "devn",
		Stats: &lib.Stats{
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
