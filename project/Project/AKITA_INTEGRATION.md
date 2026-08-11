# Akita Integration

## Purpose

The project uses Akita v4.9.0 for simulation orchestration without changing
the established functional behavior of the memory hierarchy.

The integration has two equally important requirements:

1. Exercise real Akita capabilities: engine, components, ports, messages,
   responses, a connection, and scheduled events.
2. Preserve the existing request order, cache contents, replacement decisions,
   hit/miss counters, latency accounting, reports, CSV schema, CLI flags, and
   command output.

The implementation therefore wraps the existing functional `System` rather
than rewriting independently validated cache algorithms.

## Runtime architecture

```text
benchmark.Scenario.Requests
            |
            v
  MemoryRequestDriver (Akita component)
            |
            | accessRequestMsg through sim.Port
            v
  MemoryHierarchyConnection (Akita direct connection)
            |
            v
  MemoryHierarchyExecutor (Akita component)
            |
            | exactly one System.Access call
            v
  L1 -> Victim Cache -> L2 -> Main Memory
            |
            | accessResponseMsg after a scheduled completion event
            v
  MemoryRequestDriver.Responses
```

The implementation lives in `internal/simadapter`:

- `adapter.go` builds and runs the Akita graph.
- `components.go` defines the two Akita components and completion events.
- `messages.go` defines the typed Akita request and response messages.
- `adapter_test.go` compares Akita execution with the functional oracle.

The user-facing call hierarchy is:

```text
cmd/sim.main
  -> simadapter.New -> SetRequests -> Build -> Run

cmd/testbench.main or cmd/compare.main
  -> testbench.RunSuite -> RunCase
  -> simadapter.New -> SetRequests -> Build -> Run
```

`Build` validates the functional hierarchy. `Run` creates the fresh Akita
engine, components, ports, and direct connection, executes the event queue, and
copies the completed responses. Therefore every reported CLI and CSV result is
produced after successful Akita-engine execution.

## Akita capabilities used

### Serial engine

Each adapter run creates a fresh `sim.SerialEngine`:

```go
engine := sim.NewSerialEngine()
```

The engine processes the start event, direct-connection ticks, access
completion events, and message delivery until the event queues are empty.

### Components

Both runtime endpoints embed `sim.ComponentBase` and satisfy
`sim.Component`:

- `requestDriver`
- `hierarchyExecutor`

Each component implements:

- `Handle(sim.Event) error`
- `NotifyRecv(sim.Port)`
- `NotifyPortFree(sim.Port)`

### Ports

Each component owns one Akita port with one incoming and one outgoing slot.
The request driver is deliberately limited to one outstanding request, so this
capacity is sufficient and also enforces the compatibility requirement.

### Messages

`accessRequestMsg` implements `sim.Msg` and carries `model.Request`.

`accessResponseMsg` implements `sim.Rsp` and carries `model.Response`. Its
`RspTo` field correlates the response with the Akita ID of the original
request message. The functional request ID is checked independently.

Both message types implement `Clone` by preserving their functional payload
and generating a new Akita message ID.

### Direct connection

The two component ports are plugged into an Akita
`sim/directconnection.Comp`. It transports messages using Akita's port
buffers, destination metadata, receive notifications, and secondary ticks.

### Scheduled completion events

The hierarchy executor calls `System.Access` when an Akita request arrives.
The returned response includes the already-established latency in cycles. The
executor converts those cycles to Akita time at 1 GHz:

```go
completionTime := frequency.NCyclesLater(
    int(response.LatencyCycles),
    engine.CurrentTime(),
)
```

It then schedules a `completeAccessEvent`. The response is not placed on the
Akita port until that event runs.

`System.Access` performs its functional state transition synchronously when the
Akita request arrives. Akita models the externally observable completion time:
the driver cannot receive the response or issue the next request before the
completion event. This is the compatibility model used by the project, not a
claim that every internal cache operation is a separate event.

## Why the functional System remains intact

The cache package has extensive deterministic tests for:

