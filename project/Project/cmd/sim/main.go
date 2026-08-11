package main

import (
	"flag"
	"fmt"
	"os"

	"victimcacheproject/internal/benchmark"
	"victimcacheproject/internal/config"
	"victimcacheproject/internal/simadapter"
	"victimcacheproject/internal/system"
)

func main() {
	topology := flag.String("topology", config.TopologyFull, "memory|l1|l1-l2|full")
	traceName := flag.String("trace", string(benchmark.TraceConflict), "repeated|sequential|conflict|mixed|matrix-multiply|merge-sort")
	victim := flag.Bool("victim", true, "enable victim cache in full topology")
	blocks := flag.Int("blocks", 4, "working-set/conflicting block count")
	repetitions := flag.Int("repetitions", 8, "trace repetitions")
	sequentialWords := flag.Int("sequential-words", 32, "number of consecutive words in the sequential trace")
	wordSize := flag.Uint64("word-size", 4, "word size in bytes for the sequential trace")
	matrixSize := flag.Int("matrix-size", benchmark.DefaultMatrixDimension, "square matrix dimension")
	mergeSortLength := flag.Int("merge-sort-length", benchmark.DefaultMergeSortLength, "merge-sort array length")
	policy := flag.String("victim-policy", config.ReplacementFIFO, "FIFO|LRU")
	flag.Parse()

	cfg := config.Default()
	cfg.Topology = *topology
	cfg.VictimEnabled = *victim
	cfg.VictimPolicy = *policy
	simulator := system.New(cfg)
	if err := simulator.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	kind, err := benchmark.ParseTraceKind(*traceName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	scenario, err := benchmark.GenerateScenario(kind, benchmark.SuiteConfig{
		BaseAddress:     0,
		BlockSizeBytes:  cfg.BlockSizeBytes,
		L1SizeBytes:     cfg.L1SizeBytes,
		L2SizeBytes:     cfg.L2SizeBytes,
		L2Associativity: cfg.L2Associativity,
		VictimEntries:   cfg.VictimEntries,
		NumberOfBlocks:  *blocks,
		Repetitions:     *repetitions,
		SequentialWords: *sequentialWords,
		WordSizeBytes:   *wordSize,
		AccessSizeBytes: 8,
		MatrixDimension: *matrixSize,
		MergeSortLength: *mergeSortLength,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	adapter := simadapter.New(simulator)
	adapter.SetRequests(scenario.Requests)
	if err := adapter.Build(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := adapter.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	stats := simulator.Stats
	fmt.Println("Victim Cache Project 6")
	fmt.Printf("trace=%s scenario=%q\n", scenario.Kind, scenario.Name)
	fmt.Println(simulator.Summary())
	fmt.Printf("L1:     hits=%d misses=%d hit-rate=%.4f\n", stats.L1Hits, stats.L1Misses, stats.L1HitRate())
	fmt.Printf("Victim: hits=%d misses=%d hit-rate=%.4f swaps=%d\n", stats.VictimHits, stats.VictimMisses, stats.VictimHitRate(), stats.VictimSwaps)
	fmt.Printf("L2:     hits=%d misses=%d hit-rate=%.4f reads=%d writes=%d\n", stats.L2Hits, stats.L2Misses, stats.L2HitRate(), stats.L2ReadRequests, stats.L2WriteRequests)
	fmt.Printf("Memory: accesses=%d\n", stats.MemoryAccesses)
	fmt.Printf("Average cycles/request: %.4f\n", stats.AverageCyclesPerRequest())
}
