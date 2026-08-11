package testbench

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"

	"victimcacheproject/internal/benchmark"
)

func PrintReport(w io.Writer, results []Result, checks []Check, verboseChecks bool) {
	fmt.Fprintln(w, "Victim Cache Project 6 — Complete Test Bench")
	fmt.Fprintln(w)

	for _, kind := range benchmark.AllTraceKinds() {
		var traceResults []Result
		for _, result := range results {
			if result.Scenario.Kind == kind {
				traceResults = append(traceResults, result)
			}
		}
		if len(traceResults) == 0 {
			continue
		}

		fmt.Fprintf(w, "TRACE: %s — %s\n", kind, traceResults[0].Scenario.Name)
		fmt.Fprintf(w, "%s\n", traceResults[0].Scenario.Description)
		fmt.Fprintf(w, "%-11s %6s %8s %7s %13s %8s %13s %8s %13s %8s %6s %6s %6s\n",
			"ARCH", "REQ", "CYCLES", "AVG", "L1 H/M", "L1 HR", "VC H/M", "VC HR", "L2 H/M", "L2 HR", "MEM", "L2-WR", "SWAP")
		for _, result := range traceResults {
			s := result.Stats
			fmt.Fprintf(w, "%-11s %6d %8d %7.2f %6d/%-6d %8.4f %6d/%-6d %8.4f %6d/%-6d %8.4f %6d %6d %6d\n",
				result.Architecture.Name,
				s.TotalRequests,
				s.TotalCycles,
				s.AverageCyclesPerRequest(),
				s.L1Hits, s.L1Misses, s.L1HitRate(),
				s.VictimHits, s.VictimMisses, s.VictimHitRate(),
				s.L2Hits, s.L2Misses, s.L2HitRate(),
				s.MemoryAccesses, s.L2WriteRequests, s.VictimSwaps,
			)
		}
		fmt.Fprintln(w)
	}

	passed, failed := 0, 0
	for _, check := range checks {
		if check.Passed {
			passed++
		} else {
			failed++
		}
	}

	fmt.Fprintln(w, "VALIDATION CHECKS")
	if verboseChecks {
		for _, check := range checks {
			status := "PASS"
			if !check.Passed {
				status = "FAIL"
			}
			fmt.Fprintf(w, "[%s] %s — %s\n", status, check.Name, check.Detail)
		}
	} else if failed == 0 {
		fmt.Fprintln(w, "All accounting and behavioral checks passed. Use -verbose-checks to list each assertion.")
	} else {
		for _, check := range checks {
			if !check.Passed {
				fmt.Fprintf(w, "[FAIL] %s — %s\n", check.Name, check.Detail)
			}
		}
	}
	fmt.Fprintf(w, "Checks: %d passed, %d failed\n", passed, failed)
}

func WriteCSV(w io.Writer, results []Result) error {
	writer := csv.NewWriter(w)

	header := []string{
		"trace", "scenario", "architecture", "requests", "cycles", "average_cycles",
		"l1_hits", "l1_misses", "l1_hit_rate",
		"victim_hits", "victim_misses", "victim_hit_rate", "victim_swaps",
		"l2_hits", "l2_misses", "l2_hit_rate", "l2_reads", "l2_writes",
		"memory_accesses",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, result := range results {
		s := result.Stats
		row := []string{
			string(result.Scenario.Kind),
			result.Scenario.Name,
			result.Architecture.Name,
			strconv.FormatUint(s.TotalRequests, 10),
			strconv.FormatUint(s.TotalCycles, 10),
			fmt.Sprintf("%.6f", s.AverageCyclesPerRequest()),
			strconv.FormatUint(s.L1Hits, 10),
			strconv.FormatUint(s.L1Misses, 10),
			fmt.Sprintf("%.6f", s.L1HitRate()),
			strconv.FormatUint(s.VictimHits, 10),
			strconv.FormatUint(s.VictimMisses, 10),
			fmt.Sprintf("%.6f", s.VictimHitRate()),
			strconv.FormatUint(s.VictimSwaps, 10),
			strconv.FormatUint(s.L2Hits, 10),
			strconv.FormatUint(s.L2Misses, 10),
			fmt.Sprintf("%.6f", s.L2HitRate()),
			strconv.FormatUint(s.L2ReadRequests, 10),
			strconv.FormatUint(s.L2WriteRequests, 10),
			strconv.FormatUint(s.MemoryAccesses, 10),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

// WriteCSVFile writes one complete report and propagates both write and close
// errors. It is shared by the dedicated application benchmark commands.
func WriteCSVFile(path string, results []Result) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := WriteCSV(file, results); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
