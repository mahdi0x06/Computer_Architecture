package metrics

import "testing"

func TestStatsRatesAreZeroForEmptyCounters(t *testing.T) {
	stats := Stats{}

	tests := map[string]float64{
		"L1 hit rate":                stats.L1HitRate(),
		"L1 miss rate":               stats.L1MissRate(),
		"Victim hit rate":            stats.VictimHitRate(),
		"Victim miss rate":           stats.VictimMissRate(),
		"L2 hit rate":                stats.L2HitRate(),
		"L2 miss rate":               stats.L2MissRate(),
		"average cycles per request": stats.AverageCyclesPerRequest(),
	}

	for name, got := range tests {
		if got != 0 {
			t.Errorf("%s = %v, want 0", name, got)
		}
	}
}

func TestStatsRates(t *testing.T) {
	stats := Stats{
		TotalRequests: 10,
		L1Hits:        4,
		L1Misses:      6,
		VictimHits:    3,
		VictimMisses:  3,
		L2Hits:        1,
		L2Misses:      2,
		TotalCycles:   250,
	}

	tests := []struct {
		name string
		got  float64
		want float64
	}{
		{name: "L1 hit rate", got: stats.L1HitRate(), want: 0.4},
		{name: "L1 miss rate", got: stats.L1MissRate(), want: 0.6},
		{name: "Victim hit rate", got: stats.VictimHitRate(), want: 0.5},
		{name: "Victim miss rate", got: stats.VictimMissRate(), want: 0.5},
		{name: "L2 hit rate", got: stats.L2HitRate(), want: 1.0 / 3.0},
		{name: "L2 miss rate", got: stats.L2MissRate(), want: 2.0 / 3.0},
		{name: "average cycles per request", got: stats.AverageCyclesPerRequest(), want: 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
}
