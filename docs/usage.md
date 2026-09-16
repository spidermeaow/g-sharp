# G-Sharp Usage Guide

G-Sharp is a statically typed compiled language with C#-inspired syntax and
Go-style concurrency. This guide covers the current bootstrap compiler.
The compiler validates `.gsh` source, lowers it to Go, and builds a native
executable using the Go toolchain.

## 1. Prerequisites

- Go **1.26.6 or newer**, available on your `PATH`.
- Git, to clone the repository.
- A terminal: PowerShell on Windows, or a shell on macOS/Linux.

Check your tools:

```sh
go version
git --version
```

Go is required to build G-Sharp programs. A generated native executable does not
require Go or G-Sharp to be installed on the destination machine. Build and run
on the same operating system and architecture; the CLI does not yet expose
cross-compilation options.

## 2. Download and build the compiler

```sh
git clone https://github.com/spidermeaow/g-sharp.git
cd g-sharp
```

### Windows / PowerShell

```powershell
go build -o gsharp.exe ./cmd/gsharp
.\gsharp.exe version
.\gsharp.exe run examples/hello-world
```

### macOS / Linux

```sh
go build -o gsharp ./cmd/gsharp
./gsharp version
./gsharp run examples/hello-world
```

The example prints:

```text
Hello G-Sharp
```

To run the compiler from Go source without building a reusable CLI binary:

```sh
go run ./cmd/gsharp run examples/hello-world
```

Run that command from this repository's root directory.

### Optional: install the CLI on your PATH

From the repository root:

```sh
go install ./cmd/gsharp
go env GOBIN GOPATH
```

The executable is installed in `GOBIN`, or in the `bin` directory under `GOPATH`
when `GOBIN` is empty. Add that directory to your `PATH`. You can then use
`gsharp` from any project directory. Later examples use this installed command;
otherwise substitute the path to your compiler executable.

## 3. Create your first project

From the compiler repository root on Windows:

```powershell
.\gsharp.exe new console Hello
Set-Location Hello
..\gsharp.exe run
```

From the compiler repository root on macOS/Linux:

```sh
./gsharp new console Hello
cd Hello
../gsharp run
```

With `gsharp` on your PATH, the equivalent workflow is:

```sh
gsharp new console Hello
cd Hello
gsharp run
```

The generated project contains:

```text
Hello/
├── gsharp.mod
└── src/
    └── main.gsh
```

Its starter program prints `Hello from G-Sharp!`.

To initialize an existing directory, run this inside that directory:

```sh
gsharp init example/hello
```

The module argument is optional. Initialization refuses to overwrite an existing
manifest. Use `new console NAME` when you want the CLI to create a new project
directory; the destination must not already exist.

## 4. Understand the project layout

A minimal `gsharp.mod` looks like this:

```text
module example/hello

gsharp 0.1

dependencies {
}
```

The dependency block must currently be empty. External package downloading and
dependency resolution are not implemented.

If a project has a `src` directory, the compiler discovers `.gsh` files
recursively beneath it. Otherwise it discovers `.gsh` files directly in the
project root. Files are processed in sorted path order. Keep unrelated example
programs in separate project directories so their entry points do not conflict.

A project can use one `static void Main()` entry point:

```csharp
namespace Hello;

class Program {
    static void Main() {
        Console.WriteLine("Hello from G-Sharp!");
    }
}
```

Alternatively, one source file may contain top-level statements:

```csharp
Console.WriteLine("Hello from G-Sharp!");
```

Do not combine top-level statements with a Main entry point. Top-level variables
are locals in the implicit entry function, not global variables.

## 5. Check, format, build, and run

Run these commands from a G-Sharp project directory:

```sh
gsharp check
gsharp fmt
gsharp fmt --check
gsharp build --emit-go
gsharp run
```

| Command | What it does |
| --- | --- |
| `check [PATH]` | Reports syntax and type errors without invoking the Go compiler |
| `fmt [PATH]` | Rewrites source with deterministic formatting and preserves comments |
| `fmt [PATH] --check` | Exits nonzero if formatting changes are needed; does not write files |
| `build [PATH]` | Builds a native executable and prints its path |
| `build [PATH] --emit-go` | Also prints the generated Go source path |
| `build [PATH] -o FILE` | Writes the executable to a chosen filename |
| `run [PATH]` | Builds and runs with terminal input, output, and error streams |
| `clean` | Removes the current project's compiler-owned `.gsharp` build directory |
| `version` | Shows the compiler version |
| `help` | Lists available commands |

PATH defaults to the current directory. It can also name a project directory or
a single `.gsh` file:

```sh
gsharp check examples/functions
gsharp run examples/functions/main.gsh
```

Relative output paths are resolved against the current working directory:

```powershell
gsharp build -o dist/hello.exe
.\dist\hello.exe
```

```sh
gsharp build -o dist/hello
./dist/hello
```

Default build products are:

```text
.gsharp/
├── compiler-owned
├── generated/
│   └── main.go
└── bin/
    └── app.exe       # app on macOS/Linux
```

Generated Go is retained even without `--emit-go`. Treat it as compiler output;
edit the `.gsh` files instead. `clean` leaves source files and custom output
locations such as `dist` untouched. Avoid running simultaneous builds of the same
project because they currently share the generated output directory.

