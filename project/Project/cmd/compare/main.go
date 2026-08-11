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
	traceFlag := flag.String("trace", string(benchmark.TraceConflict), "repeated|sequential|conflict|mixed|matrix-multiply|merge-sort|all")
	blocks := flag.Int("blocks", 4, "working-set/conflicting block count")
	repetitions := flag.Int("repetitions", 8, "trace repetitions")
	sequentialWords := flag.Int("sequential-words", 32, "number of consecutive words in the sequential trace")
	wordSize := flag.Uint64("word-size", 4, "word size in bytes for the sequential trace")
	matrixSize := flag.Int("matrix-size", benchmark.DefaultMatrixDimension, "square matrix dimension")
	mergeSortLength := flag.Int("merge-sort-length", benchmark.DefaultMergeSortLength, "merge-sort array length")
	policy := flag.String("victim-policy", config.ReplacementFIFO, "FIFO|LRU|BOTH")
	flag.Parse()

	base := config.Default()
	suiteCfg := benchmark.SuiteConfig{
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
		fmt.Fprintln(os.Stderr, err)
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
	benchrunner.PrintReport(os.Stdout, results, benchrunner.ValidateResults(results), false)
}
