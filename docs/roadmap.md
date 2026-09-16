# Bootstrap status and next milestones

## Implemented vertical slice

1. CLI, UTF-8 lexer, positioned AST, recursive-descent/Pratt parser and recovery.
2. Declaration registration, scopes, namespaces/import lookup, type checking.
3. Typed backend-neutral IR, deterministic Go generation, native Hello World.
4. Functions, methods, control flow, class references, struct copies, properties.
5. Structural interface satisfaction with exact method signatures.
6. Built-in Result, explicit constructors, branch access and `?` propagation.
7. Goroutines, channels, send/receive, select and case-local receive bindings.
8. Built-in generic Array/List/Map and channel/Result type arguments.
9. Comment-preserving deterministic formatter and source diagnostics.
10. Small library facade, working examples, golden and integration tests.

## Deliberately deferred

User-defined generics have AST type parameters and recursive type references, but
are rejected by the checker. The next compiler milestone should add type parameter
symbols, substitution, instantiated member lookup, and a backend-neutral generic
representation, with constraint checking kept separate from method resolution.
Do not bypass the checker by forwarding arbitrary generic declarations into Go.

Other follow-ups: optional values and safe reference-field initialization;
namespace-qualified expression/type syntax and stricter import validation;
constructors; richer field initializers; collection lookup returning a presence
result; directional/closed channels; structured concurrency; source maps;
finer diagnostic codes; `gsharp test`; dependency resolution and lockfiles.

The library is deliberately tiny: Console, File.ReadAllText, Time.Sleep, Error,
collections, channels, and a fixed-response Http.Listen. Networking callbacks,
stream I/O, and production server configuration need a later library milestone.

There is no inheritance, overload resolution, implicit numeric widening, nullable
syntax, string interpolation, reflection, LINQ, async/await, or independent native
runtime. The formatter preserves comments but does not preserve their exact
original placement. Cross-compilation is not exposed by the CLI yet.
