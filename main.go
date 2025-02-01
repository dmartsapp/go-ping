package main

import (
	"github.com/farhansabbir/goping/netutils"
)

var (
// version    = "1.0.0"
// ListenAddr                     = "0.0.0.0"
// sequence_received map[int]bool = make(map[int]bool, 1000)
// mutex             sync.Mutex
)

func main() {

	// pinger := netutils.NewPinger("home435nas.local").
	pinger := netutils.NewPinger("google.com").
		SetPingCount(20).
		// SetTTL(10).
		SetPayloadSizeInBytes(3).
		SetPingDelayInMS(0).
		SetParallelPing(true)
	// pinger := netutils.NewPinger("google.com")
	// pinger.SetParallelPing(true)
	// go func(pinger *netutils.Pinger) {
	// 	for data := range pinger.Stream() {
	// 		fmt.Println(data)
	// 	}
	// }(pinger)
	pinger.Ping()
	// pinger.MeasureStats()
	// fmt.Println(pinger)
	// fmt.Println(pinger)
	// fmt.Println(pinger.Stats.Max)

	// fmt.Println([]byte(strconv.Itoa(int(time.Now().UnixMicro()))))

}

// func Ping(dst *net.IPAddr, payload_size int, options ...map[string]int) (*net.IPAddr, time.Duration, error) {
// 	icmp_payload := strings.Repeat("d", 2) // 4 bytes per char
// 	var seq int
// 	var icmpconn *icmp.PacketConn
// 	var err error
// 	for _, option := range options {
// 		seq = option["seq"]
// 	}
// 	// Start listening for icmp replies
// 	if runtime.GOOS == "windows" {
// 		if icmpconn, err = icmp.ListenPacket("ip4:icmp", ListenAddr); err != nil {
// 			return nil, 0, err
// 		}
// 		defer icmpconn.Close()
// 	} else {
// 		if icmpconn, err = icmp.ListenPacket("udp4", ListenAddr); err != nil {
// 			return nil, 0, err
// 		}
// 		defer icmpconn.Close()
// 	}
// 	// Make a new ICMP message
// 	msg := icmp.Message{
// 		Type: ipv4.ICMPTypeEcho, Code: 0,
// 		Body: &icmp.Echo{
// 			ID:   seq & 0xffff,
// 			Seq:  seq,                  //<< uint(seq), // TODO
// 			Data: []byte(icmp_payload), // 4 bytes per char
// 		},
// 	}
// 	msg_bytes, err := msg.Marshal(nil)
// 	if err != nil {
// 		return dst, 0, err
// 	}
// 	// Send it
// 	start := time.Now()
// 	if runtime.GOOS == "windows" {
// 		_, err := icmpconn.WriteTo(msg_bytes, dst)
// 		if err != nil {
// 			fmt.Println(err)
// 			return dst, 0, err
// 		}
// 	} else {
// 		_, err = icmpconn.WriteTo(msg_bytes, &net.UDPAddr{IP: net.ParseIP(dst.IP.String())})
// 		if err != nil {
// 			fmt.Println(err)
// 			return dst, 0, err
// 		} else {
// 			sequence_received[seq] = false
// 		}
// 	}

// 	for {
// 		// Wait for a reply
// 		reply := make([]byte, 1500)
// 		err = icmpconn.SetReadDeadline(time.Now().Add(3 * time.Second))
// 		if err != nil {
// 			return dst, 0, err
// 		}
// 		n, peer, err := icmpconn.ReadFrom(reply)
// 		if err != nil {
// 			return dst, 0, err
// 		}

// 		rm, err := icmp.ParseMessage(1, reply[:n])
// 		if err != nil {
// 			return dst, 0, err
// 		}
// 		switch rm.Type {
// 		case ipv4.ICMPTypeEchoReply:
// 			body, _ := rm.Body.Marshal(ipv4.ICMPTypeEchoReply.Protocol())

// 			if int(body[3]) == seq {
// 				update_sequence_received(int(body[3]))
// 				return dst, time.Since(start), nil
// 			} else { // sequence mismatch, look for another packet to match
// 				continue
// 			}

// 		default:
// 			return dst, 0, fmt.Errorf("%v %+v", peer, rm.Type)
// 		}
// 	}

// }

// func update_sequence_received(seq int) {
// 	mutex.Lock()
// 	defer mutex.Unlock()
// 	sequence_received[seq] = true
// }
