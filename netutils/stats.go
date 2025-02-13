package netutils

import (
	"encoding/json"
	"log"
	"math"
	"slices"
	"time"
)

type Stats struct {
	Packets         []ICMPPacket  `json:"icmp_packets"`
	Loss            int           `json:"loss"`
	Min             int           `json:"min_ms"`
	Max             int           `json:"max_ms"`
	Avg             float64       `json:"avg_ms"`
	StdDev          float64       `json:"stddev"`
	ResolveTime     time.Duration `json:"resolve_time_ms"`
	ResolveTimedOut bool          `json:"is_resolve_timed_out"`
	TotalTime       time.Duration `json:"total_time_taken_ms"`
}

func (stats *Stats) String() string {
	// returns json representation of the pinger object
	if str, err := json.Marshal(stats); err != nil {
		log.Fatal(err)
		return err.Error()
	} else {
		return string(str)
	}
}

func (pinger *Pinger) MeasureStats() *Stats {
	// fmt.Println("Calculate the stats, now that pingers have stopped sending packets")
	if pinger.Stats.ResolveTimedOut {
		return pinger.Stats
	}
	timetaken := make([]int, 0)
	success_counter := 0
	sum := 0
	for _, packet := range pinger.Stats.Packets {
		if packet.ErrorEncountered {
			continue
		}
		success_counter += 1
		sum += int(packet.ReceiveDateTimeUNIX - packet.SentDateTimeUNIX)
		timetaken = append(timetaken, int(packet.ReceiveDateTimeUNIX-packet.SentDateTimeUNIX))
		// fmt.Println(int(packet.ReceiveDateTimeUNIX - packet.SentDateTimeUNIX))
	}
	pinger.Stats.Avg = float64(sum) / float64(success_counter)
	if len(timetaken) > 0 {
		pinger.Stats.Max = slices.Max(timetaken)
		pinger.Stats.Min = slices.Min(timetaken)
	}

	success_counter = 0
	for _, packet := range pinger.Stats.Packets {
		if packet.ErrorEncountered {
			continue
		}
		success_counter += 1
		pinger.Stats.StdDev += (float64(packet.ReceiveDateTimeUNIX-packet.SentDateTimeUNIX) - pinger.Stats.Avg) * (float64(packet.ReceiveDateTimeUNIX-packet.SentDateTimeUNIX) - pinger.Stats.Avg)
	}
	// fmt.Println(success_counter)
	// fmt.Println(pinger.Stats.StdDev)
	pinger.Stats.StdDev = math.Sqrt(pinger.Stats.StdDev / float64(success_counter))

	return pinger.Stats
}
