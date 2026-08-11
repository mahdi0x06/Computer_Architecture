package system

import (
	"testing"
	"victimcacheproject/internal/benchmark"
	"victimcacheproject/internal/config"
	"victimcacheproject/internal/model"
)

func TestDefaultSystemIsValid(t *testing.T) {
	if err := New(config.Default()).Validate(); err != nil {
		t.Fatal(err)
	}
}
func TestAllTopologiesRun(t *testing.T) {
	for _, top := range []string{config.TopologyMemoryOnly, config.TopologyL1, config.TopologyL1L2, config.TopologyFull} {
		cfg := config.Default()
		cfg.Topology = top
		s := New(cfg)
		rs := s.Run([]model.Request{{ID: 1, Address: 0, Op: model.Read, Size: 8}, {ID: 2, Address: 0, Op: model.Read, Size: 8}})
		if len(rs) != 2 {
			t.Fatalf("%s", top)
		}
		if s.Stats.TotalRequests != 2 {
			t.Fatalf("%s", top)
		}
	}
}
func TestVictimReducesLowerLevelTraffic(t *testing.T) {
	base := config.Default()
	trace := benchmark.GenerateConflictTrace(benchmark.ConflictConfig{CacheSizeBytes: base.L1SizeBytes, NumberOfBlocks: 4, Repetitions: 10, AccessSizeBytes: 8})
	no := base
	no.Topology = config.TopologyL1L2
	sn := New(no)
	sn.Run(trace)
	yes := base
	yes.Topology = config.TopologyFull
	yes.VictimEnabled = true
	sy := New(yes)
	sy.Run(trace)
	if sy.Stats.VictimHits == 0 {
		t.Fatal("expected victim hits")
	}
	if sy.Stats.L2ReadRequests >= sn.Stats.L2ReadRequests {
		t.Fatalf("victim did not reduce L2 reads: %d vs %d", sy.Stats.L2ReadRequests, sn.Stats.L2ReadRequests)
	}
	if sy.Stats.TotalCycles >= sn.Stats.TotalCycles {
		t.Fatalf("victim did not improve cycles: %d vs %d", sy.Stats.TotalCycles, sn.Stats.TotalCycles)
	}
}
func TestRepeatedAddressHitsL1(t *testing.T) {
	cfg := config.Default()
	cfg.Topology = config.TopologyL1
	s := New(cfg)
	s.Access(model.Request{ID: 1, Address: 128, Op: model.Read, Size: 8})
	r := s.Access(model.Request{ID: 2, Address: 128, Op: model.Read, Size: 8})
	if r.Location != model.HitL1 {
		t.Fatalf("got %s", r.Location)
	}
}
