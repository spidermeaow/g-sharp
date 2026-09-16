package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManifest(t *testing.T) {
	m, e := ParseManifest("module example/hello\ngsharp 0.1\ndependencies {\n}\n")
	if e != nil || m.Module != "example/hello" {
		t.Fatal(m, e)
	}
	for _, s := range []string{"", "module x\ngsharp 2", "module x\ngsharp 0.1\ndependencies {\nfoo v1\n}", "module x\nmodule y\ngsharp 0.1"} {
		if _, e := ParseManifest(s); e == nil {
			t.Fatal(s)
		}
	}
}
func TestInit(t *testing.T) {
	d := t.TempDir()
	if e := Init(d, "example/test"); e != nil {
		t.Fatal(e)
	}
	if e := Init(d, "example/test"); e == nil {
		t.Fatal("must not overwrite")
	}
	_, ss, e := Sources(d)
	if e != nil || len(ss) != 1 {
		t.Fatal(ss, e)
	}
	if _, e = os.Stat(filepath.Join(d, "src", "main.gsh")); e != nil {
		t.Fatal(e)
	}
}