- direct-mapped L1 behavior;
- fully-associative Victim Cache behavior;
- FIFO and LRU Victim replacement;
- set-associative FIFO L2 behavior;
- dirty write-back;
- hierarchy installation and eviction flow; and
- exact per-workload counters and cycle totals.

Replacing those algorithms while adding Akita would combine two independent
changes and risk changing the project results. Instead, each Akita request
invokes `System.Access` exactly once. Akita controls when requests and
responses move; `System` controls what the memory hierarchy does.

The only runtime messages in the current adapter are `accessRequestMsg` and
`accessResponseMsg`. L1 lookup, Victim lookup/swap, L2 lookup, block fill, and
eviction forwarding are ordinary function calls within `System.Access`. The
state-machine and sequence diagrams in the report describe these logical
internal operations; they must not be interpreted as separate network messages
between independent Akita cache components.

`System.Run` remains available as the synchronous correctness oracle. It is
used by integration tests to prove that the Akita path returns identical
responses and statistics for every workload and topology.

## Preserved execution semantics

The driver waits for each response before injecting the next request:

```text
send request 1 -> wait -> receive response 1
send request 2 -> wait -> receive response 2
...
```

This preserves the original `System.Run` ordering. Enabling multiple in-flight
requests would introduce queueing, overlap, and possibly different cache-state
observation; that would be a new simulator feature and is intentionally not
part of this compatibility integration.

The reported `TotalCycles` remains the sum of the configured hierarchy service
latencies. Akita's direct connection may schedule internal transport ticks;
those implementation ticks are not added to the established performance
metric or printed output.

## Commands

All existing commands and flags are unchanged:

```bash
go run ./cmd/sim
go run ./cmd/sim -topology full -trace conflict
go run ./cmd/sim -topology full -trace mixed -victim-policy=LRU

go run ./cmd/compare -trace all -victim-policy BOTH

go run ./cmd/testbench
go run ./cmd/testbench -trace mixed
go run ./cmd/testbench -csv results.csv
go run ./cmd/testbench -strict=false -verbose-checks

go run ./cmd/matrixbench -size 8 -csv matrix-results.csv
go run ./cmd/mergesortbench -length 16 -csv mergesort-results.csv
```

`cmd/sim` constructs `simadapter.Adapter` directly. `cmd/testbench` and
`cmd/compare` both reach the adapter through `testbench.RunCase`.

There is no separate Akita-only command or flag. The normal commands are the
Akita execution path.

## Programmatic use

```go
simulator := system.New(cfg)
adapter := simadapter.New(simulator)
adapter.SetRequests(requests)

if err := adapter.Build(); err != nil {
    return err
}
if err := adapter.Run(); err != nil {
    return err
}

responses := adapter.Responses
stats := simulator.Stats
```

`SetRequests` copies the supplied slice. `Build` validates the functional
hierarchy. `Run` creates a fresh Akita engine and graph, executes the trace,
checks response count and correlation, and copies completed responses back to
the adapter.

## Dependency and toolchain

The project pins:

```text
github.com/sarchlab/akita/v4 v4.9.0
```

Akita v4.9.0 requires Go 1.24, so this module declares `go 1.24.0`.

Install dependencies with:

```bash
go mod download
```

## Verification

Run:

```bash
go test ./...
go test -race ./...
go vet ./...
```

The simadapter integration test runs all six workloads against:

- memory only;
- L1 only;
- L1 and L2;
- full hierarchy with FIFO; and
- full hierarchy with LRU.

For all 30 combinations it compares:

- every `model.Response`;
- every `metrics.Stats` field; and
- successful Akita engine completion.

The CSV schema and all pre-existing per-trace measurements remain unchanged.
The complete suite now appends the two requested application workloads, and
their dedicated commands emit exactly three comparison rows each.

## Safe future extensions

The current design is the compatibility layer. Future timing studies may add:

- multiple outstanding requests;
- separate L1, Victim, L2, and memory Akita components;
- link latency and bandwidth;
- finite queues and backpressure;
- MSHRs and request coalescing; or
- a parallel Akita engine.

Each of those changes alters timing or concurrency semantics. They should be
introduced behind new configuration options and compared against this current
serial Akita path so existing commands and reports remain stable.
