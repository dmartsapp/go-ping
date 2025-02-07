package netutils

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"
)

type WebClient struct {
	_URL                    url.URL
	_MethodType             string
	_DestinationIPAddresses []net.IP
	_DestinationPorts       []int
	_Timeout                int
	_Payload                string
	_Count                  int
	_Stats                  *Stats
	_IsSequential           bool
	_packet_channel         chan Packet
	_log_stream_channel     chan string
}

func NewWebClient(URL string) (*WebClient, error) {
	parsed_url, err := url.Parse(URL)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %w", err) // return nil to indicate failure if there is an error
	}
	webclient := &WebClient{
		_URL:                    *parsed_url,
		_MethodType:             "GET",
		_DestinationIPAddresses: []net.IP{},
		_DestinationPorts:       []int{},
		_Timeout:                1000,
		_Payload:                "",
		_Count:                  1,
		_Stats:                  &Stats{},
		_packet_channel:         make(chan Packet),
		_log_stream_channel:     make(chan string),
	}
	return webclient, nil // return the webclient if there is no error
}

func (webclient *WebClient) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(webclient._Timeout))
	defer cancel()
	start := time.Now()
	addr, err := net.DefaultResolver.LookupIP(ctx, _DEFAULT_NETWORK, webclient._URL.Host)
	if err != nil {
		webclient._Stats.ResolveTime = time.Duration(time.Since(start).Milliseconds())
		webclient._Stats.ResolveTimedOut = true
		return fmt.Errorf("%s", "Unable to resolve for "+webclient._URL.String())
	}
	webclient._DestinationIPAddresses = addr
	webclient._Stats.ResolveTime = time.Duration(time.Since(start).Milliseconds())
	webclient._Stats.ResolveTimedOut = false
	return nil
}
