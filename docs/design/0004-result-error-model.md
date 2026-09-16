# ADR 0004: Typed errors and propagation

**Context:** Expected errors must be explicit without requiring repetitive tests.

**Decision:** Built-in Result<T,E> is a tagged value. Constructors use an expected
type. Postfix ? unwraps success or returns the error from a compatible enclosing
Result-returning function. The success types may differ, but E must match exactly.
The first entry point is void Main; it must inspect Results explicitly.

**Alternatives:** Exceptions hide control flow. Ignoring errors in void contexts
would weaken the type system. A panic-based implementation of ? could incorrectly
cross goroutine/function boundaries.

**Consequences:** Go generation emits explicit branch-and-return code with
temporaries. Short-circuit evaluation must guard propagation. Goroutine blocks
have their own void return context, so they send Results or call named helpers.
