package benchmark

import (
	"fmt"

	"victimcacheproject/internal/model"
)

const DefaultMatrixDimension = 8

// MatrixMultiplyConfig describes a square, row-major matrix multiplication.
// RegionAlignmentBytes controls the spacing between A, B, and C. Using the L1
// capacity here keeps the arrays disjoint while deliberately mapping matching
// offsets to the same direct-mapped L1 indices.
type MatrixMultiplyConfig struct {
	BaseAddress          uint64
	Dimension            int
	ElementSizeBytes     uint64
	RegionAlignmentBytes uint64
	BlockSizeBytes       uint64
}

// MatrixMultiplyWorkload contains both the verified numeric computation and
// the memory-request scenario that is executed by Akita.
type MatrixMultiplyWorkload struct {
	Scenario    Scenario
	Left        []int64
	Right       []int64
	Product     []int64
	LeftBase    uint64
	RightBase   uint64
	ProductBase uint64
	RegionBytes uint64
}

// GenerateMatrixMultiplyWorkload computes C=A*B and records the two operand
// reads and one result write performed by the conventional i-j-k algorithm.
func GenerateMatrixMultiplyWorkload(cfg MatrixMultiplyConfig) (MatrixMultiplyWorkload, error) {
	matrixBytes, err := squareStorageBytes(cfg.Dimension, cfg.ElementSizeBytes)
	if err != nil {
		return MatrixMultiplyWorkload{}, err
	}
	if cfg.RegionAlignmentBytes == 0 {
		return MatrixMultiplyWorkload{}, fmt.Errorf("matrix region alignment must be greater than zero")
	}
	if cfg.BlockSizeBytes == 0 || cfg.ElementSizeBytes > cfg.BlockSizeBytes {
		return MatrixMultiplyWorkload{}, fmt.Errorf("matrix element size must fit in a non-zero cache block")
	}
	if cfg.BlockSizeBytes%cfg.ElementSizeBytes != 0 || cfg.RegionAlignmentBytes%cfg.ElementSizeBytes != 0 {
		return MatrixMultiplyWorkload{}, fmt.Errorf("matrix block size and region alignment must be divisible by the element size")
	}
	if cfg.BaseAddress%cfg.ElementSizeBytes != 0 {
		return MatrixMultiplyWorkload{}, fmt.Errorf("matrix base address must be aligned to the element size")
	}

	regionBytes, err := alignUp(matrixBytes, cfg.RegionAlignmentBytes)
	if err != nil {
		return MatrixMultiplyWorkload{}, fmt.Errorf("matrix region size: %w", err)
	}
	rightBase, err := checkedAdd(cfg.BaseAddress, regionBytes)
	if err != nil {
		return MatrixMultiplyWorkload{}, fmt.Errorf("matrix B address: %w", err)
	}
	productBase, err := checkedAdd(rightBase, regionBytes)
	if err != nil {
		return MatrixMultiplyWorkload{}, fmt.Errorf("matrix C address: %w", err)
	}
	if _, err := checkedAdd(productBase, matrixBytes); err != nil {
		return MatrixMultiplyWorkload{}, fmt.Errorf("matrix C end address: %w", err)
	}

	elements := cfg.Dimension * cfg.Dimension
	left := make([]int64, elements)
	right := make([]int64, elements)
	product := make([]int64, elements)
	for row := 0; row < cfg.Dimension; row++ {
		for column := 0; column < cfg.Dimension; column++ {
			index := row*cfg.Dimension + column
			left[index] = int64((row+1)*2 + column)
			right[index] = int64((column+1)*3 - row)
		}
	}

	requests := make([]model.Request, 0)
	for row := 0; row < cfg.Dimension; row++ {
		for column := 0; column < cfg.Dimension; column++ {
			var sum int64
			for inner := 0; inner < cfg.Dimension; inner++ {
				leftIndex := row*cfg.Dimension + inner
				rightIndex := inner*cfg.Dimension + column
				requests = appendRequest(requests, elementAddress(cfg.BaseAddress, leftIndex, cfg.ElementSizeBytes), model.Read, cfg.ElementSizeBytes)
				requests = appendRequest(requests, elementAddress(rightBase, rightIndex, cfg.ElementSizeBytes), model.Read, cfg.ElementSizeBytes)
				sum += left[leftIndex] * right[rightIndex]
			}

			productIndex := row*cfg.Dimension + column
			product[productIndex] = sum
			requests = appendRequest(requests, elementAddress(productBase, productIndex, cfg.ElementSizeBytes), model.Write, cfg.ElementSizeBytes)
		}
	}

	return MatrixMultiplyWorkload{
		Scenario: Scenario{
			Kind:           TraceMatrixMultiply,
			Name:           fmt.Sprintf("%dx%d matrix multiplication", cfg.Dimension, cfg.Dimension),
			Description:    "Runs the conventional row-major i-j-k matrix multiplication and sends every A/B read and C write through the memory hierarchy.",
			BlockSizeBytes: cfg.BlockSizeBytes,
			Requests:       requests,
		},
		Left:        left,
		Right:       right,
		Product:     product,
		LeftBase:    cfg.BaseAddress,
		RightBase:   rightBase,
		ProductBase: productBase,
		RegionBytes: regionBytes,
	}, nil
}

func squareStorageBytes(dimension int, elementSize uint64) (uint64, error) {
	if dimension <= 0 {
		return 0, fmt.Errorf("matrix dimension must be greater than zero")
	}
	if elementSize == 0 {
		return 0, fmt.Errorf("matrix element size must be greater than zero")
	}
	dimension64 := uint64(dimension)
	maxInt := int(^uint(0) >> 1)
	if dimension > maxInt/dimension {
		return 0, fmt.Errorf("matrix dimension is too large for this platform")
	}
	maxUint64 := ^uint64(0)
	if dimension64 > maxUint64/dimension64 {
		return 0, fmt.Errorf("matrix dimension is too large")
	}
	elements := dimension64 * dimension64
	if elements > maxUint64/elementSize {
		return 0, fmt.Errorf("matrix storage size overflows uint64")
	}
	return elements * elementSize, nil
}

func elementAddress(base uint64, index int, elementSize uint64) uint64 {
	return base + uint64(index)*elementSize
}

func alignUp(value, alignment uint64) (uint64, error) {
	if alignment == 0 {
		return 0, fmt.Errorf("alignment must be greater than zero")
	}
	remainder := value % alignment
	if remainder == 0 {
		return value, nil
	}
	delta := alignment - remainder
	return checkedAdd(value, delta)
}

func checkedAdd(a, b uint64) (uint64, error) {
	if a > ^uint64(0)-b {
		return 0, fmt.Errorf("address arithmetic overflows uint64")
	}
	return a + b, nil
}
