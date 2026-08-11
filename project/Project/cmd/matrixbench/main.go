package main

import (
	"flag"
	"fmt"
	"os"

	"victimcacheproject/internal/benchmark"
	"victimcacheproject/internal/config"
	benchrunner "victimcacheproject/internal/testbench"
)

func main() {
	dimension := flag.Int("size", benchmark.DefaultMatrixDimension, "square matrix dimension")
	elementSize := flag.Uint64("element-size", 4, "matrix element size in bytes")
	csvPath := flag.String("csv", "matrix-results.csv", "CSV output path; empty disables CSV output")
	strict := flag.Bool("strict", true, "exit with status 1 if a validation check fails")
	verboseChecks := flag.Bool("verbose-checks", false, "print every validation assertion")
	printValues := flag.Bool("print-values", false, "print A, B, and C values")
	flag.Parse()

	base := config.Default()
	workload, err := benchmark.GenerateMatrixMultiplyWorkload(benchmark.MatrixMultiplyConfig{
		BaseAddress:          0,
		Dimension:            *dimension,
		ElementSizeBytes:     *elementSize,
		RegionAlignmentBytes: base.L1SizeBytes,
		BlockSizeBytes:       base.BlockSizeBytes,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "matrix benchmark configuration:", err)
		os.Exit(2)
	}

	results, err := benchrunner.RunSuite(base, []benchmark.Scenario{workload.Scenario}, benchrunner.ComparisonArchitectures())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	checks := benchrunner.ValidateResults(results)

	fmt.Printf("Matrix multiplication: %dx%d, requests=%d, product-checksum=%d\n\n", *dimension, *dimension, len(workload.Scenario.Requests), checksum(workload.Product))
	if *printValues {
		printMatrix("A", workload.Left, *dimension)
		printMatrix("B", workload.Right, *dimension)
		printMatrix("C=A*B", workload.Product, *dimension)
		fmt.Println()
	}
	benchrunner.PrintReport(os.Stdout, results, checks, *verboseChecks)

	if *csvPath != "" {
		if err := benchrunner.WriteCSVFile(*csvPath, results); err != nil {
			fmt.Fprintln(os.Stderr, "write CSV:", err)
			os.Exit(2)
		}
		fmt.Printf("\nCSV written to %s\n", *csvPath)
	}

	if *strict && hasFailedCheck(checks) {
		os.Exit(1)
	}
}

func checksum(values []int64) int64 {
	var total int64
	for _, value := range values {
		total += value
	}
	return total
}

func printMatrix(name string, values []int64, dimension int) {
	fmt.Printf("%s:\n", name)
	for row := 0; row < dimension; row++ {
		fmt.Printf("  %v\n", values[row*dimension:(row+1)*dimension])
	}
}

func hasFailedCheck(checks []benchrunner.Check) bool {
	for _, check := range checks {
		if !check.Passed {
			return true
		}
	}
	return false
}
