package benchmark

import (
	"reflect"
	"testing"

	"victimcacheproject/internal/model"
)

func TestMergeSortSortsValuesAndRecordsReadsAndWrites(t *testing.T) {
	workload, err := GenerateMergeSortWorkload(MergeSortConfig{
		BaseAddress:          0,
		Length:               4,
		ElementSizeBytes:     8,
		RegionAlignmentBytes: 64,
		BlockSizeBytes:       64,
	})
	if err != nil {
		t.Fatal(err)
	}

	if want := []int64{40, 31, 22, 10}; !reflect.DeepEqual(workload.Input, want) {
		t.Fatalf("input=%v, want %v", workload.Input, want)
	}
	if want := []int64{10, 22, 31, 40}; !reflect.DeepEqual(workload.Sorted, want) {
		t.Fatalf("sorted=%v, want %v", workload.Sorted, want)
	}
	if workload.ArrayBase != 0 || workload.ScratchBase != 64 {
		t.Fatalf("array/scratch bases=%d/%d, want 0/64", workload.ArrayBase, workload.ScratchBase)
	}

	reads, writes := 0, 0
	for index, request := range workload.Scenario.Requests {
		if request.ID != uint64(index+1) {
			t.Fatalf("request %d ID=%d, want %d", index, request.ID, index+1)
		}
		if request.Size != 8 {
			t.Fatalf("request %d size=%d, want 8", index, request.Size)
		}
		switch request.Op {
		case model.Read:
			reads++
		case model.Write:
			writes++
		}
	}
	if reads == 0 || writes == 0 {
		t.Fatalf("merge sort must emit both reads and writes, got %d/%d", reads, writes)
	}
}

func TestMergeSortIsDeterministic(t *testing.T) {
	cfg := MergeSortConfig{Length: 9, ElementSizeBytes: 4, RegionAlignmentBytes: 64, BlockSizeBytes: 64}
	first, err := GenerateMergeSortWorkload(cfg)
	if err != nil {
		t.Fatal(err)
	}
	second, err := GenerateMergeSortWorkload(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("merge-sort workload is not deterministic")
	}
}

func TestMergeSortRejectsInvalidConfiguration(t *testing.T) {
	_, err := GenerateMergeSortWorkload(MergeSortConfig{
		Length:               -1,
		ElementSizeBytes:     4,
		RegionAlignmentBytes: 64,
		BlockSizeBytes:       64,
	})
	if err == nil {
		t.Fatal("negative length must be rejected")
	}
}
