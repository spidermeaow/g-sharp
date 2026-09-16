# ADR 0006: Generic boundaries

**Context:** Collections, channels, and typed errors require type arguments from
the start, but unchecked user generic definitions would compromise soundness.

**Decision:** Implement built-in generic types completely in this slice. Parse
user type/function parameter lists but reject them with an explicit diagnostic.
AST TypeRef and semantic Type carry recursive arguments independent of Go.

**Alternatives:** Blindly passing declarations to Go would delegate type errors
to the backend. Omitting generic structure would require replacing the type model.

**Consequences:** The sample generic Processor<T> becomes Processor for this
milestone. Future substitution and constraints belong in the checker, with tests
for instantiated interface satisfaction before user generics become supported.
