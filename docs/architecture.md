# Compiler architecture

Source → tokens → AST → name/type resolution → checked IR → lowering → Go source
→ Go toolchain → native executable.

The lexer owns UTF-8 positions; the recursive-descent/Pratt parser owns syntax.
The checker registers declarations before checking bodies, resolves lexical
scopes, checks control flow and structural interfaces, and builds typed IR.
The IR carries resolved names, static types, operations, and source spans; it has
no Go AST or raw parser tokens. The lowering package converts counted loops and
their continue targets to scoped loops, guards, labels, and jumps in core IR,
without mutating the checked program. Backend expression emission elaborates
Result propagation into explicit branches and temporaries, preserving evaluation
order and short-circuiting.
Only compiler/codegen/golang knows Go syntax. The project and CLI packages own
files, manifests, subprocesses, and build output. Diagnostics remain frontend
errors; generated Go failures indicate compiler bugs, not a substitute checker.

The runtime is emitted with each executable, using Go's GC, channels, scheduler,
and standard library. A future backend must implement those IR operations and
the language's runtime contracts, without changing parsing or name resolution.

Package responsibilities:

| Package | Responsibility |
| --- | --- |
| compiler/token, lexer, diagnostics | Tokens, UTF-8 source locations, error rendering |
| compiler/ast, parser | Positioned syntax nodes and recoverable parsing |
| compiler/types, checker | Symbols, resolution, access rules, structural typing, constants, flow |
| compiler/ir, lowering | Typed operations and target-independent control-flow lowering |
| compiler/codegen/golang | Go representation, runtime emission, expression evaluation order |
| project, formatter, internal/cli | Manifests, source discovery, formatting, build/run orchestration |

Semantic analysis and type checking share the checker package because their
symbol/type data is tightly coupled; declaration registration and body checking
are distinct passes. Expressions, constants, and statements live in separate
files. No empty semantic package or visitor framework is introduced solely to
match a directory diagram. The only dependencies are Go's standard library.
