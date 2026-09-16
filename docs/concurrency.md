# Concurrency

`go Function(args);` evaluates the receiver and all arguments on the launching
goroutine, in source order, before starting the call. `go { ... };` captures local
storage. Concurrent mutation requires synchronization. A goroutine block is a
separate void-returning function: `return;` exits that block, and `?` cannot return
an error through its caller. A surrounding loop's break/continue cannot escape
from the block.

`chan<T>()` is unbuffered; `chan<T>(n)` has capacity n. Sends and receives block
until their communication can proceed. An unbuffered exchange synchronizes both
participants. Buffered channels preserve FIFO ordering. Negative capacities are
invalid. Channel direction, close, two-value receive, cancellation, and structured
task groups are future work. `foreach` over a channel waits for further values;
without close support, use an explicit bounded protocol or break condition.

Select evaluates channel operands and send values once, in source order, before
waiting. One ready case is chosen using Go's ready-case selection. Default runs
only when no communication can proceed. `case name = <-channel:` declares a new
case-local variable. Cases do not fall through; break exits the selection.

Channel communication is the bootstrap synchronization primitive. Data races are
program errors. Go's memory model supplies the initial operational ordering:
goroutine creation, sends, and corresponding receives establish synchronization;
sleeping does not. The main entry returning terminates the process without
joining background work. Programs must arrange completion using channels.

The [producer-consumer example](../examples/producer-consumer/main.gsh) transports
Result values on a channel. Propagation occurs inside a named Result-returning
function, keeping error handling valid in the worker goroutine.
