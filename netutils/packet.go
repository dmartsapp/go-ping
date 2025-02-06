package netutils

import (
	"net"
	"runtime"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type ICMPPacket struct {
	Destination         net.IPAddr `json:"destination"`
	PayloadSize         int        `json:"payload_size"`
	Sequence            int        `json:"sequence_number"`
	SentDateTimeUNIX    int64      `json:"sent_datetime_unix_ms"`
	ReceiveDateTimeUNIX int64      `json:"receive_datetime_unix_ms"`
	ErrorEncountered    bool       `json:"is_error_encountered"`
	ErrorStr            string     `json:"error_string"`
}

func (pinger *Pinger) sendICMP(destination net.IP, seq int) {
	var mu sync.Mutex
	time.Sleep(time.Millisecond * time.Duration(pinger.PingDelay))
	icmppacket := ICMPPacket{
		Destination: net.IPAddr{
			IP: destination,
		},
		Sequence:         seq,
		PayloadSize:      len(pinger.Payload),
		SentDateTimeUNIX: time.Now().UnixMilli(),
	}
	var icmpconn *icmp.PacketConn
	var err error

	// Start listening for icmp replies
	if runtime.GOOS == "windows" {
		if icmpconn, err = icmp.ListenPacket("ip4:icmp", _DEFAULT_LISTEN_ADDRESS); err != nil {
			icmppacket.ErrorEncountered = true
			pinger.Stats.Loss += 1
			icmppacket.ErrorStr = err.Error()
			mu.Lock()
			pinger._pinger_channel <- icmppacket
			mu.Unlock()
			return
		}
		defer icmpconn.Close()
	} else {
		if icmpconn, err = icmp.ListenPacket("udp4", _DEFAULT_LISTEN_ADDRESS); err != nil {
			icmppacket.ErrorEncountered = true
			pinger.Stats.Loss += 1
			icmppacket.ErrorStr = err.Error()
			mu.Lock()
			pinger._pinger_channel <- icmppacket
			mu.Unlock()
			return
		}
		defer icmpconn.Close()
	}
	// Make a new ICMP message
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID:   seq & 0xffff,
			Seq:  seq,                    //<< uint(seq), // TODO
			Data: []byte(pinger.Payload), // 4 bytes per char
		},
	}
	msg_bytes, err := msg.Marshal(nil)
	if err != nil {
		icmppacket.ErrorEncountered = true
		pinger.Stats.Loss += 1
		icmppacket.ErrorStr = err.Error()
		mu.Lock()
		pinger._pinger_channel <- icmppacket
		mu.Unlock()
		return
	}
	// _stream_channel <- "Sending request #" + strconv.Itoa(seq) + " to " + destination.String() + " with " + strconv.Itoa(len(pinger.Payload)) + " bytes of data"
	if runtime.GOOS == "windows" {
		_, err := icmpconn.WriteTo(msg_bytes, &net.IPAddr{IP: destination})
		if err != nil {
			icmppacket.ErrorEncountered = true
			pinger.Stats.Loss += 1
			icmppacket.ErrorStr = err.Error()
			mu.Lock()
			pinger._stream_channel <- "Error encountered for request #" + strconv.Itoa(seq) + " to " + destination.String() + " with " + strconv.Itoa(len(pinger.Payload)) + " bytes of data"
			pinger._pinger_channel <- icmppacket
			mu.Unlock()
			return
		}
	} else {
		_, err = icmpconn.WriteTo(msg_bytes, &net.UDPAddr{IP: destination})
		icmppacket.SentDateTimeUNIX = time.Now().UnixMilli()
		if err != nil {
			icmppacket.ErrorEncountered = true
			pinger.Stats.Loss += 1
			icmppacket.ErrorStr = err.Error()
			mu.Lock()
			pinger._stream_channel <- "Error encountered for request #" + strconv.Itoa(seq) + " to " + destination.String() + " with " + strconv.Itoa(len(pinger.Payload)) + " bytes of data"
			pinger._pinger_channel <- icmppacket
			mu.Unlock()
			return
		}
	}

	for {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			break
		}
		// Wait for a reply
		reply := make([]byte, _DEFAULT_MTU)
		err = icmpconn.SetReadDeadline(time.Now().Add(time.Duration(pinger.TTL) * time.Millisecond))
		if err != nil {
			icmppacket.ErrorEncountered = true
			pinger.Stats.Loss += 1
			icmppacket.ErrorStr = err.Error()
			mu.Lock()
			pinger._stream_channel <- "Error encountered for request #" + strconv.Itoa(seq) + " to " + destination.String() + " with " + strconv.Itoa(len(pinger.Payload)) + " bytes of data"
			pinger._pinger_channel <- icmppacket
			mu.Unlock()
			return
		}
		n, _, err := icmpconn.ReadFrom(reply)
		if err != nil {
			icmppacket.ErrorEncountered = true
			pinger.Stats.Loss += 1
			icmppacket.ErrorStr = err.Error()
			mu.Lock()
			pinger._stream_channel <- "Error encountered for request #" + strconv.Itoa(seq) + " to " + destination.String() + " with " + strconv.Itoa(len(pinger.Payload)) + " bytes of data"
			pinger._pinger_channel <- icmppacket
			mu.Unlock()
			return
		}

		rm, err := icmp.ParseMessage(1, reply[:n])
		if err != nil {
			icmppacket.ErrorEncountered = true
			pinger.Stats.Loss += 1
			icmppacket.ErrorStr = err.Error()
			mu.Lock()
			pinger._stream_channel <- "Error encountered for request #" + strconv.Itoa(seq) + " to " + destination.String() + " with " + strconv.Itoa(len(pinger.Payload)) + " bytes of data"
			pinger._pinger_channel <- icmppacket
			mu.Unlock()
			return
		}
		switch rm.Type {
		case ipv4.ICMPTypeEchoReply:
			body, _ := rm.Body.Marshal(ipv4.ICMPTypeEchoReply.Protocol())

			if int(body[3]) == seq {
				icmppacket.ReceiveDateTimeUNIX = time.Now().UnixMilli()
				mu.Lock()
				// _stream_channel <- time.Now().Local().Format("12/12/2014 18:23:21") + ": Received response for request #" + strconv.Itoa(seq) + " from " + destination.String() + " with " + strconv.Itoa(icmppacket.PayloadSize) + " bytes of data"
				pinger._stream_channel <- "Received response for request #" + strconv.Itoa(seq) + " from " + destination.String() + " with " + strconv.Itoa(icmppacket.PayloadSize) + " bytes of data in " + strconv.FormatFloat(float64(icmppacket.ReceiveDateTimeUNIX-icmppacket.SentDateTimeUNIX)/1, 'f', 0, 64) + "ms"
				pinger._pinger_channel <- icmppacket
				mu.Unlock()
				return
			} else { // sequence mismatch, look for another packet to match
				continue
			}

			// default:
			// 	return dst, 0, fmt.Errorf("%v %+v", peer, rm.Type)
		}
	}
}
