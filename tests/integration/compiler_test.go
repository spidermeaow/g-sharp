package integration

import (
	"context"
	"g-sharp/compiler"
	"g-sharp/compiler/codegen/golang"
	"g-sharp/compiler/lowering"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func execute(t *testing.T, source string) string {
	t.Helper()
	p, ds := compiler.Check([]compiler.Source{{Name: "test.gsh", Text: source}})
	if len(ds) > 0 {
		t.Fatal(ds)
	}
	b, err := golang.Generate(lowering.Lower(p))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "main.go")
	if err = os.WriteFile(file, b, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "run", file)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s\n%s", err, out, b)
	}
	return strings.ReplaceAll(string(out), "\r\n", "\n")
}
func TestPrograms(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"hello", `class Program { static void Main() { Console.WriteLine("Hello G-Sharp"); } }`, "Hello G-Sharp\n"},
		{"numeric_boundaries", `int8 a=-128; uint64 b=18446744073709551615; var c=008; Console.WriteLine(a,b,c);`, "-128 18446744073709551615 8\n"},
		{"short_circuit", `Result<bool,Error> Fail(){return Result.Err(new Error("bad"));} Result<bool,Error> Good(){return Result.Ok(false && Fail()?);} Console.WriteLine(Good().Value);`, "false\n"},
		{"propagation_evaluation", `class C { int N; Result<int,Error> Next(){N++;return Result.Ok(N);} } Result<int,Error> Run(C c){var x=c.Next()?;return Result.Ok(x+c.N);} var c=new C(); Console.WriteLine(Run(c).Value,c.N);`, "2 1\n"},
		{"evaluation_order", `class C {int N;int Next(){N++;return N;}} int Pair(int a,int b){return a*10+b;} var c=new C(); Console.WriteLine(Pair(c.N,c.Next()));`, "1\n"},
		{"struct_method_copy", `struct V {int X;void Bump(){X++;}} var v=new V();v.Bump();Console.WriteLine(v.X);`, "0\n"},
		{"list_alias", `var a=new List<int>();var b=a;b.Add(42);Console.WriteLine(a[0]);`, "42\n"},
		{"go_argument_snapshot", `void Send(chan<int> c,int x){c <- x;} var c=chan<int>();var x=42;go Send(c,x);x=99;Console.WriteLine(<-c);`, "42\n"},
		{"functions", `int Add(int a, int b) { return a+b; } Console.WriteLine(Add(20,22));`, "42\n"},
		{"interfaces", `interface I { int F(int x); } class C { int F(int x) { return x*2; } } I c = new C(); Console.WriteLine(c.F(21));`, "42\n"},
		{"copy", `struct V { int X; } class R { int X { get; set; } } var a = new V(); a.X = 2; var b = a; b.X = 3; var c = new R(); var d = c; d.X = 4; Console.WriteLine(a.X, b.X, c.X);`, "2 3 4\n"},
		{"loops", `var sum = 0; for(var i=0;i<5;i++){ if(i==2){continue;} sum += i; } var xs = new List<int>(); xs.Add(sum); foreach(var x in xs){Console.WriteLine(x);} var m = new Map<string,int>(); m["x"] = 42; Console.WriteLine(m["x"]);`, "8\n42\n"},
		{"results", `Result<int,Error> F(int x){if(x<0){return Result.Err(new Error("bad"));}return Result.Ok(x*2);} Result<int,Error> G(int x){var v=F(x)?;return Result.Ok(v+1);} Console.WriteLine(G(20).Value); Console.WriteLine(G(-1).Error.Message);`, "41\nbad\n"},
		{"concurrency", `var jobs=chan<int>(1);var results=chan<int>();go {var v=<-jobs;results <- v*2;};jobs <- 21; select {case answer = <-results: Console.WriteLine(answer);}`, "42\n"},
		{"select_default", `var c=chan<int>();select{case x = <-c: Console.WriteLine(x);default:Console.WriteLine("empty");}`, "empty\n"},
		{"switch", `enum State { Ready, Stopped } var s=State.Ready;switch(s){case State.Ready:Console.WriteLine("ready");break;default:break;}`, "ready\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := execute(t, tc.source); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
