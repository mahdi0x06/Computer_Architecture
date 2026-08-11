# Simulation adapter

`simadapter` is the Akita runtime boundary used by every command.

It creates:

- an Akita serial engine;
- a request-driver component and port;
- a hierarchy-executor component and port;
- typed Akita request and response messages;
- an Akita direct connection; and
- scheduled access-completion events.

The hierarchy executor calls the unchanged functional `System.Access` exactly
once per request. The driver waits for that response before sending the next
request, preserving all pre-integration results while making Akita responsible
for request transport and event scheduling.

## File responsibilities

- `adapter.go`: validates the supplied `System`; creates a fresh Akita graph in
  `Run`; plugs both ports into the direct connection; starts and runs the
  engine; checks errors and response count; copies responses to the adapter.
- `components.go`: implements the two `sim.Component` endpoints, the start
  event, the completion event, serial issue, response validation, and Akita-time
  scheduling.
- `messages.go`: implements `sim.Msg` and `sim.Rsp`, assigns Akita message IDs,
  records source/destination ports and traffic metadata, and correlates each
  response with its request through `RspTo`.
- `adapter_test.go`: compares the Akita path with `System.Run` for every
  workload/topology combination.

The complete suite includes repeated, sequential, conflict, mixed, matrix
multiplication, and merge sort. The application generators compute real
numeric values while recording each logical read/write; this adapter executes
those requests through the same Akita path as every other workload.

## Request lifecycle

1. `requestDriver.Start` schedules `startDriverEvent` at time zero.
2. `requestDriver.Handle` calls `sendNext`, which sends one
   `accessRequestMsg` and marks it outstanding.
3. The Akita `DirectConnection` delivers the message to
   `hierarchyExecutor.NotifyRecv`.
4. The executor calls `System.Access` exactly once and receives a functional
   response containing its serving level and latency.
5. At 1 GHz, `Freq.NCyclesLater` converts that cycle count into an Akita
   completion time and schedules `completeAccessEvent`.
6. When the event runs, the executor sends `accessResponseMsg` through its
   Akita port.
7. The driver verifies `RspTo` and `RequestID`, stores the response, and only
   then injects the next request.
8. After the final response, no new event is scheduled, the engine queue drains,
   and `Adapter.Run` returns.

## Responsibility boundary

Akita genuinely owns the engine, components, ports, message transport,
connection ticks, event ordering, and response-completion scheduling. The
functional `System` owns L1/Victim/L2/Main-Memory lookup, insertion, swap,
replacement, dirty write-back, statistics, and established latency selection.

The cache levels are not currently separate Akita components, and internal
lookup/eviction operations are not separate Akita messages. They execute
atomically inside one `System.Access` call after the request message arrives.
This boundary is intentional: it adds Akita capabilities without changing the
CLI output, CSV schema, hit/miss counts, replacement order, or cycle totals.

`Adapter.Build` performs validation and marks the adapter ready. The fresh
Akita engine and components are constructed by `Adapter.Run`, so repeated test
cases cannot leak engine time or queued events into one another.

See the repository root file `AKITA_INTEGRATION.md` for the complete design.
