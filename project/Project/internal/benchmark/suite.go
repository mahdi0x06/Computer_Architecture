package benchmark

import (
	"fmt"
	"strings"

	"victimcacheproject/internal/model"
)

// TraceKind identifies a deterministic memory-access workload.
type TraceKind string

const (
	TraceRepeated       TraceKind = "repeated"
	TraceSequential     TraceKind = "sequential"
	TraceConflict       TraceKind = "conflict"
	TraceMixed          TraceKind = "mixed"
	TraceMatrixMultiply TraceKind = "matrix-multiply"
	TraceMergeSort      TraceKind = "merge-sort"
)

// SuiteConfig contains the architectural information needed to construct
// traces that deliberately exercise specific levels of the hierarchy.
type SuiteConfig struct {
	BaseAddress     uint64
	BlockSizeBytes  uint64
	L1SizeBytes     uint64
	L2SizeBytes     uint64
	L2Associativity int
	VictimEntries   int
	NumberOfBlocks  int
	Repetitions     int
	SequentialWords int
	WordSizeBytes   uint64
	AccessSizeBytes uint64
	MatrixDimension int
	MergeSortLength int
}

// Scenario is one named workload in the complete test bench.
type Scenario struct {
	Kind           TraceKind
	Name           string
	Description    string
	BlockSizeBytes uint64
	Requests       []model.Request
}

func AllTraceKinds() []TraceKind {
	return []TraceKind{
		TraceRepeated,
		TraceSequential,
		TraceConflict,
		TraceMixed,
		TraceMatrixMultiply,
		TraceMergeSort,
	}
}

func ParseTraceKind(value string) (TraceKind, error) {
	kind := TraceKind(strings.ToLower(strings.TrimSpace(value)))
	switch kind {
	case "matrix", "matmul":
		kind = TraceMatrixMultiply
	case "mergesort":
		kind = TraceMergeSort
	}
	for _, supported := range AllTraceKinds() {
		if kind == supported {
			return kind, nil
		}
	}
	return "", fmt.Errorf("unsupported trace %q; expected repeated, sequential, conflict, mixed, matrix-multiply, merge-sort, or all", value)
}

// GenerateSuite creates all workloads used by the final project evaluation.
func GenerateSuite(cfg SuiteConfig) ([]Scenario, error) {
	scenarios := make([]Scenario, 0, len(AllTraceKinds()))
	for _, kind := range AllTraceKinds() {
		scenario, err := GenerateScenario(kind, cfg)
		if err != nil {
			return nil, err
		}
		scenarios = append(scenarios, scenario)
	}
	return scenarios, nil
}

// GenerateScenario creates one deterministic workload.
func GenerateScenario(kind TraceKind, cfg SuiteConfig) (Scenario, error) {
	if err := validateSuiteConfig(cfg); err != nil {
		return Scenario{}, err
	}

	var scenario Scenario
	scenario.Kind = kind
	scenario.BlockSizeBytes = cfg.BlockSizeBytes
	switch kind {
	case TraceRepeated:
		scenario.Name = "Repeated locality"
		scenario.Description = "Repeats one address so the first access misses and all later accesses hit in L1."
		scenario.Requests = generateRepeatedTrace(cfg)
	case TraceSequential:
		scenario.Name = "Sequential word stream"
		scenario.Description = "Reads consecutive words one by one so each cache-line fill is followed by spatial-locality hits within that block."
		scenario.Requests = generateSequentialTrace(cfg)
	case TraceConflict:
		scenario.Name = "L1 conflict thrashing"
		scenario.Description = "Uses addresses separated by the L1 capacity so every block maps to the same direct-mapped L1 line."
		scenario.Requests = GenerateConflictTrace(ConflictConfig{
			BaseAddress:     cfg.BaseAddress,
			CacheSizeBytes:  cfg.L1SizeBytes,
			NumberOfBlocks:  cfg.NumberOfBlocks,
			Repetitions:     cfg.Repetitions,
			AccessSizeBytes: cfg.AccessSizeBytes,
		})
	case TraceMixed:
		scenario.Name = "Mixed hierarchy coverage"
		scenario.Description = "Uses deterministic locality, Victim reuse, and a long policy-stress phase that makes FIFO and LRU diverge measurably."
		scenario.Requests = generateMixedTrace(cfg)
	case TraceMatrixMultiply:
		dimension := cfg.MatrixDimension
		if dimension == 0 {
			dimension = DefaultMatrixDimension
		}
		workload, err := GenerateMatrixMultiplyWorkload(MatrixMultiplyConfig{
			BaseAddress:          cfg.BaseAddress,
			Dimension:            dimension,
			ElementSizeBytes:     cfg.WordSizeBytes,
			RegionAlignmentBytes: cfg.L1SizeBytes,
			BlockSizeBytes:       cfg.BlockSizeBytes,
		})
		if err != nil {
			return Scenario{}, err
		}
		scenario = workload.Scenario
		scenario.BlockSizeBytes = cfg.BlockSizeBytes
	case TraceMergeSort:
		length := cfg.MergeSortLength
		if length == 0 {
			length = DefaultMergeSortLength
		}
		workload, err := GenerateMergeSortWorkload(MergeSortConfig{
			BaseAddress:          cfg.BaseAddress,
			Length:               length,
			ElementSizeBytes:     cfg.WordSizeBytes,
			RegionAlignmentBytes: cfg.L1SizeBytes,
			BlockSizeBytes:       cfg.BlockSizeBytes,
		})
		if err != nil {
			return Scenario{}, err
		}
		scenario = workload.Scenario
		scenario.BlockSizeBytes = cfg.BlockSizeBytes
	default:
		return Scenario{}, fmt.Errorf("unsupported trace kind %q", kind)
	}

	return scenario, nil
}

