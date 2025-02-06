package netutils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Pinger struct {
	DestinationStr      string   `json:"destination"`
	Destination         []net.IP `json:"destination_ip_addresses"`
	TTL                 int      `json:"ttl"`
	ResolveTimeout      int      `json:"resolve_timeout_ms"`
	Payload             string   `json:"payload_data"`
	Count               int      `json:"ping_count"`
	Stats               *Stats   `json:"stats"`
	IsSequential        bool     `json:"is_sequential_ping"`
	PingDelay           int      `json:"ping_delay_ms"`
	RandomizePingDelay  bool     `json:"is_ping_delay_random"`
	MTU                 int      `json:"mtu"`
	_packet_channel     chan ICMPPacket
	_log_stream_channel chan string
	_is_ping_done       int32
}

func NewPinger(destination string) (*Pinger, error) {
	pinger := Pinger{
		DestinationStr:      destination,
		TTL:                 _DEFAULT_TTL,
		Destination:         []net.IP{},
		Payload:             strings.Repeat("d", _DEFAULT_PAYLOAD_SIZE),
		Count:               _DEFAULT_MIN_PING_COUNT,
		Stats:               &Stats{},
		IsSequential:        true,
		ResolveTimeout:      _DEFAULT_RESOLVE_TIMEOUT_MS,
		PingDelay:           _DEFAULT_PING_DELAY_MS,
		MTU:                 _DEFAULT_MTU,
		_packet_channel:     make(chan ICMPPacket, _DEFAULT_MAX_PING_COUNT),
		_log_stream_channel: make(chan string, 1),
	}
	start := time.Now()
	if err := pinger.resolveName(pinger.DestinationStr); err != nil {
		pinger.Stats.TotalTime = time.Since(start)
		return nil, errors.New("Unable to resolve the name for '" + destination + "'")
	}
	pinger.Stats.ResolveTime = time.Since(start)
	return &pinger, nil
}

func (pinger *Pinger) PingAll() error {
	var producer_wg sync.WaitGroup
	producer_wg.Add(1)
	var err error
	go func(producer_wg *sync.WaitGroup, err *error) {
		defer producer_wg.Done()
		*err = startPingProducer(pinger)
		if err != nil {
			return
		}
	}(&producer_wg, &err)

	producer_wg.Wait()
	return err
}

func startPingProducer(pinger *Pinger) error {
	if pinger.Count == 0 {
		return fmt.Errorf("invalid ping count")
	}

	start := time.Now()
	pinger.logToStreamChannel(fmt.Sprintf("Started pinger producer: %v", start))
	if pinger.IsSequential {
		for iteration := 0; iteration < pinger.Count; iteration++ {
			pinger.logToStreamChannel(fmt.Sprintf("Producting %v", iteration))
			time.Sleep(time.Duration(pinger.PingDelay))
		}
	} else {
		fmt.Println("Producing parallel pings")

	}
	end := time.Since(start)
	pinger.logToStreamChannel(fmt.Sprintf("Total time taken for ping: %v", end))
	return nil
}

func startPingConsumer(pinger *Pinger) error {
	return nil
}

func (pinger *Pinger) IsPingComplete() bool {
	return atomic.LoadInt32(&pinger._is_ping_done) == 1
}

func (pinger *Pinger) logToStreamChannel(data string) {
	var mu sync.Mutex
	mu.Lock()
	pinger._log_stream_channel <- data
	mu.Unlock()
}

// need to work on this pinger channel to gracefully handle the incoming data
func (pinger *Pinger) Stream() <-chan string {
	return pinger._log_stream_channel
}

func (pinger *Pinger) SetParallelPing(parallel bool) *Pinger {
	// explicitly sets the ping to run in parallel
	pinger.IsSequential = !parallel
	return pinger
}

func (pinger *Pinger) SetPayloadSizeInBytes(payload_size int) *Pinger {
	// explicitly sets the size of the ping requests within boundary of _DEFAULT_MAX_PAYLOAD_SIZE
	// returns nil
	pinger.Payload = strings.Repeat("d", payload_size%_DEFAULT_MAX_PAYLOAD_SIZE)
	return pinger
}

func (pinger *Pinger) SetPingCount(count int) *Pinger {
	// explicitly set ping count. Checks if set below 0, then converts to absolute
	// default is usually 4 as defined in _DEFAULT_PING_COUNT
	// returns nil
	if count < 0 {
		count *= -1
	} else if count > _DEFAULT_MAX_PING_COUNT {
		pinger.Count = _DEFAULT_MAX_PING_COUNT
		return pinger
	}
	pinger.Count = count
	return pinger
}

func (pinger *Pinger) SetResolveTimeout(timeout int) *Pinger {
	// explicitly set ping delay. Checks for timeout less than 0ms
	// default is usually 5000ms as defined in _DEFAULT_RESOLVE_TIMEOUT_MS, hence sets if timeout <0
	if timeout < 0 {
		pinger.ResolveTimeout = _DEFAULT_RESOLVE_TIMEOUT_MS
	} else {
		pinger.ResolveTimeout = timeout
	}
	return pinger
}

func (pinger *Pinger) SetPingDelayInMS(delay int) *Pinger {
	// explicitly set ping delay. Checks for delay
	// default is usually 1000ms as defined in _DEFAULT_PING_DELAY_MS, hence sets if delay <=0
	if delay < 0 {
		pinger.PingDelay = _DEFAULT_PING_DELAY_MS
	} else {
		pinger.PingDelay = delay
	}
	return pinger
}

func (pinger *Pinger) SetTTL(ttl int) *Pinger {
	pinger.TTL = ttl
	return pinger
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
	var mu sync.Mutex
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
		mu.Lock()
		pinger._log_stream_channel <- "Unable to resolve for " + destination + " with " + strconv.Itoa(len(pinger.Payload)) + " bytes of data"
		mu.Unlock()
		return err
	}
	pinger.Destination = addr
	pinger.Stats.ResolveTime = time.Duration(time.Since(start).Milliseconds())
	pinger.Stats.ResolveTimedOut = false

	return nil
}
