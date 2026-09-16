# Bootstrap grammar

The implementation uses recursive descent for declarations/statements and Pratt
precedence climbing for expressions. Comments are trivia, retained by the lexer
for formatting. `IDENT` excludes reserved keywords. Primitive type names are
resolved as types, not special parser tokens.

```ebnf
file        = { namespace | using | typeDecl | function | statement } ;
namespace   = "namespace" qualified ";" ;
using       = "using" qualified ";" ;
qualified   = IDENT { "." IDENT } ;
type        = IDENT [ "<" type { "," type } ">" ] ;
typeParams  = "<" IDENT { "," IDENT } ">" ;
typeDecl    = ("class" | "struct" | "interface") IDENT [typeParams]
              "{" {member} "}"
            | "enum" IDENT "{" IDENT {"," IDENT} [","] "}" ;
member      = {"public" | "private" | "static"} (function | field) ;
field       = type IDENT ( ["=" expression] ";"
            | "{" "get" ";" [ ["private"] "set" ";" ] "}" ) ;
function    = type IDENT [typeParams] "(" [parameters] ")" (block | ";") ;
parameters  = type IDENT { "," type IDENT } ;
block       = "{" {statement} "}" ;
statement   = block | simple ";"
            | "if" "(" expression ")" statement ["else" statement]
            | "for" "(" [simple] ";" [expression] ";" [simple] ")" statement
            | "foreach" "(" ("var" | type) IDENT "in" expression ")" statement
            | "return" [expression] ";" | ("break" | "continue") ";"
            | "go" (call | block) ";" | switch | select ;
simple      = ["const"] ("var" | type) IDENT "=" expression
            | expression [ ("=" | "+=" | "-=" | "*=" | "/=" | "<-") expression
                         | "++" | "--" ] ;
switch      = "switch" "(" expression ")" "{" {switchCase} "}" ;
switchCase  = ("case" expression | "default") ":" {statement} ;
select      = "select" "{" {selectCase} "}" ;
selectCase  = ("case" (expression "<-" expression | [IDENT "="] "<-" expression)
             | "default") ":" {statement} ;
expression  = prefix {postfix | binaryOp expression} ;
prefix      = literal | IDENT | "(" expression ")"
            | ("!" | "-" | "+" | "<-") expression
            | "new" type arguments | "chan" "<" type ">" arguments ;
postfix     = "." IDENT | arguments | "[" expression "]" | "?" ;
arguments   = "(" [expression {"," expression}] ")" ;
```

Precedence, low to high: `||`; `&&`; `== !=`; `< <= > >=`; `+ -`;
`* / %`; prefix operators; postfix calls/member/index/`?`. Binary operators
associate left. Assignment is a statement, not an expression. A function body is
required except for interface signatures. Generic declaration syntax is parsed
but explicitly rejected by the bootstrap checker. Directional channels,
inheritance, null, while, await, and lambda expressions are outside this grammar.

Manifest grammar:

```ebnf
manifest = "module" modulePath NEWLINE "gsharp" "0.1" NEWLINE
           ["dependencies" "{" "}"] ;
```

Directives occupy lines; whitespace and `//` comments are allowed. The empty
dependency block may span lines. Duplicate/unknown directives are errors.