`run -- ARGUMENTS...` forwards arguments to the native process, but this bootstrap
does not yet provide a language-level command-line argument API.

## 6. Try language features

### Variables and functions

```csharp
int Add(int a, int b) {
    return a + b;
}

const int Offset = 20;
var answer = Add(Offset, 22);
Console.WriteLine(answer);
```

`var` infers a static type. All local variables require an initializer. Constants
cannot be reassigned. There is no general implicit numeric widening.

### Classes, structs, and interfaces

```csharp
interface Processor {
    int Process(int value);
}

class Worker {
    int Process(int value) {
        return value * 2;
    }
}

Processor worker = new Worker();
Console.WriteLine(worker.Process(21));
```

Worker satisfies Processor because its public instance method has the required
signature. No implementation declaration or inheritance syntax is necessary.
Class assignments share references; struct assignments copy values. Struct
methods operate on a copy of their receiver.

### Collections

```csharp
var values = new List<int>();
values.Add(20);
values.Add(22);

var total = 0;
foreach (var value in values) {
    total += value;
}

var scores = new Map<string, int>();
scores["answer"] = total;
Console.WriteLine(scores["answer"]);
```

Array, List, Map, channels, and Result support built-in type arguments.
User-defined generic classes and functions are not implemented yet.

### Typed errors

```csharp
Result<int, Error> Divide(int numerator, int denominator) {
    if (denominator == 0) {
        return Result.Err(new Error("division by zero"));
    }
    return Result.Ok(numerator / denominator);
}

Result<int, Error> Calculate() {
    var value = Divide(84, 2)?;
    return Result.Ok(value);
}

var result = Calculate();
if (result.IsOk) {
    Console.WriteLine(result.Value);
} else {
    Console.WriteLine(result.Error.Message);
}
```

`?` unwraps success or returns the error from the enclosing function. That
function must return Result with the same error type. You cannot use `?` directly
in void Main or an anonymous goroutine block. Access `.Value` only on success
and `.Error` only on failure.

Result constructors need an expected type. For a local constructor, write:

```csharp
Result<int, Error> result = Result.Ok(42);
Console.WriteLine(result.Value);
```

### Concurrency

```csharp
var answers = chan<int>();
go {
    answers <- 42;
};

select {
    case answer = <-answers:
        Console.WriteLine(answer);
}
```

`chan<int>()` creates an unbuffered channel; `chan<int>(10)` creates a buffered
one. Channel operations synchronize work. Main returning terminates the program,
so explicitly wait for background work through a channel or another protocol.

## 7. Run the included examples

From this repository's root, using the installed CLI:

```sh
gsharp run examples/variables
gsharp run examples/classes
gsharp run examples/structs
gsharp run examples/interfaces
gsharp run examples/collections
gsharp run examples/result-errors
gsharp run examples/goroutines
gsharp run examples/select
gsharp run examples/namespaces
gsharp run examples/producer-consumer
```

The producer-consumer program combines interfaces, Result values, and channels,
and prints `42`. The `examples/generics` directory demonstrates only implemented
built-in generic collection types.

The HTTP example runs until interrupted:

```sh
gsharp run examples/http-server
```

Open [http://127.0.0.1:8080](http://127.0.0.1:8080) in a browser. Stop the server
with Ctrl+C. This minimal API serves a fixed response; custom request handlers
and production server configuration are future work.

## 8. Troubleshooting

### The terminal cannot find gsharp

Use `.\gsharp.exe` in PowerShell or `./gsharp` on macOS/Linux when the executable
is in the current directory. Otherwise use its full path or install it on PATH.

### The Go toolchain is missing or too old

Run `go version`. Install Go 1.26.6 or newer and reopen your terminal if PATH was
changed. Both `build` and `run` invoke Go; `check` only runs the G-Sharp frontend.

### No source files were found

Check your current directory and PATH argument. Use the `.gsh` extension. If a
`src` directory exists, put the project's source files inside it.

### Multiple entry points or conflicting top-level statements

Keep only one static Main, or put top-level statements in one file without Main.
Run an individual example directory instead of combining unrelated examples.

### GS1001, GS1101, or GS2001 diagnostics

These report lexical, syntax, or semantic/type errors respectively. Read the
filename, line, column, excerpt, and caret span. `check` can be used to fix these
before a native build. See [Diagnostics and Errors](errors.md).

### The build directory contains unrecognized files

The compiler refuses to take ownership of a nonempty, unrecognized `.gsharp`
directory. Inspect its contents and move files you need to keep elsewhere before
retrying. Do not use this directory for manually maintained project files.

### Go rejects generated source after check succeeds

This may indicate a compiler bug. Include the original `.gsh` source, the output
of `gsharp version` and `go version`, the complete error, and generated
`.gsharp/generated/main.go` when reporting it. Remove confidential data first.

## 9. Validate compiler changes

From this repository's root:

```sh
go fmt ./...
go vet ./...
go test ./...
```

The suite includes lexer/parser tests, semantic rejection tests, lowering tests,
formatter tests, generated-Go golden tests, CLI workflows, and native execution
tests. Every checked-in example is compiled; the HTTP server is not started by
the example test. `gsharp test` is not implemented: use Go's test command for the
compiler itself.

For exact language behavior and current limits, read the
[language specification](language-spec.md), [concurrency guide](concurrency.md),
and [roadmap](roadmap.md).
