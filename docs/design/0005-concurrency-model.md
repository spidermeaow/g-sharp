# ADR 0005: Channels and goroutines

**Context:** Simple concurrency and native deployment are defining goals.

**Decision:** go, typed channels, and select are core language constructs backed
by Go's runtime. Arguments evaluate before spawning; anonymous go blocks capture
lexical storage. Select bindings are case-local. Program exit does not join work.

**Alternatives:** Task/async/await would add a second execution model. A custom
scheduler would consume effort without improving bootstrap language semantics.

**Consequences:** Programs synchronize completion themselves. Higher-level task
groups can be added without changing channel primitives. Closed channels,
directionality, cancellation, and safe shutdown protocols are future work.
