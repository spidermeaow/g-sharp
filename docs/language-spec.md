# G-Sharp bootstrap language specification

G-Sharp is statically typed, garbage collected, and compiled. Go is the initial
backend, not the language definition. This document describes the bootstrap
subset; the roadmap distinguishes deferred features.

## Lexical rules

UTF-8 source uses Unicode letters and digits in identifiers, with underscore.
Keywords are reserved. Integers are decimal; floats use a decimal point. Strings
and single-character literals support escaped newline, tab, carriage return,
backslash, and quotes. `//` and non-nesting `/* */` comments are supported.
Semicolons are required. Operators follow familiar C precedence; evaluation is
left to right and `&&` / `||` short circuit.

## Declarations and initialization

A file can have `namespace Name;`, `using Name;`, types, functions, and top-level
statements. A program has either top-level statements or one `static void Main()`.
Top-level functions use `int Add(int a, int b) { return a + b; }` syntax.
Top-level statements act as an implicit Main; their locals are not globals.
Namespace and using directives precede declarations; each file has at most one
namespace. Standard-library facade names are predeclared and may be shadowed by
local values.
No overloads, inheritance, constructors, reflection, or implicit numeric casts.
Local declarations require initializers. `var` infers one static type; `const`
requires a constant expression and cannot be reassigned. Primitive arithmetic
requires matching types. Unsuffixed literals have type int, float64, string, char,
or bool, with contextual numeric literal conversion when representable.

## Types and members

Primitives: bool, byte, int8/16/32/64, uint8/16/32/64, int, uint,
float32/64, char, string, void. `byte` aliases uint8. `char` is a Unicode scalar.
`class` has reference identity; `struct` copies its fields on assignment.
`new Name()` zero-initializes primitive fields. Methods are public unless private.
Struct methods operate on a copy; reference members still share their objects.
Direct field initializers run for `new Name()`; nested struct zero values and
array elements use their safe zero values without calling initializers.
Auto-properties use `T Name { get; set; }`; `private set` restricts writes to the
declaring type. Properties lower to storage plus access checks, not user code.
Interfaces contain method signatures and are satisfied structurally, with exact
parameter and return types. Static/private methods do not satisfy interfaces.
Enums contain named integer constants and are distinct from int.

Unrestricted `null` and nullable syntax are not implemented. Reference-typed fields
requiring unsafe implicit null initialization are rejected; use explicit field
initializers. Recursive object graphs need a future explicit optional type.

## Generics and collections

Type references carry type arguments independently of any backend. Built-in
`Array<T>`, `List<T>`, `Map<K,V>`, `chan<T>`, and `Result<T,E>` are parameterized.
Array construction takes a length, List/Map construction takes no arguments.
List.Add appends; collections have Count and support indexing. Array and List
share storage on assignment. Map keys must be comparable primitive/enum values.
Map indexing returns a zero value when absent, and is restricted to value types
with safe zero values. User generic declarations may be deferred, with an explicit
diagnostic, rather than accepting unchecked generic programs.
Map iteration yields keys in unspecified order. Array/List iteration yields
values and snapshots the iteration length. Map elements containing structs must
be read, changed as local copies, and assigned back; fields are not writable
directly through a map lookup. Strings expose Count as a UTF-8 byte count.

## Statements and control flow

Blocks introduce lexical scopes. Implemented forms include if/else, C-style for,
foreach, switch (no implicit fallthrough), return, break, and continue. Conditions
must be bool. Non-void functions must return on all reachable paths. Unreachable
statements are rejected. Assignment targets must be mutable storage.

## Expected errors

`Result<T,E>` is a tagged value. `Result.Ok(value)` and `Result.Err(error)` obtain
their missing type argument from the expected Result type. `Error` is a built-in
message value made with `new Error("message")`. Results expose IsOk, Value, and
Error; reading the wrong branch is an exceptional runtime failure. `expression?`
unwraps success and returns failure from the enclosing function. That function
must return Result with exactly the same error type. Operand evaluation occurs
exactly once. `?` is forbidden in void functions and goroutine blocks.

## Concurrency and memory

`go Call();` evaluates arguments before spawning. `go { statements };` captures
lexical variables and has a separate void return context. Channels are typed,
created by `chan<T>()` or `chan<T>(capacity)`; send is `ch <- value;`, receive is
`<-ch`. `select` chooses a ready operation, or default if supplied. Receive cases
may introduce a case-local binding: `case value = <-ch:`. A channel operation is
a synchronization boundary; shared unsynchronized mutation is a data race.
Program termination does not wait for goroutines. Synchronize explicitly with
channels. Backend memory ordering is Go's channel/goroutine memory model for v0.1.

## Modules and library

`gsharp.mod` contains `module <path>` and `gsharp 0.1`, optionally an empty
`dependencies { }` block. Dependencies are reserved, not fetched. Source discovery
uses src/**/*.gsh, or root *.gsh if src is absent, sorted by path. Namespace names
are logical scopes; using imports names without runtime initialization. File and
declaration order do not affect name lookup. Initialization runs before Main.
Console.WriteLine and Console.Write are available through System. Library APIs
are compiler-resolved signatures, not arbitrary access to Go packages.
