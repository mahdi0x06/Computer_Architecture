package testbench

import (
	"bytes"
	"encoding/csv"
	"testing"

	"victimcacheproject/internal/benchmark"
	"victimcacheproject/internal/config"
)

func testSuiteConfig(base config.Config) benchmark.SuiteConfig {
	return benchmark.SuiteConfig{
		BlockSizeBytes:  base.BlockSizeBytes,
		L1SizeBytes:     base.L1SizeBytes,
		L2SizeBytes:     base.L2SizeBytes,
		L2Associativity: base.L2Associativity,
		VictimEntries:   base.VictimEntries,
		NumberOfBlocks:  4,
		Repetitions:     8,
		SequentialWords: 32,
		WordSizeBytes:   4,
		AccessSizeBytes: 8,
	}
}

func TestApplicationBenchmarksUseThreeComparisonArchitecturesAndCSVRows(t *testing.T) {
	base := config.Default()
	suiteCfg := testSuiteConfig(base)
	suiteCfg.MatrixDimension = 3
	suiteCfg.MergeSortLength = 8
	architectures := ComparisonArchitectures()
	wantArchitectures := []string{"l1-l2", "full-fifo", "full-lru"}
	if len(architectures) != len(wantArchitectures) {
		t.Fatalf("architectures=%d, want %d", len(architectures), len(wantArchitectures))
	}
	for index, want := range wantArchitectures {
		if architectures[index].Name != want {
			t.Fatalf("architecture %d=%q, want %q", index, architectures[index].Name, want)
		}
	}

	for _, kind := range []benchmark.TraceKind{benchmark.TraceMatrixMultiply, benchmark.TraceMergeSort} {
		t.Run(string(kind), func(t *testing.T) {
			scenario, err := benchmark.GenerateScenario(kind, suiteCfg)
			if err != nil {
				t.Fatal(err)
			}
			results, err := RunSuite(base, []benchmark.Scenario{scenario}, architectures)
			if err != nil {
				t.Fatal(err)
			}
			if len(results) != 3 {
				t.Fatalf("results=%d, want 3", len(results))
			}
			for _, check := range ValidateResults(results) {
				if !check.Passed {
					t.Fatalf("failed check %q: %s", check.Name, check.Detail)
				}
			}

			var output bytes.Buffer
			if err := WriteCSV(&output, results); err != nil {
				t.Fatal(err)
			}
			rows, err := csv.NewReader(&output).ReadAll()
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != 4 {
				t.Fatalf("CSV rows=%d, want one header plus three data rows", len(rows))
			}
			for index, want := range wantArchitectures {
				if rows[index+1][2] != want {
					t.Fatalf("CSV architecture row %d=%q, want %q", index, rows[index+1][2], want)
				}
			}
		})
	}
}

func TestCompleteSuitePassesAllChecks(t *testing.T) {
	base := config.Default()
	scenarios, err := benchmark.GenerateSuite(testSuiteConfig(base))
	if err != nil {
		t.Fatal(err)
	}
	architectures, err := Architectures(PolicyBoth)
	if err != nil {
		t.Fatal(err)
	}
	results, err := RunSuite(base, scenarios, architectures)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != len(scenarios)*len(architectures) {
		t.Fatalf("results=%d, want %d", len(results), len(scenarios)*len(architectures))
	}
	for _, check := range ValidateResults(results) {
		if !check.Passed {
			t.Errorf("failed check %q: %s", check.Name, check.Detail)
		}
	}
}

func TestSequentialWordTraceHasPredictablePerTopologyCounters(t *testing.T) {
	base := config.Default()
	scenario, err := benchmark.GenerateScenario(benchmark.TraceSequential, testSuiteConfig(base))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		architecture Architecture
		wantL1Hits   uint64
		wantL1Miss   uint64
		wantVCMiss   uint64
		wantL2Miss   uint64
		wantMemory   uint64
		wantCycles   uint64
	}{
		{Architecture{Name: "memory", Topology: config.TopologyMemoryOnly}, 0, 0, 0, 0, 32, 3200},
		{Architecture{Name: "l1", Topology: config.TopologyL1}, 30, 2, 0, 0, 2, 232},
		{Architecture{Name: "l1-l2", Topology: config.TopologyL1L2}, 30, 2, 0, 2, 2, 256},
		{Architecture{Name: "full-fifo", Topology: config.TopologyFull, VictimEnabled: true, VictimPolicy: config.ReplacementFIFO}, 30, 2, 2, 2, 2, 260},
		{Architecture{Name: "full-lru", Topology: config.TopologyFull, VictimEnabled: true, VictimPolicy: config.ReplacementLRU}, 30, 2, 2, 2, 2, 260},
	}

	for _, tt := range tests {
		t.Run(tt.architecture.Name, func(t *testing.T) {
			result, err := RunCase(base, scenario, tt.architecture)
			if err != nil {
				t.Fatal(err)
			}
			got := result.Stats
			if got.TotalRequests != 32 ||
				got.L1Hits != tt.wantL1Hits || got.L1Misses != tt.wantL1Miss ||
				got.VictimHits != 0 || got.VictimMisses != tt.wantVCMiss ||
				got.L2Hits != 0 || got.L2Misses != tt.wantL2Miss ||
				got.MemoryAccesses != tt.wantMemory || got.TotalCycles != tt.wantCycles {
				t.Fatalf("unexpected sequential counters: requests=%d L1=%d/%d VC=%d/%d L2=%d/%d MEM=%d cycles=%d",
					got.TotalRequests, got.L1Hits, got.L1Misses, got.VictimHits, got.VictimMisses,
					got.L2Hits, got.L2Misses, got.MemoryAccesses, got.TotalCycles)
			}
		})
	}
}

