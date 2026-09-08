package modgearman

import (
	"strings"
	"testing"
)

func TestCheckLoads(t *testing.T) {
	disableLogging()
	t.Cleanup(func() { setLogLevel(0) })

	tests := []struct {
		name       string
		limits     [3]float64
		wantOK     bool
		wantReason string
	}{
		{name: "limits disabled", wantOK: true},
		{name: "below all limits", limits: [3]float64{1.1, 5.1, 15.1}, wantOK: true},
		{name: "one minute limit exceeded", limits: [3]float64{0.9, 5.1, 15.1}, wantReason: "load1"},
		{name: "five minute limit exceeded", limits: [3]float64{1.1, 4.9, 15.1}, wantReason: "load5"},
		{name: "fifteen minute limit exceeded", limits: [3]float64{1.1, 5.1, 14.9}, wantReason: "load15"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := config{
				loadLimit1:  test.limits[0],
				loadLimit5:  test.limits[1],
				loadLimit15: test.limits[2],
			}
			mainworker := &mainWorker{cfg: &cfg, min1: 1, min5: 5, min15: 15}

			ok, reason := mainworker.checkLoads()
			if ok != test.wantOK {
				t.Fatalf("checkLoads() ok = %v, want %v (reason: %q)", ok, test.wantOK, reason)
			}
			if !strings.Contains(reason, test.wantReason) {
				t.Errorf("checkLoads() reason = %q, want it to contain %q", reason, test.wantReason)
			}
		})
	}
}

func TestCheckMemory(t *testing.T) {
	disableLogging()
	t.Cleanup(func() { setLogLevel(0) })

	tests := []struct {
		name     string
		limit    uint64
		total    uint64
		free     uint64
		wantOK   bool
		wantUsed string
	}{
		{name: "limit disabled", total: 100, free: 0, wantOK: true},
		{name: "memory information unavailable", limit: 70, wantOK: true},
		{name: "below limit", limit: 70, total: 100, free: 31, wantOK: true},
		{name: "at limit", limit: 70, total: 100, free: 30, wantUsed: "70%"},
		{name: "above limit", limit: 70, total: 100, free: 20, wantUsed: "80%"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := config{memLimit: test.limit}
			mainworker := &mainWorker{cfg: &cfg, memTotal: test.total, memFree: test.free}

			ok, reason := mainworker.checkMemory()
			if ok != test.wantOK {
				t.Fatalf("checkMemory() ok = %v, want %v (reason: %q)", ok, test.wantOK, reason)
			}
			if !strings.Contains(reason, test.wantUsed) {
				t.Errorf("checkMemory() reason = %q, want it to contain %q", reason, test.wantUsed)
			}
		})
	}
}
