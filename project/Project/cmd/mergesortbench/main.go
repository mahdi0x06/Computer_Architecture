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
	length := flag.Int("length", benchmark.DefaultMergeSortLength, "number of array elements")
	elementSize := flag.Uint64("element-size", 4, "array element size in bytes")
	csvPath := flag.String("csv", "mergesort-results.csv", "CSV output path; empty disables CSV output")
	strict := flag.Bool("strict", true, "exit with status 1 if a validation check fails")
	verboseChecks := flag.Bool("verbose-checks", false, "print every validation assertion")
	printValues := flag.Bool("print-values", false, "print the input and sorted arrays")
	flag.Parse()

	base := config.Default()
	workload, err := benchmark.GenerateMergeSortWorkload(benchmark.MergeSortConfig{
		BaseAddress:          0,
		Length:               *length,
		ElementSizeBytes:     *elementSize,
		RegionAlignmentBytes: base.L1SizeBytes,
		BlockSizeBytes:       base.BlockSizeBytes,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "merge-sort benchmark configuration:", err)
		os.Exit(2)
	}
	if !isSorted(workload.Sorted) {
		fmt.Fprintln(os.Stderr, "merge-sort benchmark produced an unsorted result")
		os.Exit(1)
	}

	results, err := benchrunner.RunSuite(base, []benchmark.Scenario{workload.Scenario}, benchrunner.ComparisonArchitectures())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	checks := benchrunner.ValidateResults(results)

	fmt.Printf("Merge sort: elements=%d, requests=%d, sorted=true\n\n", *length, len(workload.Scenario.Requests))
	if *printValues {
		fmt.Printf("input:  %v\n", workload.Input)
		fmt.Printf("sorted: %v\n\n", workload.Sorted)
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

func isSorted(values []int64) bool {
	for index := 1; index < len(values); index++ {
		if values[index-1] > values[index] {
			return false
		}
	}
	return true
}

func hasFailedCheck(checks []benchrunner.Check) bool {
	for _, check := range checks {
		if !check.Passed {
			return true
		}
	}
	return false
}
