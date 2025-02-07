package netutils

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"
)

type WebClient struct {
	URL                    url.URL  `json:"url"`
	MethodType             string   `json:"request_type"`
	DestinationIPAddresses []net.IP `json:"destination_ip_addresses"`
	DestinationPorts       []int    `json:"destination_ports"`
	Timeout                int      `json:"timeout_ms"`
	Payload                string   `json:"payload_data"`
	Count                  int      `json:"ping_count"`
	Stats                  *Stats   `json:"stats"`
	IsSequential           bool     `json:"is_sequential_ping"`
	_packet_channel        chan Packet
	_log_stream_channel    chan string
}

func (webclient *WebClient) Ping() error {
	// method resolves the name against a timeout defined in ResolveTimeout
	// also populates basic properties like
	// - resolved addresses and
	// - time taken to resolve
	// - if timed out to resolve, marks resolvedtimedout to true
	// - returns error if error encountered while resolve execution
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(webclient.Timeout))
	defer cancel()
	start := time.Now()
	addr, err := net.DefaultResolver.LookupIP(ctx, _DEFAULT_NETWORK, webclient.URL.Host)
	if err != nil {
		webclient.Stats.ResolveTime = time.Duration(time.Since(start).Milliseconds())
		webclient.Stats.ResolveTimedOut = true
		return fmt.Errorf("%s", "Unable to resolve for " + webclient.URL.String())
	}
	webclient.DestinationIPAddresses = addr
	webclient.Stats.ResolveTime = time.Duration(time.Since(start).Milliseconds())
	webclient.Stats.ResolveTimedOut = false

	return nil
}
