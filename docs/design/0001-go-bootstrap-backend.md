# ADR 0001: Go bootstrap backend

**Context:** A useful native compiler needs memory management, scheduling, and
portable deployment before a custom backend is practical.

**Decision:** Implement the compiler in Go and emit Go from typed, backend-neutral
IR. The parser and checker never construct Go AST nodes. Generated files remain
under a compiler-owned build directory. The host Go toolchain builds executables.

**Alternatives:** A custom native backend or LLVM would greatly expand the initial
implementation and runtime work. Textual source substitution would undermine
diagnostics and semantic independence.

**Consequences:** Users need Go to compile, but not to run binaries. Go codegen
must preserve G-Sharp evaluation order and type rules. A future backend consumes
the same semantic operations and supplies their runtime contracts.
