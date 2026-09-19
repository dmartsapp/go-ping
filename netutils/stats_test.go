package netutils

import (
	"encoding/json"
	"testing"
	"time"
)

func packetAt(sentMS, recvMS int64, errored bool) ICMPPacket {
	return ICMPPacket{
		SentDateTimeUNIX:    sentMS,
		ReceiveDateTimeUNIX: recvMS,
		ErrorEncountered:    errored,
	}
}

func TestMeasureStatsAllSuccess(t *testing.T) {
	pinger := &Pinger{Stats: &Stats{Packets: []ICMPPacket{
		packetAt(0, 10, false),
		packetAt(0, 20, false),
		packetAt(0, 30, false),
	}}}
	stats := pinger.MeasureStats()
	if stats.Min != 10 || stats.Max != 30 {
		t.Errorf("Min/Max = %d/%d, want 10/30", stats.Min, stats.Max)
	}
	if stats.Avg != 20 {
		t.Errorf("Avg = %v, want 20", stats.Avg)
	}
	if stats.StdDev == 0 {
		t.Error("StdDev = 0, want > 0 for varying round trips")
	}
}

func TestMeasureStatsIgnoresErroredPackets(t *testing.T) {
	pinger := &Pinger{Stats: &Stats{Packets: []ICMPPacket{
		packetAt(0, 10, false),
		packetAt(0, 9999, true), // errored: must not skew Min/Max/Avg
	}}}
	stats := pinger.MeasureStats()
	if stats.Min != 10 || stats.Max != 10 || stats.Avg != 10 {
		t.Errorf("Min/Max/Avg = %d/%d/%v, want 10/10/10 (errored packet must be excluded)", stats.Min, stats.Max, stats.Avg)
	}
}

func TestMeasureStatsZeroSuccessesDoesNotProduceNaN(t *testing.T) {
	// Regression test: the pre-2.0 implementation divided by a zero success
	// count here, producing NaN/Inf that then serialized as invalid JSON
	// numbers. Every packet failed, so every derived field should just stay
	// at its zero value instead.
	pinger := &Pinger{Stats: &Stats{Packets: []ICMPPacket{
		packetAt(0, 10, true),
		packetAt(0, 20, true),
	}}}
	stats := pinger.MeasureStats()
	if stats.Avg != 0 || stats.StdDev != 0 || stats.Min != 0 || stats.Max != 0 {
		t.Errorf("expected all-zero stats for zero successes, got Min=%d Max=%d Avg=%v StdDev=%v", stats.Min, stats.Max, stats.Avg, stats.StdDev)
	}

	js, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(js, &decoded); err != nil {
		t.Fatalf("zero-success Stats did not produce valid JSON: %v\nraw: %s", err, js)
	}
}

func TestMeasureStatsNoPackets(t *testing.T) {
	pinger := &Pinger{Stats: &Stats{}}
	stats := pinger.MeasureStats()
	if stats.Min != 0 || stats.Max != 0 || stats.Avg != 0 {
		t.Errorf("expected zero-value stats for no packets at all, got Min=%d Max=%d Avg=%v", stats.Min, stats.Max, stats.Avg)
	}
}

func TestMeasureStatsResolveTimedOutShortCircuits(t *testing.T) {
	pinger := &Pinger{Stats: &Stats{
		ResolveTimedOut: true,
		Packets:         []ICMPPacket{packetAt(0, 10, false)},
	}}
	stats := pinger.MeasureStats()
	if stats.Min != 0 || stats.Max != 0 {
		t.Error("MeasureStats should not compute anything when ResolveTimedOut is true")
	}
}

func TestStatsAddLossIsConcurrencySafe(t *testing.T) {
	stats := &Stats{}
	const n = 100
	done := make(chan struct{})
	for i := 0; i < n; i++ {
		go func() {
			stats.addLoss()
			done <- struct{}{}
		}()
	}
	for i := 0; i < n; i++ {
		<-done
	}
	if stats.Loss != n {
		t.Errorf("Loss = %d, want %d", stats.Loss, n)
	}
}

func TestStatsMarshalJSONUsesMilliseconds(t *testing.T) {
	// Regression test: time.Duration marshals as raw nanoseconds by
	// default, which previously ended up under a field name (resolve_time_ms)
	// that promised milliseconds.
	stats := &Stats{
		ResolveTime: 7 * time.Millisecond,
		TotalTime:   3632 * time.Millisecond,
	}
	js, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(js, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if got := decoded["resolve_time_ms"]; got != float64(7) {
		t.Errorf("resolve_time_ms = %v, want 7", got)
	}
	if got := decoded["total_time_taken_ms"]; got != float64(3632) {
		t.Errorf("total_time_taken_ms = %v, want 3632", got)
	}
}
