package benchmark

import (
	"reflect"
	"testing"

	"victimcacheproject/internal/model"
)

func testConflictConfig() ConflictConfig {
	return ConflictConfig{
		BaseAddress:     0x1000,
		CacheSizeBytes:  4 * 1024,
		NumberOfBlocks:  3,
		Repetitions:     4,
		AccessSizeBytes: 8,
	}
}

func TestGenerateConflictTraceLengthAndPattern(t *testing.T) {
	cfg := testConflictConfig()
	trace := GenerateConflictTrace(cfg)

	wantLength := cfg.NumberOfBlocks * cfg.Repetitions
	if len(trace) != wantLength {
		t.Fatalf("trace length = %d, want %d", len(trace), wantLength)
	}

	for i, req := range trace {
		block := i % cfg.NumberOfBlocks
		wantAddress := cfg.BaseAddress + uint64(block)*cfg.CacheSizeBytes
		if req.Address != wantAddress {
			t.Errorf("request %d address = %#x, want %#x", i, req.Address, wantAddress)
		}
		if req.ID != uint64(i+1) {
			t.Errorf("request %d ID = %d, want %d", i, req.ID, i+1)
		}
		if req.Op != model.Read {
			t.Errorf("request %d operation = %v, want Read", i, req.Op)
		}
		if req.Size != cfg.AccessSizeBytes {
			t.Errorf("request %d size = %d, want %d", i, req.Size, cfg.AccessSizeBytes)
		}
	}
}

func TestGenerateConflictTraceAlternatesConflictingAddresses(t *testing.T) {
	cfg := ConflictConfig{
		BaseAddress:     0x2000,
		CacheSizeBytes:  4 * 1024,
		NumberOfBlocks:  2,
		Repetitions:     4,
		AccessSizeBytes: 8,
	}
	trace := GenerateConflictTrace(cfg)

	addressA := cfg.BaseAddress
	addressB := cfg.BaseAddress + cfg.CacheSizeBytes
	want := []uint64{addressA, addressB, addressA, addressB, addressA, addressB, addressA, addressB}

	for i, req := range trace {
		if req.Address != want[i] {
			t.Errorf("request %d address = %#x, want %#x", i, req.Address, want[i])
		}
	}
}

func TestGenerateConflictTraceAddressSpacing(t *testing.T) {
	cfg := testConflictConfig()
	trace := GenerateConflictTrace(cfg)

	for i := 1; i < cfg.NumberOfBlocks; i++ {
		spacing := trace[i].Address - trace[i-1].Address
		if spacing != cfg.CacheSizeBytes {
			t.Errorf("address spacing = %d, want %d", spacing, cfg.CacheSizeBytes)
		}
	}
}

func TestGenerateConflictTraceIsDeterministic(t *testing.T) {
	cfg := testConflictConfig()
	first := GenerateConflictTrace(cfg)
	second := GenerateConflictTrace(cfg)

	if !reflect.DeepEqual(first, second) {
		t.Fatal("GenerateConflictTrace returned different traces for the same configuration")
	}
}

func TestGenerateConflictTraceInvalidInputReturnsEmptyTrace(t *testing.T) {
	tests := []struct {
		name string
		cfg  ConflictConfig
	}{
		{name: "zero cache capacity", cfg: ConflictConfig{CacheSizeBytes: 0, NumberOfBlocks: 1, Repetitions: 1}},
		{name: "no blocks", cfg: ConflictConfig{CacheSizeBytes: 1, NumberOfBlocks: 0, Repetitions: 1}},
		{name: "negative blocks", cfg: ConflictConfig{CacheSizeBytes: 1, NumberOfBlocks: -1, Repetitions: 1}},
		{name: "no repetitions", cfg: ConflictConfig{CacheSizeBytes: 1, NumberOfBlocks: 1, Repetitions: 0}},
		{name: "negative repetitions", cfg: ConflictConfig{CacheSizeBytes: 1, NumberOfBlocks: 1, Repetitions: -1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trace := GenerateConflictTrace(tt.cfg)
			if len(trace) != 0 {
				t.Fatalf("trace length = %d, want 0", len(trace))
			}
		})
	}
}
