package testbench

import (
	"fmt"
	"strings"

	"victimcacheproject/internal/benchmark"
	"victimcacheproject/internal/config"
	"victimcacheproject/internal/metrics"
	"victimcacheproject/internal/model"
	"victimcacheproject/internal/simadapter"
	"victimcacheproject/internal/system"
)

const PolicyBoth = "BOTH"

// Architecture describes one hierarchy configuration in the matrix.
type Architecture struct {
	Name          string
	Topology      string
	VictimEnabled bool
	VictimPolicy  string
}

// Result contains all measurements from one scenario/architecture pair.
type Result struct {
	Scenario     benchmark.Scenario
	Architecture Architecture
	Stats        metrics.Stats
	Responses    []model.Response
}

// Check is a machine-readable correctness or performance assertion.
type Check struct {
	Name   string
	Passed bool
	Detail string
}

// ComparisonArchitectures returns the three cache hierarchies used by the
// application benchmarks. The order is stable and is also the CSV row order.
func ComparisonArchitectures() []Architecture {
	return []Architecture{
		{Name: "l1-l2", Topology: config.TopologyL1L2},
		{Name: "full-fifo", Topology: config.TopologyFull, VictimEnabled: true, VictimPolicy: config.ReplacementFIFO},
		{Name: "full-lru", Topology: config.TopologyFull, VictimEnabled: true, VictimPolicy: config.ReplacementLRU},
	}
}

func Architectures(policy string) ([]Architecture, error) {
	normalized := strings.ToUpper(strings.TrimSpace(policy))
	architectures := []Architecture{
		{Name: "memory", Topology: config.TopologyMemoryOnly},
		{Name: "l1", Topology: config.TopologyL1},
		{Name: "l1-l2", Topology: config.TopologyL1L2},
	}

	switch normalized {
	case config.ReplacementFIFO, config.ReplacementLRU:
		architectures = append(architectures, Architecture{
			Name:          "full-" + strings.ToLower(normalized),
			Topology:      config.TopologyFull,
			VictimEnabled: true,
			VictimPolicy:  normalized,
		})
	case PolicyBoth:
		architectures = append(architectures,
			Architecture{Name: "full-fifo", Topology: config.TopologyFull, VictimEnabled: true, VictimPolicy: config.ReplacementFIFO},
			Architecture{Name: "full-lru", Topology: config.TopologyFull, VictimEnabled: true, VictimPolicy: config.ReplacementLRU},
		)
	default:
		return nil, fmt.Errorf("unsupported victim policy %q; expected FIFO, LRU, or BOTH", policy)
	}
	return architectures, nil
}

func RunSuite(base config.Config, scenarios []benchmark.Scenario, architectures []Architecture) ([]Result, error) {
	results := make([]Result, 0, len(scenarios)*len(architectures))
	for _, scenario := range scenarios {
		for _, architecture := range architectures {
			result, err := RunCase(base, scenario, architecture)
			if err != nil {
				return nil, err
			}
			results = append(results, result)
		}
	}
	return results, nil
}

func RunCase(base config.Config, scenario benchmark.Scenario, architecture Architecture) (Result, error) {
	cfg := base
	cfg.Topology = architecture.Topology
	cfg.VictimEnabled = architecture.VictimEnabled
	if architecture.VictimPolicy != "" {
		cfg.VictimPolicy = architecture.VictimPolicy
	}

	simulator := system.New(cfg)
	if err := simulator.Validate(); err != nil {
		return Result{}, fmt.Errorf("%s/%s: %w", scenario.Kind, architecture.Name, err)
	}
	adapter := simadapter.New(simulator)
	adapter.SetRequests(scenario.Requests)
	if err := adapter.Build(); err != nil {
		return Result{}, fmt.Errorf("%s/%s: %w", scenario.Kind, architecture.Name, err)
	}
	if err := adapter.Run(); err != nil {
		return Result{}, fmt.Errorf("%s/%s: %w", scenario.Kind, architecture.Name, err)
	}
	return Result{
		Scenario:     scenario,
		Architecture: architecture,
		Stats:        simulator.Stats,
		Responses:    adapter.Responses,
	}, nil
}

// ValidateResults checks accounting invariants for every individual run and
// then checks the intended behavior of each workload.
func ValidateResults(results []Result) []Check {
	checks := make([]Check, 0)
	for _, result := range results {
		checks = append(checks, validateAccounting(result)...)
	}
	checks = append(checks, validateBehavior(results)...)
	return checks
}

