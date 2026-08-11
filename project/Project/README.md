# Victim Cache Project 6

An Akita-driven simulator with a deterministic functional memory-hierarchy
core. It supports four memory hierarchies:

- `memory`: CPU -> Main Memory
- `l1`: CPU -> L1 -> Main Memory
- `l1-l2`: CPU -> L1 -> L2 -> Main Memory
- `full`: CPU -> L1 -> Victim Cache -> L2 -> Main Memory

All normal CLI commands use the Akita execution path. There is no separate
"Akita mode": `cmd/sim` creates the adapter directly, while `cmd/testbench` and
`cmd/compare` reach it through `testbench.RunCase`.

The runtime call path is:

```text
benchmark requests
  -> MemoryRequestDriver (Akita component)
  -> accessRequestMsg through an Akita port and DirectConnection
  -> MemoryHierarchyExecutor (Akita component)
  -> System.Access exactly once
  -> completeAccessEvent scheduled after the calculated service latency
  -> accessResponseMsg through the Akita connection
  -> MemoryRequestDriver responses and project reports
```

## Run one topology and one workload

The simulator supports six deterministic traces:

- `repeated`: proves L1 warm-up and L1 hits
- `sequential`: reads consecutive 4-byte words one by one, demonstrating spatial locality inside 64-byte blocks
- `conflict`: forces direct-mapped L1 thrashing and measures Victim Cache benefit
- `mixed`: runs 1312 deterministic requests, exercises every hierarchy level, and creates a clear FIFO-versus-LRU Victim Cache difference
- `matrix-multiply`: runs a real square matrix multiplication and records every operand read and result write
- `merge-sort`: runs a real top-down merge sort and records array/scratch reads and writes

Examples:

```bash
go run ./cmd/sim -topology l1 -trace repeated
go run ./cmd/sim -topology l1-l2 -trace sequential -sequential-words 32 -word-size 4
go run ./cmd/sim -topology l1-l2 -trace conflict
go run ./cmd/sim -topology full -trace mixed -victim=true -victim-policy=FIFO
go run ./cmd/sim -topology full -trace matrix-multiply -matrix-size 8 -victim-policy=LRU
go run ./cmd/sim -topology full -trace merge-sort -merge-sort-length 16 -victim-policy=FIFO
```


For the default sequential trace, the addresses are `0, 4, 8, ..., 124`.
A 64-byte block contains sixteen 4-byte words, so 32 requests touch exactly
two blocks and produce 30 L1 hits plus 2 L1 misses.

For the default mixed trace, FIFO records 60 Victim hits and 11922 cycles, while LRU records 188 Victim hits and 10386 cycles. The workload deliberately keeps four hot blocks recent before overflowing the eight-entry Victim Cache, so LRU retains them and FIFO evicts them by arrival order.

Trace controls:

```bash
go run ./cmd/sim -topology full -trace conflict -blocks 4 -repetitions 20
go run ./cmd/sim -topology l1-l2 -trace sequential -sequential-words 64 -word-size 4
```

## Complete final test bench

Run every workload on every hierarchy, test both FIFO and LRU Victim Cache policies, and execute automatic accounting and behavioral checks:

```bash
go run ./cmd/testbench
```

Run one workload only:

```bash
go run ./cmd/testbench -trace mixed
```

Write the measurements to CSV for the final report:

```bash
go run ./cmd/testbench -csv results.csv
```

Useful options:

```bash
go run ./cmd/testbench -blocks 4 -repetitions 20 -victim-policy BOTH
go run ./cmd/testbench -trace conflict -victim-policy FIFO
go run ./cmd/testbench -strict=false
go run ./cmd/testbench -verbose-checks
```

`-strict=true` is the default. The command exits with status 1 when any validation check fails, which makes it suitable for CI.

## Application test benches

Two dedicated commands execute real algorithms and always compare the three
requested cache architectures in this exact order:

1. `l1-l2`
2. `full-fifo`
3. `full-lru`

Matrix multiplication:

```bash
go run ./cmd/matrixbench
go run ./cmd/matrixbench -size 12 -print-values -csv matrix-results.csv
```

Merge sort:

```bash
go run ./cmd/mergesortbench
go run ./cmd/mergesortbench -length 32 -print-values -csv mergesort-results.csv
```

The default CSV names are `matrix-results.csv` and
`mergesort-results.csv`. Each file contains one row for each of the three
architectures and preserves the existing CSV schema.

The functional memory model stores cache-block metadata rather than numeric
payloads. Therefore each benchmark computes and verifies its numeric result in
Go while emitting every logical read/write as a simulator-independent request.
Akita then transports, schedules, and completes that exact memory trace through
the selected hierarchy. See [APPLICATION_BENCHMARKS.md](APPLICATION_BENCHMARKS.md)
for the full algorithms and address mapping.

## Compare command

`cmd/compare` remains as a shorter alias for summary reporting:

```bash
go run ./cmd/compare -trace conflict
go run ./cmd/compare -trace all -victim-policy BOTH
```

## Verification

```bash
go test ./...
go test -race ./...
go vet ./...
```

## Akita execution

Every user-facing command runs requests through Akita v4.9.0. The
`internal/simadapter` package builds a serial Akita engine, a request-driver
component, a hierarchy-executor component, typed request/response messages,
an Akita direct connection, and one completion event per memory access.

The cache behavior under `internal/system` remains the functional correctness
oracle. Requests are issued one at a time so the existing cache state,
statistics, reported cycles, command output, validation checks, and CSV files
remain unchanged. Akita owns message delivery and simulated event ordering;
the functional core owns the established hierarchy semantics.

The current integration uses two Akita components around the complete
functional hierarchy. L1, Victim Cache, L2, and Main Memory are concrete cache
objects called inside `System.Access`; they are not yet separate concurrent
Akita components. Consequently, `accessRequestMsg` and `accessResponseMsg` are
real Akita messages, while lookup, insertion, eviction, and swap are synchronous
functional operations inside the hierarchy executor. The response becomes
visible to the driver only when Akita executes the scheduled completion event.

At 1 GHz the executor converts `Response.LatencyCycles` to Akita time with
`Freq.NCyclesLater`. The printed `TotalCycles` remains the sum of established
cache-service latencies and intentionally excludes internal connection ticks,
preserving all previous CLI and CSV results.

See [AKITA_INTEGRATION.md](AKITA_INTEGRATION.md) for the complete component,
message, timing, execution, and compatibility design.
