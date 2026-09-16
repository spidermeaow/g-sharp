# Runtime implementation

`compiler/codegen/golang/runtime.go` contains the self-contained runtime emitted
with generated programs. It implements typed Result values, reference-oriented
lists, Error, and the standard-library facade using Go's standard library.
Generated executables have no dependency on a separate G-Sharp installation.
Scheduling, channels, memory management, and networking are supplied by Go.