func validateAccounting(result Result) []Check {
	prefix := fmt.Sprintf("%s/%s", result.Scenario.Kind, result.Architecture.Name)
	stats := result.Stats
	requestCount := uint64(len(result.Scenario.Requests))
	checks := []Check{
		newCheck(prefix+" request accounting", stats.TotalRequests == requestCount,
			fmt.Sprintf("stats=%d trace=%d", stats.TotalRequests, requestCount)),
		newCheck(prefix+" response accounting", uint64(len(result.Responses)) == requestCount,
			fmt.Sprintf("responses=%d trace=%d", len(result.Responses), requestCount)),
	}

	usesL1 := result.Architecture.Topology != config.TopologyMemoryOnly
	if usesL1 {
		checks = append(checks, newCheck(prefix+" L1 accounting",
			stats.L1Hits+stats.L1Misses == requestCount,
			fmt.Sprintf("hits+misses=%d requests=%d", stats.L1Hits+stats.L1Misses, requestCount)))
	} else {
		checks = append(checks, newCheck(prefix+" no L1 activity",
			stats.L1Hits+stats.L1Misses == 0,
			fmt.Sprintf("L1 accesses=%d", stats.L1Hits+stats.L1Misses)))
	}

	usesVictim := result.Architecture.Topology == config.TopologyFull && result.Architecture.VictimEnabled
	if usesVictim {
		checks = append(checks, newCheck(prefix+" victim accounting",
			stats.VictimHits+stats.VictimMisses == stats.L1Misses,
			fmt.Sprintf("victim accesses=%d L1 misses=%d", stats.VictimHits+stats.VictimMisses, stats.L1Misses)))
	} else {
		checks = append(checks, newCheck(prefix+" no victim activity",
			stats.VictimHits+stats.VictimMisses == 0,
			fmt.Sprintf("victim accesses=%d", stats.VictimHits+stats.VictimMisses)))
	}

	usesL2 := result.Architecture.Topology == config.TopologyL1L2 || result.Architecture.Topology == config.TopologyFull
	if usesL2 {
		checks = append(checks, newCheck(prefix+" L2 accounting",
			stats.L2Hits+stats.L2Misses == stats.L2ReadRequests,
			fmt.Sprintf("hits+misses=%d reads=%d", stats.L2Hits+stats.L2Misses, stats.L2ReadRequests)))
	} else {
		checks = append(checks, newCheck(prefix+" no L2 activity",
			stats.L2Hits+stats.L2Misses+stats.L2ReadRequests == 0,
			fmt.Sprintf("L2 activity=%d", stats.L2Hits+stats.L2Misses+stats.L2ReadRequests)))
	}

	var responseCycles uint64
	var l1Responses, victimResponses, l2Responses, memoryResponses uint64
	for _, response := range result.Responses {
		responseCycles += response.LatencyCycles
		switch response.Location {
		case model.HitL1:
			l1Responses++
		case model.HitVictim:
			victimResponses++
		case model.HitL2:
			l2Responses++
		case model.HitMemory:
			memoryResponses++
		}
	}
	checks = append(checks,
		newCheck(prefix+" cycle accounting", responseCycles == stats.TotalCycles,
			fmt.Sprintf("responses=%d stats=%d", responseCycles, stats.TotalCycles)),
		newCheck(prefix+" response locations",
			l1Responses == stats.L1Hits && victimResponses == stats.VictimHits && l2Responses == stats.L2Hits,
			fmt.Sprintf("response L1/VC/L2=%d/%d/%d stats=%d/%d/%d", l1Responses, victimResponses, l2Responses, stats.L1Hits, stats.VictimHits, stats.L2Hits)),
		newCheck(prefix+" memory response bound", stats.MemoryAccesses >= memoryResponses,
			fmt.Sprintf("memory accesses=%d memory responses=%d", stats.MemoryAccesses, memoryResponses)),
	)
	return checks
}

