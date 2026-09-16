package checker

import (
	"g-sharp/compiler/ast"
	"g-sharp/compiler/lexer"
	"g-sharp/compiler/parser"
	"testing"
)

func TestErrors(t *testing.T) {
	for _, src := range []string{`int x = "a";`, `var x = missing;`, `var x = 1; var x = 2;`, `const int x = 1; x = 2;`, `var c = chan<int>(); c <- "x";`, `var x = <-1;`, `int F() { return "bad"; } F();`, `int F(int x) { return x; } F();`, `var x = 1 + true;`, `interface I { int F(); } class C {} I x = new C();`, `Result<int, Error> F() { return Result.Ok(1); } var x = F()?;`, `class C { int x; } var c = new C(); c.Missing();`, `break;`, `var x = new List<void>();`, `int F() { if (true) { return 1; } } F();`} {
		t.Run(src, func(t *testing.T) {
			ts, ds := lexer.Scan("x", src)
			if len(ds) > 0 {
				t.Fatal(ds)
			}
			f, ds := parser.Parse(ts)
			if len(ds) > 0 {
				t.Fatal(ds)
			}
			_, ds = Check([]*ast.File{f})
			if len(ds) == 0 {
				t.Fatal("expected type error")
			}
		})
	}
}
func TestValid(t *testing.T) {
	src := `interface I { int F(int x); } class C { int F(int x) { return x * 2; } } I c = new C(); Console.WriteLine(c.F(21));`
	ts, _ := lexer.Scan("x", src)
	f, ds := parser.Parse(ts)
	if len(ds) > 0 {
		t.Fatal(ds)
	}
	_, ds = Check([]*ast.File{f})
	if len(ds) > 0 {
		t.Fatal(ds)
	}
}
