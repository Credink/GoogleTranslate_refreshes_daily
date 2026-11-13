package probe

import (
	"testing"
	"time"
)

func TestBestIPPreferV4(t *testing.T) {
	r := SpeedtestResult{Available: []Result{
		{IP: "2001:db8::1", Latency: 10 * time.Millisecond},
		{IP: "1.1.1.1", Latency: 20 * time.Millisecond},
		{IP: "8.8.8.8", Latency: 15 * time.Millisecond},
	}}
	ip, lat := r.BestIPPreferV4()
	if ip != "8.8.8.8" || lat != 15*time.Millisecond {
		t.Fatalf("got %s %v", ip, lat)
	}
}

func TestBestIPPreferV4OnlyV6(t *testing.T) {
	r := SpeedtestResult{Available: []Result{
		{IP: "2001:db8::2", Latency: 5 * time.Millisecond},
	}}
	ip, _ := r.BestIPPreferV4()
	if ip != "2001:db8::2" {
		t.Fatalf("got %s", ip)
	}
}
