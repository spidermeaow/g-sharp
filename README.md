# G-Sharp

G-Sharp is a statically typed compiled programming language combining C#-inspired
developer ergonomics with Go-inspired simplicity and concurrency.

This repository contains a usable **0.1 bootstrap compiler**, implemented in Go.
It checks G-Sharp source, constructs typed IR, generates Go, and invokes the Go
toolchain to produce a native executable. It uses Go's runtime, GC, scheduler,
and channels; it does not yet have an independent runtime or machine-code backend.

## Start

For complete installation instructions, Windows and macOS/Linux walkthroughs,
command examples, and troubleshooting, read the [Usage Guide](docs/usage.md).

Install Go 1.26.6 or newer, then:

```sh
go build -o gsharp ./cmd/gsharp
./gsharp new console Hello
cd Hello
../gsharp run
```

On Windows build with `go build -o gsharp.exe ./cmd/gsharp`, then use
`.\gsharp.exe` and `..\gsharp.exe` for the commands above.

Output: `Hello from G-Sharp!`

```csharp
class Program {
    static void Main() {
        var answers = chan<int>();
        go {
            answers <- 42;
        };
        Console.WriteLine(<-answers);
    }
}
```

## Commands

| Command | Behavior |
| --- | --- |
| `gsharp init [MODULE]` | Create a manifest and, for an empty project, a starter program |
| `gsharp new console NAME` | Create a new project directory |
| `gsharp check [PATH]` | Parse and type-check without invoking Go |
| `gsharp build [PATH] [-o FILE] [--emit-go]` | Produce a native executable; optionally print the generated source path |
| `gsharp run [PATH] [-- ARGS...]` | Build and execute with inherited standard streams |
| `gsharp fmt [PATH] [--check]` | Format source, or verify formatting without writing |
| `gsharp clean` | Remove the current project's compiler-owned `.gsharp` directory |
| `gsharp version` | Print the compiler version |

PATH is a project directory or single `.gsh` file. Build products live under
`.gsharp/generated` and `.gsharp/bin`. Generated Go is retained for debugging.
Source discovery is deterministic: `src/**/*.gsh`, or root `*.gsh` without `src`.

## Implemented

Primitive types, local inference and constants; functions and methods; classes,
structs, auto-properties and enums; implicit interfaces; lexical scopes and local
namespaces; if/else, for, foreach, switch; Array/List/Map; Result and checked `?`;
goroutines, typed channels, send/receive and select. See [examples](examples).

The [producer-consumer example](examples/producer-consumer/main.gsh) demonstrates
interfaces, typed errors, and concurrency together, and prints `42`.

User-defined generics, nullable types, external dependency fetching, `gsharp test`,
constructors, interpolation, and general HTTP handlers are **not implemented**.
Unsupported generic declarations produce frontend diagnostics. The `generics`
example uses only implemented built-in generic types. Reference fields cannot be
implicitly null; the initial field-initialization rules are deliberately limited.
This is an early compiler, with no compatibility or production-stability promise.

## Validate and contribute

```sh
go fmt ./...
go vet ./...
go test ./...
go run ./cmd/gsharp run examples/producer-consumer
go run ./cmd/gsharp build examples/hello-world --emit-go
```

Tests cover lexing, parsing/recovery, semantic rejection, formatting, manifests,
golden Go output, CLI workflows, and execution through the real Go toolchain.
All example programs are compiled in integration tests; the HTTP server is built
but not started by that test.

Start with the [language specification](docs/language-spec.md),
[grammar](docs/grammar.md), [architecture](docs/architecture.md),
[concurrency rules](docs/concurrency.md), [diagnostics](docs/errors.md), and
[roadmap](docs/roadmap.md).
