package netutils

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type WebClient struct {
	_URL                    url.URL
	_MethodType             string
	_DestinationIPAddresses []net.IP
	_DestinationPort        int
	_Timeout                int
	_Header                 map[string]string
	_Body                   string
	_Count                  int
	_Stats                  *Stats
	_IsSequential           bool
	_AgentName              string
	_packet_channel         chan Packet
	_log_stream_channel     chan string
}

func NewWebClient(URL string) (*WebClient, error) {
	parsed_url, err := url.Parse(URL)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %w", err) // return nil to indicate failure if there is an error
	}
	port, err := strconv.Atoi(parsed_url.Port())
	if err != nil {
		return nil, fmt.Errorf("error parsing port: %w", err) // return nil to indicate failure if there is an error
	}
	webclient := &WebClient{
		_URL:                    *parsed_url,
		_MethodType:             "GET",
		_DestinationIPAddresses: []net.IP{},
		_DestinationPort:        port,
		_Timeout:                1000,
		_Header:                 map[string]string{"user-agent": _DEFAULT_HTTP_CLIENT_USER_AGENT},
		_Body:                   "",
		_Count:                  1,
		_Stats:                  &Stats{},
		_packet_channel:         make(chan Packet),
		_log_stream_channel:     make(chan string),
	}
	return webclient, nil // return the webclient if there is no error
}

func (webclient *WebClient) Ping() error {
	err := webclient.ResolveHost()
	if err != nil {
		return err
	}

	return nil
}

func (webclient *WebClient) SendRequest(method string) error {
	issupported, ok := _DEFAULT_SUPPORTED_WEB_METHODS[strings.ToUpper(method)]
	if !ok || !issupported {
		return fmt.Errorf("%s method is not supported", method)
	}

	return nil

}

func (webclient *WebClient) SetHeader(header map[string]string) *WebClient {
	webclient._Header = header
	return webclient
}

func (webclient *WebClient) SetTimeoutInMS(timeout int) *WebClient {
	webclient._Timeout = timeout
	return webclient
}

func (webclient *WebClient) SetBody(body string) *WebClient {
	webclient._Body = body
	return webclient
}

func (webclient *WebClient) SetCount(count int) *WebClient {
	webclient._Count = count
	return webclient
}

func (webclient *WebClient) SetSequentialRequests(sequential bool) *WebClient {
	webclient._IsSequential = sequential
	return webclient
}

func (webclient *WebClient) SetMethodType(method string) *WebClient {
	webclient._MethodType = method
	return webclient
}

func (webclient *WebClient) ResolveHost() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(webclient._Timeout))
	defer cancel()
	start := time.Now()
	addr, err := net.DefaultResolver.LookupIP(ctx, _DEFAULT_NETWORK, webclient._URL.Hostname())
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
