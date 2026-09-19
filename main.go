// Command go-ping is a small usage example for the netutils package. The
// package is meant to be imported (see README.md); this binary just
// exercises it end to end against a couple of well-known dual-stack hosts.
package main

import (
	"fmt"
	"sync"

	"github.com/dmartsapp/go-ping/v2/netutils"
)

func main() {
	pinger, err := netutils.NewPinger("google.com")
	if err != nil {
		fmt.Println(err)
		return
	}

	pinger.
		SetPingCount(4).
		SetPayloadSizeInBytes(16).
		SetPingDelayInMS(500).
		SetParallelPing(true)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for line := range pinger.StreamLog() {
			fmt.Println(line)
		}
	}()

	if err := pinger.PingAll(); err != nil {
		fmt.Println(err)
		return
	}
	wg.Wait()

	pinger.MeasureStats()
	fmt.Println(pinger.Stats)
}
