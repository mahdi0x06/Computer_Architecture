# Application Memory Test Benches

This project now includes two deterministic algorithmic workloads:

- square matrix multiplication;
- stable top-down merge sort.

Both run through Akita and compare exactly `l1-l2`, `full-fifo`, and
`full-lru`. Their CSV rows use the same schema as the existing test bench.

## Execution model

The project is a trace-driven cache simulator, not an instruction-set or
value-level CPU simulator. `MainMemory` persists block metadata such as block
address and dirty state; it does not store individual integer payloads.

Each application benchmark therefore performs two connected tasks:

1. Execute the real algorithm on Go integer slices and retain the verified
   numeric output.
2. Record every logical array access as a `model.Request` with an address,
   operation (`Read` or `Write`), and element size.

The resulting request stream follows the normal runtime path:

```text
algorithm memory trace
  -> testbench.RunSuite / RunCase
  -> simadapter.Adapter
  -> Akita request driver, ports, messages, and direct connection
  -> MemoryHierarchyExecutor
  -> System.Access
  -> L1 -> optional Victim Cache -> L2 -> Main Memory
  -> Akita completion event and response
  -> metrics and CSV
```

This makes cache statistics faithful to the algorithm's memory behavior while
remaining compatible with the established simulator design.

## Matrix multiplication

Implementation: `internal/benchmark/matrix.go`

The benchmark uses the conventional square, row-major algorithm:

```text
for i = 0..N-1
  for j = 0..N-1
    sum = 0
    for k = 0..N-1
      sum += A[i,k] * B[k,j]
    C[i,j] = sum
```

For every `k`, the generator records:

1. read `A[i,k]`;
2. read `B[k,j]`.

After the inner loop it records one write to `C[i,j]`. Therefore a square
matrix of dimension `N` always emits:

```text
N * N * (2*N + 1)
```

requests. The default `N=8` emits 1088 requests.

The three matrix regions are disjoint and separated by an aligned stride that
is a multiple of the L1 capacity:

```text
A base = base
B base = base + region stride
C base = base + 2 * region stride
```

This layout maps corresponding offsets to matching direct-mapped L1 indices,
creating repeatable conflict traffic that a Victim Cache can capture.

Run:

```bash
go run ./cmd/matrixbench
go run ./cmd/matrixbench -size 12 -element-size 4 -print-values \
  -csv matrix-results.csv
```

## Merge sort

Implementation: `internal/benchmark/mergesort.go`

The generator performs stable recursive merge sort. During each merge it
records:

1. both candidate reads for each comparison;
2. the selected-value write into scratch;
3. one source read and scratch write for every remaining value;
4. one scratch read and one source-array write for every copy-back value.

The default input contains 16 deterministic descending values. The source and
scratch arrays are disjoint and separated by an L1-capacity-aligned stride,
again making matching offsets conflict predictably in direct-mapped L1. The
default execution emits 288 memory requests.

Run:

```bash
go run ./cmd/mergesortbench
go run ./cmd/mergesortbench -length 32 -element-size 4 -print-values \
  -csv mergesort-results.csv
```

## CSV comparison

Each dedicated command uses `testbench.ComparisonArchitectures`, whose stable
order is:

```text
l1-l2
full-fifo
full-lru
```

The CSV contains request and cycle totals, average cycles, hit/miss counts and
rates for every cache level, Victim swaps, L2 reads/writes, and memory accesses.
The schema is unchanged, so existing analysis scripts can consume the new
rows.

Both workloads are also part of the complete suite and can be selected through
the generic commands:

```bash
go run ./cmd/testbench -trace matrix-multiply -matrix-size 8 -csv matrix.csv
go run ./cmd/testbench -trace merge-sort -merge-sort-length 16 -csv merge.csv
go run ./cmd/sim -topology full -trace matrix-multiply -victim-policy LRU
go run ./cmd/sim -topology full -trace merge-sort -victim-policy FIFO
```

## Verification

Focused tests verify:

- known matrix products;
- exact matrix request counts and read/write counts;
- merge-sort input/output correctness;
- deterministic request generation;
- sequential request IDs and access sizes;
- Akita equivalence to the synchronous functional reference;
- all accounting invariants; and
- exactly three CSV data rows in the required architecture order.

Run all checks with:

```bash
go test ./...
go test -race ./...
go vet ./...
```
