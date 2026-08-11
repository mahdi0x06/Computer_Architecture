package benchmark

import (
	"reflect"
	"testing"

	"victimcacheproject/internal/model"
)

func TestMatrixMultiplyComputesValuesAndRecordsMemoryTraffic(t *testing.T) {
	workload, err := GenerateMatrixMultiplyWorkload(MatrixMultiplyConfig{
		BaseAddress:          0,
		Dimension:            2,
		ElementSizeBytes:     8,
		RegionAlignmentBytes: 64,
		BlockSizeBytes:       64,
	})
	if err != nil {
		t.Fatal(err)
	}

	if want := []int64{12, 27, 22, 49}; !reflect.DeepEqual(workload.Product, want) {
		t.Fatalf("product=%v, want %v", workload.Product, want)
	}
	if workload.LeftBase != 0 || workload.RightBase != 64 || workload.ProductBase != 128 {
		t.Fatalf("unexpected matrix bases A/B/C=%d/%d/%d", workload.LeftBase, workload.RightBase, workload.ProductBase)
	}
	if workload.Scenario.BlockSizeBytes != 64 {
		t.Fatalf("block size=%d, want 64", workload.Scenario.BlockSizeBytes)
	}

	requests := workload.Scenario.Requests
	if len(requests) != 20 {
		t.Fatalf("requests=%d, want 20", len(requests))
	}
	reads, writes := 0, 0
	for index, request := range requests {
		if request.ID != uint64(index+1) {
			t.Fatalf("request %d ID=%d, want %d", index, request.ID, index+1)
		}
		switch request.Op {
		case model.Read:
			reads++
		case model.Write:
			writes++
		}
	}
	if reads != 16 || writes != 4 {
		t.Fatalf("reads/writes=%d/%d, want 16/4", reads, writes)
	}
	if requests[0].Address != workload.LeftBase || requests[1].Address != workload.RightBase {
		t.Fatalf("first A/B addresses=%d/%d", requests[0].Address, requests[1].Address)
	}
}

func TestMatrixMultiplyIsDeterministic(t *testing.T) {
	cfg := MatrixMultiplyConfig{Dimension: 3, ElementSizeBytes: 4, RegionAlignmentBytes: 64, BlockSizeBytes: 64}
	first, err := GenerateMatrixMultiplyWorkload(cfg)
	if err != nil {
		t.Fatal(err)
	}
	second, err := GenerateMatrixMultiplyWorkload(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("matrix workload is not deterministic")
	}
}

func TestMatrixMultiplyRejectsInvalidConfiguration(t *testing.T) {
	_, err := GenerateMatrixMultiplyWorkload(MatrixMultiplyConfig{
		Dimension:            0,
		ElementSizeBytes:     4,
		RegionAlignmentBytes: 64,
		BlockSizeBytes:       64,
	})
	if err == nil {
		t.Fatal("zero dimension must be rejected")
	}
}
