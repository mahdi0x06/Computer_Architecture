package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"victimcacheproject/internal/benchmark"
	"victimcacheproject/internal/config"
	benchrunner "victimcacheproject/internal/testbench"
)

func main() {
	traceFlag := flag.String("trace", "all", "all|repeated|sequential|conflict|mixed|matrix-multiply|merge-sort")
	blocks := flag.Int("blocks", 4, "working-set/conflicting block count")
	repetitions := flag.Int("repetitions", 8, "workload repetitions")
	sequentialWords := flag.Int("sequential-words", 32, "number of consecutive words in the sequential trace")
	wordSize := flag.Uint64("word-size", 4, "word size in bytes for the sequential trace")
	matrixSize := flag.Int("matrix-size", benchmark.DefaultMatrixDimension, "square matrix dimension")
	mergeSortLength := flag.Int("merge-sort-length", benchmark.DefaultMergeSortLength, "merge-sort array length")
	policy := flag.String("victim-policy", benchrunner.PolicyBoth, "FIFO|LRU|BOTH")
	csvPath := flag.String("csv", "", "optional CSV output path")
	strict := flag.Bool("strict", true, "exit with status 1 if a validation check fails")
	verboseChecks := flag.Bool("verbose-checks", false, "print every accounting and behavioral assertion")
	flag.Parse()

	base := config.Default()
	suiteCfg := benchmark.SuiteConfig{
		BaseAddress:     0,
		BlockSizeBytes:  base.BlockSizeBytes,
		L1SizeBytes:     base.L1SizeBytes,
		L2SizeBytes:     base.L2SizeBytes,
		L2Associativity: base.L2Associativity,
		VictimEntries:   base.VictimEntries,
		NumberOfBlocks:  *blocks,
		Repetitions:     *repetitions,
		SequentialWords: *sequentialWords,
		WordSizeBytes:   *wordSize,
		AccessSizeBytes: 8,
		MatrixDimension: *matrixSize,
		MergeSortLength: *mergeSortLength,
	}

	var scenarios []benchmark.Scenario
	var err error
	if strings.EqualFold(strings.TrimSpace(*traceFlag), "all") {
		scenarios, err = benchmark.GenerateSuite(suiteCfg)
	} else {
		var kind benchmark.TraceKind
		kind, err = benchmark.ParseTraceKind(*traceFlag)
		if err == nil {
			var scenario benchmark.Scenario
			scenario, err = benchmark.GenerateScenario(kind, suiteCfg)
			scenarios = []benchmark.Scenario{scenario}
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "test-bench configuration:", err)
		os.Exit(2)
	}

	architectures, err := benchrunner.Architectures(*policy)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	results, err := benchrunner.RunSuite(base, scenarios, architectures)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	checks := benchrunner.ValidateResults(results)
	benchrunner.PrintReport(os.Stdout, results, checks, *verboseChecks)

	if *csvPath != "" {
		file, createErr := os.Create(*csvPath)
		if createErr != nil {
			fmt.Fprintln(os.Stderr, "create CSV:", createErr)
			os.Exit(2)
		}
		if writeErr := benchrunner.WriteCSV(file, results); writeErr != nil {
			_ = file.Close()
			fmt.Fprintln(os.Stderr, "write CSV:", writeErr)
			os.Exit(2)
		}
		if closeErr := file.Close(); closeErr != nil {
			fmt.Fprintln(os.Stderr, "close CSV:", closeErr)
			os.Exit(2)
		}
		fmt.Printf("\nCSV written to %s\n", *csvPath)
	}

	if *strict {
		for _, check := range checks {
			if !check.Passed {
				os.Exit(1)
			}
		}
	}
}