func validateSuiteConfig(cfg SuiteConfig) error {
	if cfg.BlockSizeBytes == 0 {
		return fmt.Errorf("block size must be greater than zero")
	}
	if cfg.L1SizeBytes == 0 || cfg.L1SizeBytes%cfg.BlockSizeBytes != 0 {
		return fmt.Errorf("L1 size must be non-zero and divisible by block size")
	}
	if cfg.L2SizeBytes == 0 || cfg.L2Associativity <= 0 {
		return fmt.Errorf("L2 size and associativity must be greater than zero")
	}
	if cfg.L2SizeBytes%(cfg.BlockSizeBytes*uint64(cfg.L2Associativity)) != 0 {
		return fmt.Errorf("L2 size must be divisible by block size times associativity")
	}
	if cfg.NumberOfBlocks <= 0 {
		return fmt.Errorf("number of blocks must be greater than zero")
	}
	if cfg.Repetitions <= 0 {
		return fmt.Errorf("repetitions must be greater than zero")
	}
	if cfg.SequentialWords <= 0 {
		return fmt.Errorf("sequential word count must be greater than zero")
	}
	if cfg.WordSizeBytes == 0 || cfg.WordSizeBytes > cfg.BlockSizeBytes {
		return fmt.Errorf("word size must be between 1 and the block size")
	}
	if cfg.BlockSizeBytes%cfg.WordSizeBytes != 0 {
		return fmt.Errorf("block size must be divisible by word size")
	}
	if cfg.BaseAddress%cfg.WordSizeBytes != 0 {
		return fmt.Errorf("base address must be aligned to the word size")
	}
	if cfg.AccessSizeBytes == 0 || cfg.AccessSizeBytes > cfg.BlockSizeBytes {
		return fmt.Errorf("access size must be between 1 and the block size")
	}
	if cfg.VictimEntries < 0 {
		return fmt.Errorf("victim entries cannot be negative")
	}
	if cfg.MatrixDimension < 0 {
		return fmt.Errorf("matrix dimension cannot be negative")
	}
	if cfg.MergeSortLength < 0 {
		return fmt.Errorf("merge-sort length cannot be negative")
	}
	return nil
}

func generateRepeatedTrace(cfg SuiteConfig) []model.Request {
	total := cfg.NumberOfBlocks * cfg.Repetitions
	requests := make([]model.Request, 0, total)
	for i := 0; i < total; i++ {
		requests = appendRequest(requests, cfg.BaseAddress, model.Read, cfg.AccessSizeBytes)
	}
	return requests
}

func generateSequentialTrace(cfg SuiteConfig) []model.Request {
	requests := make([]model.Request, 0, cfg.SequentialWords)
	for word := 0; word < cfg.SequentialWords; word++ {
		address := cfg.BaseAddress + uint64(word)*cfg.WordSizeBytes
		requests = appendRequest(requests, address, model.Read, cfg.WordSizeBytes)
	}
	return requests
}

