# go-ping Package Documentation

This document provides a comprehensive overview of the `go-ping` package, detailing its structures, methods, and error handling mechanisms. This package facilitates sending ICMP echo requests (pings) to network hosts, enabling network diagnostics and performance monitoring.

## Package Overview

The `go-ping` package offers a flexible and configurable way to perform ICMP pings. It supports both sequential and parallel ping operations, customizable payload sizes, and detailed statistics gathering. The package is designed to be cross-platform, with platform-specific implementations for Windows and Unix-like systems.

## Data Structures

### `ICMPPacket`

This struct encapsulates data related to a single ICMP echo request and its response.

```go
type ICMPPacket struct {
    Destination         net.IPAddr `json:"destination"`         // Destination IP address.
    PayloadSize         int        `json:"payload_size"`         // Size of the ICMP payload in bytes.
    Sequence            int        `json:"sequence_number"`      // Sequence number of the ICMP packet.
    SentDateTimeUNIX    int64      `json:"sent_datetime_unix_ms"`// Unix timestamp (milliseconds) when the packet was sent.
    ReceiveDateTimeUNIX int64      `json:"receive_datetime_unix_ms"`// Unix timestamp (milliseconds) when the response was received.
    ErrorEncountered    bool       `json:"is_error_encountered"` // Indicates if an error occurred during transmission or reception.
    ErrorStr            string     `json:"error_string"`         // Error message, if any.
}

Fields:

Destination: net.IPAddr - The destination IP address of the ping.
PayloadSize: int - The size of the payload sent in the ICMP packet.
Sequence: int - The sequence number of the ICMP packet, useful for tracking packets.
SentDateTimeUNIX: int64 - Unix timestamp in milliseconds representing when the packet was sent.
ReceiveDateTimeUNIX: int64 - Unix timestamp in milliseconds representing when the response was received.
ErrorEncountered: bool - A boolean flag indicating whether an error occurred during the ping attempt.
ErrorStr: string - A string containing the error message if an error occurred.
Stats
This struct aggregates statistics gathered from multiple ICMP ping operations.

type Stats struct {
    Packets         []ICMPPacket  `json:"icmp_packets"`         // Slice of ICMPPacket structs representing each ping.
    Loss            int           `json:"loss"`                 // Number of packets lost.
    Min             int           `json:"min_ms"`                  // Minimum round-trip time (RTT) in milliseconds.
    Max             int           `json:"max_ms"`                  // Maximum RTT in milliseconds.
    Avg             float64       `json:"avg_ms"`                // Average RTT in milliseconds.
    StdDev          float64       `json:"stddev"`               // Standard deviation of RTT in milliseconds.
    ResolveTime     time.Duration `json:"resolve_time_ms"`      // Time taken to resolve the hostname to IP address(es).
    ResolveTimedOut bool          `json:"is_resolve_timed_out"` // Indicates if the hostname resolution timed out.
    TotalTime       time.Duration `json:"total_time_taken_ms"`  // Total time taken for all ping operations.
}

Fields:

Packets: []ICMPPacket - A slice containing ICMPPacket structs for each ping attempt.
Loss: int - The number of ICMP packets that were sent but did not receive a response.
Min: int - The minimum round-trip time (RTT) observed during the ping operations, in milliseconds.
Max: int - The maximum round-trip time (RTT) observed during the ping operations, in milliseconds.
Avg: float64 - The average round-trip time (RTT) calculated from all successful ping operations, in milliseconds.
StdDev: float64 - The standard deviation of the round-trip times, providing a measure of the variability in network latency.
ResolveTime: time.Duration - The time taken to resolve the hostname to an IP address.
ResolveTimedOut: bool - A boolean indicating whether the hostname resolution timed out.
TotalTime: time.Duration - The total time taken for all ping operations, including hostname resolution and sending/receiving packets.


Pinger
This is the main struct for configuring and executing ICMP ping operations.
type Pinger struct {
    DestinationStr     string   `json:"destination"`            // Destination hostname or IP address as a string.
    Destination        []net.IP `json:"destination_ip_addresses"`// Resolved IP addresses of the destination.
    TTL                int      `json:"ttl"`                    // Time-To-Live value for the IP packets.
    ResolveTimeout     int      `json:"resolve_timeout_ms"`     // Timeout for hostname resolution in milliseconds.
    Payload            string   `json:"payload_data"`           // Payload data to be sent in the ICMP packets.
    Count              int      `json:"ping_count"`             // Number of ping packets to send.
    Stats              *Stats   `json:"stats"`                  // Pointer to the Stats struct to store ping statistics.
    IsSequential       bool     `json:"is_sequential_ping"`     // Flag indicating whether to send pings sequentially.
    PingDelay          int      `json:"ping_delay_ms"`          // Delay between ping packets in milliseconds.
    RandomizePingDelay bool     `json:"is_ping_delay_random"`   // Flag indicating whether to randomize the ping delay.
    MTU                int      `json:"mtu"`                    // Maximum Transmission Unit for the ping packets.
}

Fields:

DestinationStr: string - The destination hostname or IP address to ping.
Destination: []net.IP - A slice of resolved IP addresses for the destination. A host name may resolve to multiple IP addresses.
TTL: int - The Time-To-Live (TTL) value for the IP packets. Controls how many hops the packet can traverse.
ResolveTimeout: int - The timeout in milliseconds for resolving the destination hostname to an IP address.
Payload: string - The payload data to include in the ICMP packets.
Count: int - The number of ICMP ping packets to send to each resolved IP address.
Stats: *Stats - A pointer to a Stats struct where the ping statistics will be stored.
IsSequential: bool - A boolean flag indicating whether to send ping packets sequentially (true) or in parallel (false).
PingDelay: int - The delay in milliseconds between sending consecutive ping packets.
RandomizePingDelay: bool - A boolean flag indicating whether to randomize the delay between ping packets.
MTU: int - The Maximum Transmission Unit (MTU) for the ping packets.
Methods
NewPinger(destination string) *Pinger
Creates a new Pinger instance with the specified destination.

func NewPinger(destination string) *Pinger

Parameters:

destination: string - The destination hostname or IP address to ping.
Returns:

*Pinger: A pointer to a new Pinger instance.
Usage:
pinger := NewPinger("example.com")

Returns:

error: An error if hostname resolution fails or if there are issues sending the ICMP packets. Returns nil on success.
Error Handling:

Returns an error if the destination hostname cannot be resolved.
Captures and stores any errors encountered while sending ICMP packets in the ICMPPacket struct.
Usage:

err := pinger.PingAll()
if err != nil {
    fmt.Println("Error:", err)
}

IsPingComplete() bool
Checks if all ping requests have been completed.

func (pinger *Pinger) IsPingComplete() bool

Returns:

bool: true if all ping requests are complete, false otherwise.

if pinger.IsPingComplete() {
    fmt.Println("Ping complete")
}

Returns:

*Stats: A pointer to the Stats struct containing the ping statistics.
Usage:

stats := pinger.MeasureStats()
fmt.Println("Average RTT:", stats.Avg)

Stream() <-chan string
Returns a channel for streaming ping results as strings.

func (pinger *Pinger) Stream() <-chan string
Returns:
<-chan string: A receive-only channel that yields string representations of each ping result.
Usage:
go func() {
    for result := range pinger.Stream() {
        fmt.Println(result)
    }
}()

SetParallelPing(parallel bool) *Pinger
Configures whether to send ping requests in parallel.

func (pinger *Pinger) SetParallelPing(parallel bool) *Pinger
Parameters:

parallel: bool - true to send pings in parallel, false to send them sequentially.
Returns:

*Pinger: A pointer to the Pinger instance (for chaining).
Usage:
pinger.SetParallelPing(true)

sendicmp(destination net.IP, seq int)
Sends an ICMP ping request to the specified destination.

func (pinger *Pinger) sendicmp(destination net.IP, seq int)
Parameters:

destination: net.IP - The destination IP address.
seq: int - The sequence number of the ICMP packet.
Error Handling:

Captures and stores any errors encountered while sending the ICMP packet in the ICMPPacket struct.
Error Handling
The go-ping package incorporates error handling at several levels:

Hostname Resolution: Errors during hostname resolution are captured in the resolveName method and returned by the PingAll method. The ResolveTimedOut field in the Stats struct indicates if the resolution timed out.
ICMP Packet Transmission: Errors encountered while sending ICMP packets are captured in the sendicmp method and stored in the ErrorStr field of the ICMPPacket struct. The ErrorEncountered field is set to true in such cases.
Statistics Calculation: The MeasureStats method calculates statistics based on the successful and failed ping attempts, providing insights into network performance and reliability.
Platform-Specific Implementations
The go-ping package utilizes platform-specific implementations for sending and receiving ICMP packets:

netutils/windows.go: Contains Windows-specific code for sending ICMP requests using the syscall package.
netutils/unix.go: Contains Unix-specific code for sending ICMP requests using raw sockets.
These files define the sendReq function, which prints the platform name.

Usage Example
package main

import (
    "fmt"
    "sync"
    "time"

    "github.com/farhansabbir/go-ping/netutils"
)

func main() {
    pinger := netutils.NewPinger("example.com")
    pinger.SetPingCount(4)
    pinger.SetPayloadSizeInBytes(32)
    pinger.SetParallelPing(true)
    pinger.SetResolveTimeout(1000) // 1 second
    pinger.SetPingDelayInMS(200)    // 200 milliseconds

    var wg sync.WaitGroup
    wg.Add(1)

    go func() {
        defer wg.Done()
        for result := range pinger.Stream() {
            fmt.Println(result)
        }
    }()

    err := pinger.PingAll()
    if err != nil {
        fmt.Println("Ping error:", err)
    }

    wg.Wait() // Wait for the stream to complete

    stats := pinger.MeasureStats()
    fmt.Printf("--- Ping statistics for %s ---\n", pinger.DestinationStr)
    fmt.Printf("%d packets transmitted, %d packets received, %d%% packet loss\n",
        len(stats.Packets), len(stats.Packets)-stats.Loss, stats.Loss*100/len(stats.Packets))
    fmt.Printf("round-trip min/avg/max/stddev = %d/%f/%d/%f ms\n",
        stats.Min, stats.Avg, stats.Max, stats.StdDev)
    fmt.Printf("resolve time = %v, resolve timed out = %v\n", stats.ResolveTime, stats.ResolveTimedOut)
    fmt.Printf("total time = %v\n", stats.TotalTime)
}

This example demonstrates how to create a Pinger instance, configure its parameters, send ping requests, and process the results. It showcases the use of the Stream method for real-time output and the MeasureStats method for obtaining aggregated statistics.

