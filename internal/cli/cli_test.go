package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func invoke(args ...string) (int, string, string) {
	var out, err bytes.Buffer
	code := Run(args, strings.NewReader(""), &out, &err)
	return code, out.String(), err.String()
}
func TestLifecycle(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, args := range [][]string{{"init", "example/hello"}, {"check"}, {"fmt"}, {"fmt", "--check"}, {"build", "--emit-go"}, {"run"}, {"version"}} {
		code, out, err := invoke(args...)
		if code != 0 {
			t.Fatalf("%v: %s", args, err)
		}
		if args[0] == "run" && out != "Hello from G-Sharp!\n" {
			t.Fatal(out)
		}
	}
	if _, e := os.Stat(filepath.Join(".gsharp", "generated", "main.go")); e != nil {
		t.Fatal(e)
	}
	if code, _, err := invoke("clean"); code != 0 {
		t.Fatal(err)
	}
	if _, e := os.Stat(".gsharp"); !os.IsNotExist(e) {
		t.Fatal(e)
	}
}
func TestFrontendDiagnostic(t *testing.T) {
	t.Chdir(t.TempDir())
	if e := os.WriteFile("bad.gsh", []byte("int age = \"hello\";"), 0600); e != nil {
		t.Fatal(e)
	}
	code, _, err := invoke("build", "bad.gsh")
	if code == 0 || !strings.Contains(err, "GS2001") || !strings.Contains(err, "^^^^^^^") {
		t.Fatal(code, err)
	}
	if _, e := os.Stat(".gsharp"); !os.IsNotExist(e) {
		t.Fatal("frontend failure reached backend")
	}
}
func TestCleanRefusesUnknownDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	if e := os.Mkdir(".gsharp", 0755); e != nil {
		t.Fatal(e)
	}
	if code, _, _ := invoke("clean"); code == 0 {
		t.Fatal("should refuse unknown directory")
	}
}