func validateBehavior(results []Result) []Check {
	checks := make([]Check, 0)

	for _, result := range resultsForTrace(results, benchmark.TraceRepeated) {
		if result.Architecture.Topology == config.TopologyMemoryOnly {
			continue
		}
		requests := uint64(len(result.Scenario.Requests))
		checks = append(checks, newCheck(
			fmt.Sprintf("repeated/%s warms L1", result.Architecture.Name),
			result.Stats.L1Misses == 1 && result.Stats.L1Hits == requests-1,
			fmt.Sprintf("L1=%d hits/%d misses", result.Stats.L1Hits, result.Stats.L1Misses),
		))
	}

	for _, result := range resultsForTrace(results, benchmark.TraceSequential) {
		if result.Architecture.Topology == config.TopologyMemoryOnly {
			continue
		}
		touchedBlocks := make(map[uint64]struct{})
		for _, request := range result.Scenario.Requests {
			touchedBlocks[request.Address/result.Scenario.BlockSizeBytes] = struct{}{}
		}
		expectedMisses := uint64(len(touchedBlocks))
		expectedHits := uint64(len(result.Scenario.Requests)) - expectedMisses
		checks = append(checks, newCheck(
			fmt.Sprintf("sequential/%s follows word-level spatial locality", result.Architecture.Name),
			result.Stats.L1Hits == expectedHits && result.Stats.L1Misses == expectedMisses,
			fmt.Sprintf("L1=%d hits/%d misses, expected=%d/%d", result.Stats.L1Hits, result.Stats.L1Misses, expectedHits, expectedMisses),
		))
	}

	baseline, baselineFound := findResult(results, benchmark.TraceConflict, "l1-l2")
	for _, fullName := range []string{"full-fifo", "full-lru"} {
		full, found := findResult(results, benchmark.TraceConflict, fullName)
		if !baselineFound || !found {
			continue
		}
		checks = append(checks,
			newCheck("conflict/"+fullName+" exercises victim", full.Stats.VictimHits > 0,
				fmt.Sprintf("victim hits=%d", full.Stats.VictimHits)),
			newCheck("conflict/"+fullName+" reduces L2 reads", full.Stats.L2ReadRequests < baseline.Stats.L2ReadRequests,
				fmt.Sprintf("full=%d baseline=%d", full.Stats.L2ReadRequests, baseline.Stats.L2ReadRequests)),
			newCheck("conflict/"+fullName+" reduces cycles", full.Stats.TotalCycles < baseline.Stats.TotalCycles,
				fmt.Sprintf("full=%d baseline=%d", full.Stats.TotalCycles, baseline.Stats.TotalCycles)),
		)
	}

	for _, fullName := range []string{"full-fifo", "full-lru"} {
		mixed, found := findResult(results, benchmark.TraceMixed, fullName)
		if !found {
			continue
		}
		checks = append(checks, newCheck(
			"mixed/"+fullName+" reaches every level",
			mixed.Stats.L1Hits > 0 && mixed.Stats.VictimHits > 0 && mixed.Stats.L2Hits > 0 && mixed.Stats.MemoryAccesses > 0,
			fmt.Sprintf("L1=%d VC=%d L2=%d MEM=%d", mixed.Stats.L1Hits, mixed.Stats.VictimHits, mixed.Stats.L2Hits, mixed.Stats.MemoryAccesses),
		))
	}

	fifo, fifoFound := findResult(results, benchmark.TraceMixed, "full-fifo")
	lru, lruFound := findResult(results, benchmark.TraceMixed, "full-lru")
	if fifoFound && lruFound {
		checks = append(checks,
			newCheck("mixed LRU improves Victim hits",
				lru.Stats.VictimHits > fifo.Stats.VictimHits,
				fmt.Sprintf("LRU=%d FIFO=%d", lru.Stats.VictimHits, fifo.Stats.VictimHits)),
			newCheck("mixed LRU reduces L2 reads",
				lru.Stats.L2ReadRequests < fifo.Stats.L2ReadRequests,
				fmt.Sprintf("LRU=%d FIFO=%d", lru.Stats.L2ReadRequests, fifo.Stats.L2ReadRequests)),
			newCheck("mixed LRU reduces cycles",
				lru.Stats.TotalCycles < fifo.Stats.TotalCycles,
				fmt.Sprintf("LRU=%d FIFO=%d", lru.Stats.TotalCycles, fifo.Stats.TotalCycles)),
		)
	}

	return checks
}

func resultsForTrace(results []Result, kind benchmark.TraceKind) []Result {
	filtered := make([]Result, 0)
	for _, result := range results {
		if result.Scenario.Kind == kind {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

func findResult(results []Result, kind benchmark.TraceKind, architectureName string) (Result, bool) {
	for _, result := range results {
		if result.Scenario.Kind == kind && result.Architecture.Name == architectureName {
			return result, true
		}
	}
	return Result{}, false
}

func newCheck(name string, passed bool, detail string) Check {
	return Check{Name: name, Passed: passed, Detail: detail}
}
