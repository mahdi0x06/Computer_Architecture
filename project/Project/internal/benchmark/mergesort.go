package benchmark

import (
	"fmt"

	"victimcacheproject/internal/model"
)

const DefaultMergeSortLength = 16

// MergeSortConfig describes an in-place merge sort with one scratch array.
type MergeSortConfig struct {
	BaseAddress          uint64
	Length               int
	ElementSizeBytes     uint64
	RegionAlignmentBytes uint64
	BlockSizeBytes       uint64
}

// MergeSortWorkload exposes the original values, verified sorted values, and
// the exact read/write trace executed by Akita.
type MergeSortWorkload struct {
	Scenario    Scenario
	Input       []int64
	Sorted      []int64
	ArrayBase   uint64
	ScratchBase uint64
	RegionBytes uint64
}

// GenerateMergeSortWorkload performs a stable top-down merge sort. Each
// comparison reads both candidates, each merge writes the scratch array, and
// each copy-back reads scratch and writes the source array.
func GenerateMergeSortWorkload(cfg MergeSortConfig) (MergeSortWorkload, error) {
	if cfg.Length <= 0 {
		return MergeSortWorkload{}, fmt.Errorf("merge-sort length must be greater than zero")
	}
	if cfg.ElementSizeBytes == 0 {
		return MergeSortWorkload{}, fmt.Errorf("merge-sort element size must be greater than zero")
	}
	if cfg.RegionAlignmentBytes == 0 {
		return MergeSortWorkload{}, fmt.Errorf("merge-sort region alignment must be greater than zero")
	}
	if cfg.BlockSizeBytes == 0 || cfg.ElementSizeBytes > cfg.BlockSizeBytes {
		return MergeSortWorkload{}, fmt.Errorf("merge-sort element size must fit in a non-zero cache block")
	}
	if cfg.BlockSizeBytes%cfg.ElementSizeBytes != 0 || cfg.RegionAlignmentBytes%cfg.ElementSizeBytes != 0 {
		return MergeSortWorkload{}, fmt.Errorf("merge-sort block size and region alignment must be divisible by the element size")
	}
	if cfg.BaseAddress%cfg.ElementSizeBytes != 0 {
		return MergeSortWorkload{}, fmt.Errorf("merge-sort base address must be aligned to the element size")
	}

	maxUint64 := ^uint64(0)
	if uint64(cfg.Length) > maxUint64/cfg.ElementSizeBytes {
		return MergeSortWorkload{}, fmt.Errorf("merge-sort storage size overflows uint64")
	}
	arrayBytes := uint64(cfg.Length) * cfg.ElementSizeBytes
	regionBytes, err := alignUp(arrayBytes, cfg.RegionAlignmentBytes)
	if err != nil {
		return MergeSortWorkload{}, fmt.Errorf("merge-sort region size: %w", err)
	}
	scratchBase, err := checkedAdd(cfg.BaseAddress, regionBytes)
	if err != nil {
		return MergeSortWorkload{}, fmt.Errorf("merge-sort scratch address: %w", err)
	}
	if _, err := checkedAdd(scratchBase, arrayBytes); err != nil {
		return MergeSortWorkload{}, fmt.Errorf("merge-sort scratch end address: %w", err)
	}

	values := make([]int64, cfg.Length)
	for index := range values {
		// Strictly descending with a small deterministic offset. This guarantees
		// a non-trivial input without random seeds or run-to-run variation.
		values[index] = int64((cfg.Length-index)*10 + index%3)
	}
	input := append([]int64(nil), values...)
	scratch := make([]int64, cfg.Length)
	requests := make([]model.Request, 0)

	var sortRange func(begin, end int)
	sortRange = func(begin, end int) {
		if end-begin <= 1 {
			return
		}
		middle := begin + (end-begin)/2
		sortRange(begin, middle)
		sortRange(middle, end)

		left, right, output := begin, middle, begin
		for left < middle && right < end {
			requests = appendRequest(requests, elementAddress(cfg.BaseAddress, left, cfg.ElementSizeBytes), model.Read, cfg.ElementSizeBytes)
			requests = appendRequest(requests, elementAddress(cfg.BaseAddress, right, cfg.ElementSizeBytes), model.Read, cfg.ElementSizeBytes)
			if values[left] <= values[right] {
				scratch[output] = values[left]
				left++
			} else {
				scratch[output] = values[right]
				right++
			}
			requests = appendRequest(requests, elementAddress(scratchBase, output, cfg.ElementSizeBytes), model.Write, cfg.ElementSizeBytes)
			output++
		}
		for left < middle {
			requests = appendRequest(requests, elementAddress(cfg.BaseAddress, left, cfg.ElementSizeBytes), model.Read, cfg.ElementSizeBytes)
			scratch[output] = values[left]
			requests = appendRequest(requests, elementAddress(scratchBase, output, cfg.ElementSizeBytes), model.Write, cfg.ElementSizeBytes)
			left++
			output++
		}
		for right < end {
			requests = appendRequest(requests, elementAddress(cfg.BaseAddress, right, cfg.ElementSizeBytes), model.Read, cfg.ElementSizeBytes)
			scratch[output] = values[right]
			requests = appendRequest(requests, elementAddress(scratchBase, output, cfg.ElementSizeBytes), model.Write, cfg.ElementSizeBytes)
			right++
			output++
		}
		for index := begin; index < end; index++ {
			requests = appendRequest(requests, elementAddress(scratchBase, index, cfg.ElementSizeBytes), model.Read, cfg.ElementSizeBytes)
			values[index] = scratch[index]
			requests = appendRequest(requests, elementAddress(cfg.BaseAddress, index, cfg.ElementSizeBytes), model.Write, cfg.ElementSizeBytes)
		}
	}

	sortRange(0, len(values))

	return MergeSortWorkload{
		Scenario: Scenario{
			Kind:           TraceMergeSort,
			Name:           fmt.Sprintf("Merge sort of %d elements", cfg.Length),
			Description:    "Runs stable top-down merge sort and sends comparison reads, scratch writes, scratch reads, and copy-back writes through the memory hierarchy.",
			BlockSizeBytes: cfg.BlockSizeBytes,
			Requests:       requests,
		},
		Input:       input,
		Sorted:      append([]int64(nil), values...),
		ArrayBase:   cfg.BaseAddress,
		ScratchBase: scratchBase,
		RegionBytes: regionBytes,
	}, nil
}
