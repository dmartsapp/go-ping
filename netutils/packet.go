package netutils

import (
	"net"
	"strconv"
	"time"

	"golang.org/x/net/icmp"
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

// fail records a failed/lost packet and hands it to the packet channel. It's
// the one place sendICMP gives up on a request, so every "record a loss and
// stop" path below reduces to a single call instead of five near-duplicates.
func (pinger *Pinger) fail(packet ICMPPacket, err error) {
	packet.ErrorEncountered = true
	packet.ErrorStr = err.Error()
	pinger.Stats.addLoss()
	pinger.packetCh <- packet
}

// sendICMP sends one ICMP echo request to destination and waits for its
// reply (or the pinger's reply timeout), dispatching to ICMPv4 or ICMPv6
// automatically based on the destination's own address family - the two
// protocols agree closely enough at the golang.org/x/net/icmp level that a
// single implementation, parameterized by *icmpFamily, covers both.
func (pinger *Pinger) sendICMP(destination net.IP, seq int) {
	family := familyFor(destination)
	packet := ICMPPacket{
		Destination: net.IPAddr{IP: destination},
		Sequence:    seq,
		PayloadSize: len(pinger.Payload),
	}

	conn, err := icmp.ListenPacket(family.network(), family.listenAddr)
	if err != nil {
		packet.SentDateTimeUNIX = time.Now().UnixMilli()
		pinger.fail(packet, err)
		return
	}
	defer func() { _ = conn.Close() }()

	msg := icmp.Message{
		Type: family.echoRequestType,
		Code: 0,
		Body: &icmp.Echo{
			ID:   seq & 0xffff,
			Seq:  seq,
			Data: pinger.Payload,
		},
	}
	msgBytes, err := msg.Marshal(nil)
	if err != nil {
		packet.SentDateTimeUNIX = time.Now().UnixMilli()
		pinger.fail(packet, err)
		return
	}

	packet.SentDateTimeUNIX = time.Now().UnixMilli()
	if _, err = conn.WriteTo(msgBytes, family.writeAddr(destination)); err != nil {
		pinger.logToStreamChannel("error sending request #" + strconv.Itoa(seq) + " to " + destination.String() + ": " + err.Error())
		pinger.fail(packet, err)
		return
	}

	buf := make([]byte, pinger.MTU)
	for {
		if err := conn.SetReadDeadline(time.Now().Add(time.Duration(pinger.ReplyTimeoutMS) * time.Millisecond)); err != nil {
			pinger.fail(packet, err)
			return
		}

		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			pinger.logToStreamChannel("no reply for request #" + strconv.Itoa(seq) + " from " + destination.String() + ": " + err.Error())
			pinger.fail(packet, err)
			return
		}

		reply, err := icmp.ParseMessage(family.protocolNumber, buf[:n])
		if err != nil {
			pinger.fail(packet, err)
			return
		}
		if reply.Type != family.echoReplyType {
			continue // some other ICMP traffic on the same socket; keep waiting
		}
		echo, ok := reply.Body.(*icmp.Echo)
		if !ok || echo.Seq != seq {
			continue // reply to a different request sharing this socket's address; keep waiting
		}

		packet.ReceiveDateTimeUNIX = time.Now().UnixMilli()
		pinger.logToStreamChannel("received reply for request #" + strconv.Itoa(seq) + " from " + destination.String() + " (" + family.name + ") in " + strconv.Itoa(int(packet.ReceiveDateTimeUNIX-packet.SentDateTimeUNIX)) + "ms")
		pinger.packetCh <- packet
		return
	}
}