func TestMixedTraceHasPredictablePerTopologyCounters(t *testing.T) {
	base := config.Default()
	scenario, err := benchmark.GenerateScenario(benchmark.TraceMixed, testSuiteConfig(base))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		architecture Architecture
		wantL1Hits   uint64
		wantL1Miss   uint64
		wantVCHits   uint64
		wantVCMiss   uint64
		wantL2Hits   uint64
		wantL2Miss   uint64
		wantMemory   uint64
		wantCycles   uint64
	}{
		{Architecture{Name: "memory", Topology: config.TopologyMemoryOnly}, 0, 0, 0, 0, 0, 0, 1312, 131200},
		{Architecture{Name: "l1", Topology: config.TopologyL1}, 667, 645, 0, 0, 0, 0, 645, 65812},
		{Architecture{Name: "l1-l2", Topology: config.TopologyL1L2}, 667, 645, 0, 0, 622, 23, 23, 11352},
		{Architecture{Name: "full-fifo", Topology: config.TopologyFull, VictimEnabled: true, VictimPolicy: config.ReplacementFIFO}, 667, 645, 60, 585, 562, 23, 23, 11922},
		{Architecture{Name: "full-lru", Topology: config.TopologyFull, VictimEnabled: true, VictimPolicy: config.ReplacementLRU}, 667, 645, 188, 457, 434, 23, 23, 10386},
	}

	for _, tt := range tests {
		t.Run(tt.architecture.Name, func(t *testing.T) {
			result, err := RunCase(base, scenario, tt.architecture)
			if err != nil {
				t.Fatal(err)
			}
			got := result.Stats
			if got.TotalRequests != 1312 ||
				got.L1Hits != tt.wantL1Hits || got.L1Misses != tt.wantL1Miss ||
				got.VictimHits != tt.wantVCHits || got.VictimMisses != tt.wantVCMiss ||
				got.L2Hits != tt.wantL2Hits || got.L2Misses != tt.wantL2Miss ||
				got.MemoryAccesses != tt.wantMemory || got.TotalCycles != tt.wantCycles {
				t.Fatalf("unexpected mixed counters: requests=%d L1=%d/%d VC=%d/%d L2=%d/%d MEM=%d cycles=%d",
					got.TotalRequests, got.L1Hits, got.L1Misses, got.VictimHits, got.VictimMisses,
					got.L2Hits, got.L2Misses, got.MemoryAccesses, got.TotalCycles)
			}
		})
	}
}

func TestMixedTraceMakesLRUOutperformFIFO(t *testing.T) {
	base := config.Default()
	scenario, err := benchmark.GenerateScenario(benchmark.TraceMixed, testSuiteConfig(base))
	if err != nil {
		t.Fatal(err)
	}
	fifo, err := RunCase(base, scenario, Architecture{Name: "full-fifo", Topology: config.TopologyFull, VictimEnabled: true, VictimPolicy: config.ReplacementFIFO})
	if err != nil {
		t.Fatal(err)
	}
	lru, err := RunCase(base, scenario, Architecture{Name: "full-lru", Topology: config.TopologyFull, VictimEnabled: true, VictimPolicy: config.ReplacementLRU})
	if err != nil {
		t.Fatal(err)
	}
	if lru.Stats.VictimHits <= fifo.Stats.VictimHits {
		t.Fatalf("LRU victim hits=%d, FIFO=%d", lru.Stats.VictimHits, fifo.Stats.VictimHits)
	}
	if lru.Stats.L2ReadRequests >= fifo.Stats.L2ReadRequests {
		t.Fatalf("LRU L2 reads=%d, FIFO=%d", lru.Stats.L2ReadRequests, fifo.Stats.L2ReadRequests)
	}
	if lru.Stats.TotalCycles >= fifo.Stats.TotalCycles {
		t.Fatalf("LRU cycles=%d, FIFO=%d", lru.Stats.TotalCycles, fifo.Stats.TotalCycles)
	}
}

func TestConflictVictimImprovesBaseline(t *testing.T) {
	base := config.Default()
	scenario, err := benchmark.GenerateScenario(benchmark.TraceConflict, testSuiteConfig(base))
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := RunCase(base, scenario, Architecture{Name: "l1-l2", Topology: config.TopologyL1L2})
	if err != nil {
		t.Fatal(err)
	}
	full, err := RunCase(base, scenario, Architecture{Name: "full-fifo", Topology: config.TopologyFull, VictimEnabled: true, VictimPolicy: config.ReplacementFIFO})
	if err != nil {
		t.Fatal(err)
	}
	if full.Stats.VictimHits == 0 {
		t.Fatal("expected victim hits")
	}
	if full.Stats.L2ReadRequests >= baseline.Stats.L2ReadRequests {
		t.Fatalf("L2 reads full=%d baseline=%d", full.Stats.L2ReadRequests, baseline.Stats.L2ReadRequests)
	}
	if full.Stats.TotalCycles >= baseline.Stats.TotalCycles {
		t.Fatalf("cycles full=%d baseline=%d", full.Stats.TotalCycles, baseline.Stats.TotalCycles)
	}
}
