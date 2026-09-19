package netutils

import (
	"encoding/json"
	"math"
	"slices"
	"sync"
	"time"
)

// Stats holds the results of one PingAll run. ResolveTime and TotalTime are
// ordinary time.Duration for internal/Go-facing use (formatting with %v,
// comparisons, arithmetic); see MarshalJSON for how they're represented on
// the wire.
type Stats struct {
	Packets         []ICMPPacket
	Loss            int
	Min             int
	Max             int
	Avg             float64
	StdDev          float64
	ResolveTime     time.Duration
	ResolveTimedOut bool
	TotalTime       time.Duration

	mu sync.Mutex // guards Loss; sendICMP calls addLoss concurrently in parallel ping mode
}

// MarshalJSON reports ResolveTime/TotalTime in milliseconds, matching their
// field names on the wire (resolve_time_ms/total_time_taken_ms). Plain
// json.Marshal on a time.Duration - an int64 nanosecond count with no
// special-cased encoding - would otherwise silently serialize nanoseconds
// under a field name that promises milliseconds.
func (stats *Stats) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Packets         []ICMPPacket `json:"icmp_packets"`
		Loss            int          `json:"loss"`
		Min             int          `json:"min_ms"`
		Max             int          `json:"max_ms"`
		Avg             float64      `json:"avg_ms"`
		StdDev          float64      `json:"stddev"`
		ResolveTimeMS   int64        `json:"resolve_time_ms"`
		ResolveTimedOut bool         `json:"is_resolve_timed_out"`
		TotalTimeMS     int64        `json:"total_time_taken_ms"`
	}{
		Packets:         stats.Packets,
		Loss:            stats.Loss,
		Min:             stats.Min,
		Max:             stats.Max,
		Avg:             stats.Avg,
		StdDev:          stats.StdDev,
		ResolveTimeMS:   stats.ResolveTime.Milliseconds(),
		ResolveTimedOut: stats.ResolveTimedOut,
		TotalTimeMS:     stats.TotalTime.Milliseconds(),
	})
}

// addLoss increments Loss. A plain mutex rather than an atomic field: this
// is called once per failed ping, not a hot path, and keeps Loss an
// ordinary int for JSON marshaling and direct field access from callers.
func (stats *Stats) addLoss() {
	stats.mu.Lock()
	stats.Loss++
	stats.mu.Unlock()
}

func (stats *Stats) String() string {
	str, err := json.Marshal(stats)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// MeasureStats computes Min/Max/Avg/StdDev from every successfully
// round-tripped packet. Safe to call with zero successes (e.g. total loss,
// or a resolve timeout): every field is simply left at its zero value
// instead of the pre-2.0 behavior of dividing by a zero success count and
// silently producing NaN/Inf in the JSON output.
func (pinger *Pinger) MeasureStats() *Stats {
	if pinger.Stats.ResolveTimedOut {
		return pinger.Stats
	}

	roundTrips := make([]int, 0, len(pinger.Stats.Packets))
	sum := 0
	for _, packet := range pinger.Stats.Packets {
		if packet.ErrorEncountered {
			continue
		}
		rt := int(packet.ReceiveDateTimeUNIX - packet.SentDateTimeUNIX)
		roundTrips = append(roundTrips, rt)
		sum += rt
	}
	if len(roundTrips) == 0 {
		return pinger.Stats
	}

	pinger.Stats.Avg = float64(sum) / float64(len(roundTrips))
	pinger.Stats.Max = slices.Max(roundTrips)
	pinger.Stats.Min = slices.Min(roundTrips)

	var variance float64
	for _, rt := range roundTrips {
		d := float64(rt) - pinger.Stats.Avg
		variance += d * d
	}
	pinger.Stats.StdDev = math.Sqrt(variance / float64(len(roundTrips)))

	return pinger.Stats
}
