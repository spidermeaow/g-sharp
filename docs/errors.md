# Errors and diagnostics

Diagnostics contain a stable stage code, message, filename, one-based line/column,
source excerpt, and caret span. Offsets are UTF-8 byte offsets; columns count
Unicode code points. Tabs can affect terminal visual alignment.

| Code | Stage |
| --- | --- |
| GS1001 | Invalid lexical input or malformed literal |
| GS1101 | Syntax error |
| GS2001 | Name, type, access, or control-flow error |

Codes currently group errors by stage; finer codes and machine-readable output
are planned. Parser recovery synchronizes at semicolons or closing braces. A
failed frontend stage prevents backend invocation. CLI failures exit nonzero;
run forwards the child executable's exit status.

```text
main.gsh:1:11: error GS2001: cannot assign 'string' to 'int'

1 | int age = "hello";
              ^^^^^^^
```

Expected runtime errors use `Result<T,E>`. There are no exceptions for normal
control flow. `Result.Ok(value)` / `Result.Err(error)` require an expected Result
type from a return, argument, assignment, or explicit variable declaration.
Use `Result<int, Error> r = Result.Ok(42);`, not `var r = Result.Ok(42);`.
`?` evaluates once, returns the error if present, otherwise yields the success
value. The enclosing function must return a Result with the same E; its success
type may differ. `.Value` or `.Error` on the wrong branch panics. Bounds errors,
runtime division by zero, and invalid channel capacities are exceptional errors.

If Go rejects generated source after check succeeded, report it as a compiler bug
with the `.gsh` input and `.gsharp/generated/main.go`. Missing/incompatible Go
installations are toolchain errors, separate from source diagnostics.
