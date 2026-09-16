package golang

import (
	"g-sharp/compiler"
	"g-sharp/compiler/lowering"
	"os"
	"testing"
)

func TestGolden(t *testing.T) {
	input, err := os.ReadFile("testdata/hello.gsh")
	if err != nil {
		t.Fatal(err)
	}
	p, ds := compiler.Check([]compiler.Source{{Name: "hello.gsh", Text: string(input)}})
	if len(ds) > 0 {
		t.Fatal(ds)
	}
	got, err := Generate(lowering.Lower(p))
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/hello.go.golden")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("generated Go differs from testdata/hello.go.golden\n%s", got)
	}
}
