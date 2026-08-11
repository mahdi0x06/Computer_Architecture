package benchmark

import (
	"reflect"
	"testing"

	"victimcacheproject/internal/model"
)

func suiteTestConfig() SuiteConfig {
	return SuiteConfig{
		BaseAddress:     0,
		BlockSizeBytes:  64,
		L1SizeBytes:     4 * 1024,
		L2SizeBytes:     64 * 1024,
		L2Associativity: 8,
		VictimEntries:   8,
		NumberOfBlocks:  4,
		Repetitions:     8,
		SequentialWords: 32,
		WordSizeBytes:   4,
		AccessSizeBytes: 8,
	}
}

func TestGenerateRepeatedScenario(t *testing.T) {
	scenario, err := GenerateScenario(TraceRepeated, suiteTestConfig())
	if err != nil {
		t.Fatal(err)
	}
	if len(scenario.Requests) != 32 {
		t.Fatalf("requests=%d, want 32", len(scenario.Requests))
	}
	for i, request := range scenario.Requests {
		if request.Address != 0 {
			t.Fatalf("request %d address=%d, want 0", i, request.Address)
		}
		if request.ID != uint64(i+1) {
			t.Fatalf("request %d ID=%d, want %d", i, request.ID, i+1)
		}
	}
}

func TestGenerateSequentialScenarioWalksWordByWord(t *testing.T) {
	cfg := suiteTestConfig()
	scenario, err := GenerateScenario(TraceSequential, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(scenario.Requests) != cfg.SequentialWords {
		t.Fatalf("requests=%d, want %d", len(scenario.Requests), cfg.SequentialWords)
	}
	for i, request := range scenario.Requests {
		wantAddress := cfg.BaseAddress + uint64(i)*cfg.WordSizeBytes
		if request.Address != wantAddress {
			t.Fatalf("request %d address=%d, want %d", i, request.Address, wantAddress)
		}
		if request.Size != cfg.WordSizeBytes {
			t.Fatalf("request %d size=%d, want word size %d", i, request.Size, cfg.WordSizeBytes)
		}
	}

	// With 64-byte blocks and 4-byte words, requests 0..15 belong to
	// block 0 and requests 16..31 belong to block 1.
	if scenario.Requests[15].Address != 60 || scenario.Requests[16].Address != 64 {
		t.Fatalf("unexpected block boundary: word15=%d word16=%d", scenario.Requests[15].Address, scenario.Requests[16].Address)
	}
}

func TestGenerateMixedScenarioIsDeterministicAndPolicySensitive(t *testing.T) {
	cfg := suiteTestConfig()
	first, err := GenerateScenario(TraceMixed, cfg)
	if err != nil {
		t.Fatal(err)
	}
	second, err := GenerateScenario(TraceMixed, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.Requests, second.Requests) {
		t.Fatal("mixed trace is not deterministic")
	}

	// Default recipe:
	//   32 policy epochs x 38 requests = 1216
	//   32 explicit L1-locality requests
	//   4 conflicting blocks x 16 Victim-reuse passes = 64
	// Total = 1312 requests.
	if len(first.Requests) != 1312 {
		t.Fatalf("mixed requests=%d, want 1312", len(first.Requests))
	}
	for _, request := range first.Requests {
		if request.Op != model.Read {
			t.Fatalf("mixed trace contains non-read operation %v", request.Op)
		}
	}

	const epochLength = 38
	for epoch := 0; epoch < 32; epoch++ {
		offset := epoch * epochLength

		// A0..A8: blocks 16..24, nine distinct L1 indices.
		for block := 0; block < 9; block++ {
			want := uint64(16+block) * cfg.BlockSizeBytes
			if got := first.Requests[offset+block].Address; got != want {
				t.Fatalf("epoch %d A%d address=%d, want %d", epoch, block, got, want)
			}
		}

		// Four rounds over hot A0..A3.
		for round := 0; round < 4; round++ {
			for block := 0; block < 4; block++ {
				i := offset + 9 + round*4 + block
				want := uint64(16+block) * cfg.BlockSizeBytes
				if got := first.Requests[i].Address; got != want {
					t.Fatalf("epoch %d hot round %d block %d address=%d, want %d", epoch, round, block, got, want)
				}
			}
		}

		// B0..B8 conflict with A0..A8 by exactly one L1 capacity.
		for block := 0; block < 9; block++ {
			i := offset + 25 + block
			want := uint64(16+block)*cfg.BlockSizeBytes + cfg.L1SizeBytes
			if got := first.Requests[i].Address; got != want {
				t.Fatalf("epoch %d B%d address=%d, want %d", epoch, block, got, want)
			}
		}

		// Probe hot A0..A3. These are the FIFO/LRU discriminator.
		for block := 0; block < 4; block++ {
			i := offset + 34 + block
			want := uint64(16+block) * cfg.BlockSizeBytes
			if got := first.Requests[i].Address; got != want {
				t.Fatalf("epoch %d probe A%d address=%d, want %d", epoch, block, got, want)
			}
		}
	}

	// Explicit L1 phase: 32 requests to block 1.
	for i := 1216; i < 1248; i++ {
		if got := first.Requests[i].Address; got != cfg.BlockSizeBytes {
			t.Fatalf("L1 locality request %d address=%d, want %d", i, got, cfg.BlockSizeBytes)
		}
	}

	// Policy-independent Victim phase: four index-2 conflicts, 16 passes.
	wantVictimPass := []uint64{128, 4224, 8320, 12416}
	for pass := 0; pass < 16; pass++ {
		for block, want := range wantVictimPass {
			i := 1248 + pass*len(wantVictimPass) + block
			if got := first.Requests[i].Address; got != want {
				t.Fatalf("Victim phase request %d address=%d, want %d", i, got, want)
			}
		}
	}
}

func TestGenerateSuiteContainsEveryTrace(t *testing.T) {
	scenarios, err := GenerateSuite(suiteTestConfig())
	if err != nil {
		t.Fatal(err)
	}
	if len(scenarios) != len(AllTraceKinds()) {
		t.Fatalf("scenarios=%d, want %d", len(scenarios), len(AllTraceKinds()))
	}
	for i, kind := range AllTraceKinds() {
		if scenarios[i].Kind != kind {
			t.Fatalf("scenario %d kind=%s, want %s", i, scenarios[i].Kind, kind)
		}
	}
}

func TestParseTraceKindRejectsUnknownValue(t *testing.T) {
	if _, err := ParseTraceKind("not-a-trace"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestApplicationTraceAliases(t *testing.T) {
	tests := map[string]TraceKind{
		"matrix":          TraceMatrixMultiply,
		"matmul":          TraceMatrixMultiply,
		"matrix-multiply": TraceMatrixMultiply,
		"mergesort":       TraceMergeSort,
		"merge-sort":      TraceMergeSort,
	}
	for input, want := range tests {
		got, err := ParseTraceKind(input)
		if err != nil {
			t.Fatalf("ParseTraceKind(%q): %v", input, err)
		}
		if got != want {
			t.Fatalf("ParseTraceKind(%q)=%q, want %q", input, got, want)
		}
	}
}

func TestGenerateApplicationScenariosUseConfiguredSizes(t *testing.T) {
	cfg := suiteTestConfig()
	cfg.MatrixDimension = 3
	cfg.MergeSortLength = 7

	matrix, err := GenerateScenario(TraceMatrixMultiply, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(matrix.Requests) != 3*3*(2*3+1) {
		t.Fatalf("matrix requests=%d, want 63", len(matrix.Requests))
	}

	mergeSort, err := GenerateScenario(TraceMergeSort, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(mergeSort.Requests) == 0 {
		t.Fatal("merge-sort scenario must contain memory requests")
	}
}
