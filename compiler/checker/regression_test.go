package checker

import (
	"g-sharp/compiler/ast"
	"g-sharp/compiler/lexer"
	"g-sharp/compiler/parser"
	"strings"
	"testing"
)

func TestFrontendRegressions(t *testing.T) {
	cases := []struct{ source, message string }{
		{`var c=chan<void>();`, "void"},
		{`var c=chan<int>(-1);`, "nonnegative"},
		{`var a=new Array<int>(-2);`, "nonnegative"},
		{`class C {} var a=new Array<C>(1);`, "safe zero"},
		{`const int8 a=127; var b=a+1;`, "out of range"},
		{`const int x=1/(2-2);`, "division by zero"},
		{`var x=1/false;`, "same type"},
		{`class C { int X=F(); int F(){return 1;} } var c=new C();`, "constant"},
		{`class C { C Other; } var c=new C();`, "safe initializer"},
		{`class C { int Value {get;private set;} } var c=new C(); c.Value=1;`, "mutable"},
		{`interface I {int F();} class C {private int F(){return 1;}} I c=new C();`, "cannot assign"},
		{`interface I {int F();} class C {static int F(){return 1;}} I c=new C();`, "cannot assign"},
		{`Result<int,Error> F(){return Result.Ok(1);} Result<int,string> G(){return Result.Ok(F()?);} G();`, "same error type"},
		{`Result<int,Error> F(){return Result.Ok(1);} go {var x=F()?;};`, "requires a function"},
		{`for(var i=0;i<1;i++){go {break;};}`, "context"},
		{`switch(2){case 1+1:break;case 2:break;}`, "duplicate switch"},
		{`class Box<T>{T Value;} Console.WriteLine("x");`, "generics"},
		{`void F(){return;Console.WriteLine(1);} F();`, "unreachable"},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			ts, ds := lexer.Scan("test.gsh", tc.source)
			if len(ds) > 0 {
				t.Fatal(ds)
			}
			f, ds := parser.Parse(ts)
			if len(ds) > 0 {
				t.Fatal(ds)
			}
			_, ds = Check([]*ast.File{f})
			if len(ds) == 0 || !strings.Contains(ds.Error(), tc.message) {
				t.Fatalf("expected %q, got %v", tc.message, ds)
			}
		})
	}
}
