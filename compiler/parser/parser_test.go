package parser

import (
	"g-sharp/compiler/ast"
	"g-sharp/compiler/lexer"
	"testing"
)

func TestPrecedence(t *testing.T) {
	ts, _ := lexer.Scan("x", "var x = 1 + 2 * 3;")
	f, ds := Parse(ts)
	if len(ds) > 0 {
		t.Fatal(ds)
	}
	e := f.Statements[0].(*ast.Variable).Init.(*ast.Binary)
	if e.Op != "+" || e.Right.(*ast.Binary).Op != "*" {
		t.Fatal(e)
	}
}
func TestRecovery(t *testing.T) {
	ts, _ := lexer.Scan("x", "var a = ; var b = ; var c = 1;")
	f, ds := Parse(ts)
	if len(ds) != 2 || len(f.Statements) != 1 {
		t.Fatalf("%v %#v", ds, f)
	}
}
func TestDeclarations(t *testing.T) {
	ts, _ := lexer.Scan("x", `namespace Demo; interface Writer { void Write(string text); } class Program { static void Main() { Console.WriteLine("Hello"); } }`)
	f, ds := Parse(ts)
	if len(ds) > 0 || len(f.Types) != 2 {
		t.Fatalf("%v %#v", ds, f)
	}
}
