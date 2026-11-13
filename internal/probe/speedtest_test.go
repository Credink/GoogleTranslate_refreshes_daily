package probe

import (
	"context"
	"testing"
	"time"
)

func TestOrderCandidatesKeepsPriority(t *testing.T) {
	ips := []string{"10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4"}
	ordered := orderCandidates(ips, 1)
	if ordered[0] != "10.0.0.1" {
		t.Fatalf("priority IP moved: %v", ordered)
	}
	if len(ordered) != 4 {
		t.Fatalf("len=%d", len(ordered))
	}
}

func TestSpeedtestEmpty(t *testing.T) {
	r, err := Speedtest(context.Background(), Config{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Available) != 0 || r.Failed != 0 {
		t.Fatalf("unexpected %+v", r)
	}
}

func TestSpeedtestEarlyExitStops(t *testing.T) {
	// Unreachable RFC5737 / documentation addresses: all fail fast under short timeout.
	ips := []string{"192.0.2.1", "198.51.100.1", "203.0.113.1"}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	start := time.Now()
	_, _ = Speedtest(ctx, Config{
		Timeout:          200 * time.Millisecond,
		Repeat:           1,
		Workers:          3,
		StopAfterSuccess: 1,
		PriorityKeep:     1,
	}, ips)
	if time.Since(start) > 2500*time.Millisecond {
		t.Fatalf("pool scan too slow on failing IPs: %v", time.Since(start))
	}
}
