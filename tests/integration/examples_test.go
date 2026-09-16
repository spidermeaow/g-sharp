package integration

import (
	"context"
	"g-sharp/compiler"
	"g-sharp/compiler/codegen/golang"
	"g-sharp/compiler/lowering"
	"g-sharp/formatter"
	"g-sharp/project"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExamples(t *testing.T) {
	entries, err := os.ReadDir("../../examples")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			_, sources, err := project.Sources(filepath.Join("../../examples", entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			p, ds := compiler.Check(sources)
			if len(ds) > 0 {
				t.Fatal(ds)
			}
			code, err := golang.Generate(lowering.Lower(p))
			if err != nil {
				t.Fatal(err)
			}
			again, err := golang.Generate(lowering.Lower(p))
			if err != nil || string(code) != string(again) {
				t.Fatal("nondeterministic generation", err)
			}
			for i, s := range sources {
				f, ds := formatter.Format(s.Name, s.Text)
				if len(ds) > 0 {
					t.Fatal(ds)
				}
				f2, ds := formatter.Format(s.Name, f)
				if len(ds) > 0 || f2 != f {
					t.Fatal("formatter not idempotent", ds)
				}
				sources[i].Text = f
			}
			if _, ds = compiler.Check(sources); len(ds) > 0 {
				t.Fatal("formatting changed validity", ds)
			}
			dir := t.TempDir()
			src := filepath.Join(dir, "main.go")
			if err = os.WriteFile(src, code, 0600); err != nil {
				t.Fatal(err)
			}
			exe := filepath.Join(dir, "example.exe")
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if out, err := exec.CommandContext(ctx, "go", "build", "-o", exe, src).CombinedOutput(); err != nil {
				t.Fatalf("%v\n%s", err, out)
			}
			if entry.Name() == "http-server" {
				return
			}
			out, err := exec.CommandContext(ctx, exe).CombinedOutput()
			if err != nil {
				t.Fatalf("%v\n%s", err, out)
			}
			if entry.Name() == "producer-consumer" && strings.TrimSpace(string(out)) != "42" {
				t.Fatalf("got %q", out)
			}
		})
	}
}