func generateMixedTrace(cfg SuiteConfig) []model.Request {
	requests := make([]model.Request, 0, 1400)

	// The mixed trace is intentionally deterministic. It contains three
	// disjoint regions so that each region has one clear purpose and cannot
	// accidentally inherit L1 hits from another region.

	// Phase 1 — long FIFO/LRU policy stress.
	//
	// A0..A8 occupy nine different L1 indices. A0..A3 are repeatedly touched,
	// making them the most recently used blocks. B0..B8 then conflict with the
	// corresponding A blocks and push A0..A7 into the eight-entry Victim Cache.
	// Inserting A8 overflows the Victim Cache:
	//   * FIFO evicts A0 because A0 entered first.
	//   * LRU evicts one of the cold A4..A7 blocks because A0..A3 were touched.
	// Probing A0..A3 therefore creates four L2 hits under FIFO but four Victim
	// hits under LRU. Repeating this complete epoch many times amplifies the
	// policy difference while keeping the address sequence reproducible.
	const (
		policyIterations = 32
		hotBlocks        = 4
		hotRounds        = 4
	)
	policyBaseBlock := uint64(16) // L1 indices 16..24; disjoint from phases 2/3.
	for iteration := 0; iteration < policyIterations; iteration++ {
		// Load A0..A8 into nine distinct L1 lines.
		for block := 0; block < cfg.VictimEntries+1; block++ {
			address := (policyBaseBlock + uint64(block)) * cfg.BlockSizeBytes
			requests = appendRequest(requests, address, model.Read, cfg.AccessSizeBytes)
		}

		// Refresh A0..A3. Their L1 recency must be newer than the cold A4..A7
		// blocks before all A blocks are evicted into the Victim Cache.
		for round := 0; round < hotRounds; round++ {
			for block := 0; block < hotBlocks; block++ {
				address := (policyBaseBlock + uint64(block)) * cfg.BlockSizeBytes
				requests = appendRequest(requests, address, model.Read, cfg.AccessSizeBytes)
			}
		}

		// B_i is exactly one L1 capacity away from A_i, so B_i and A_i map
		// to the same direct-mapped L1 line. B0..B7 fill the Victim Cache;
		// B8 inserts A8 and triggers one replacement decision.
		for block := 0; block < cfg.VictimEntries+1; block++ {
			blockAddress := policyBaseBlock + uint64(block)
			address := blockAddress*cfg.BlockSizeBytes + cfg.L1SizeBytes
			requests = appendRequest(requests, address, model.Read, cfg.AccessSizeBytes)
		}

		// These four probes are the policy discriminator. FIFO has already
		// discarded them in insertion order; LRU retained them because they
		// were the most recently used A blocks.
		for block := 0; block < hotBlocks; block++ {
			address := (policyBaseBlock + uint64(block)) * cfg.BlockSizeBytes
			requests = appendRequest(requests, address, model.Read, cfg.AccessSizeBytes)
		}
	}

	// Phase 2 — explicit L1 locality, on L1 index 1.
	// The first request misses and the next 31 requests hit in L1.
	const localityAccesses = 32
	localityAddress := cfg.BaseAddress + cfg.BlockSizeBytes
	for i := 0; i < localityAccesses; i++ {
		requests = appendRequest(requests, localityAddress, model.Read, cfg.AccessSizeBytes)
	}

	// Phase 3 — policy-independent Victim reuse, on L1 index 2.
	// Four conflicting blocks fit in one L1 line plus the Victim Cache. This
	// guarantees that FIFO also records many Victim hits; the mixed benchmark
	// does not make FIFO look inactive, it only demonstrates that LRU performs
	// better under the policy-stress region above.
	smallConflictBlocks := maxInt(2, cfg.NumberOfBlocks)
	if cfg.VictimEntries > 0 && smallConflictBlocks > cfg.VictimEntries+1 {
		smallConflictBlocks = cfg.VictimEntries + 1
	}
	const victimReusePasses = 16
	smallConflictBase := cfg.BaseAddress + 2*cfg.BlockSizeBytes
	for repetition := 0; repetition < victimReusePasses; repetition++ {
		for block := 0; block < smallConflictBlocks; block++ {
			address := smallConflictBase + uint64(block)*cfg.L1SizeBytes
			requests = appendRequest(requests, address, model.Read, cfg.AccessSizeBytes)
		}
	}

	return requests
}

func appendRequest(requests []model.Request, address uint64, op model.Operation, size uint64) []model.Request {
	return append(requests, model.Request{
		ID:      uint64(len(requests) + 1),
		Address: address,
		Op:      op,
		Size:    size,
	})
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
