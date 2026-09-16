// Package project owns manifests and deterministic source discovery.
package project

import (
	"fmt"
	"g-sharp/compiler"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Manifest struct{ Module, Version string }

func ParseManifest(text string) (Manifest, error) {
	var m Manifest
	inDeps := false
	closed := false
	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "//", 2)[0])
		if line == "" {
			continue
		}
		f := strings.Fields(line)
		bad := func() error {
			return fmt.Errorf("gsharp.mod:%d: invalid or unsupported manifest directive %q", i+1, line)
		}
		if inDeps {
			if line != "}" {
				return m, fmt.Errorf("gsharp.mod:%d: dependency resolution is not implemented", i+1)
			}
			inDeps = false
			closed = true
			continue
		}
		switch f[0] {
		case "module":
			if len(f) != 2 || m.Module != "" || strings.ContainsAny(f[1], "{}\\") {
				return m, bad()
			}
			m.Module = f[1]
		case "gsharp":
			if len(f) != 2 || m.Version != "" || f[1] != "0.1" {
				return m, bad()
			}
			m.Version = f[1]
		case "dependencies":
			if closed {
				return m, bad()
			}
			if line == "dependencies { }" || line == "dependencies {}" {
				closed = true
			} else if line == "dependencies {" {
				inDeps = true
			} else {
				return m, bad()
			}
		default:
			return m, bad()
		}
	}
	if inDeps || m.Module == "" || m.Version == "" {
		return m, fmt.Errorf("gsharp.mod requires module, gsharp 0.1, and balanced dependency braces")
	}
	return m, nil
}
func Sources(path string) (string, []compiler.Source, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", nil, err
	}
	if !info.IsDir() {
		if filepath.Ext(abs) != ".gsh" {
			return "", nil, fmt.Errorf("expected a .gsh source file")
		}
		b, err := os.ReadFile(abs)
		return filepath.Dir(abs), []compiler.Source{{Name: abs, Text: string(b)}}, err
	}
	manifest := filepath.Join(abs, "gsharp.mod")
	if b, e := os.ReadFile(manifest); e == nil {
		if _, e = ParseManifest(string(b)); e != nil {
			return "", nil, e
		}
	} else if !os.IsNotExist(e) {
		return "", nil, e
	}
	base := filepath.Join(abs, "src")
	var paths []string
	if st, e := os.Stat(base); e == nil && st.IsDir() {
		err = filepath.WalkDir(base, func(p string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if !d.IsDir() && filepath.Ext(p) == ".gsh" {
				paths = append(paths, p)
			}
			return nil
		})
	} else {
		paths, err = filepath.Glob(filepath.Join(abs, "*.gsh"))
	}
	if err != nil {
		return "", nil, err
	}
	sort.Strings(paths)
	var sources []compiler.Source
	for _, p := range paths {
		b, e := os.ReadFile(p)
		if e != nil {
			return "", nil, e
		}
		sources = append(sources, compiler.Source{Name: p, Text: string(b)})
	}
	if len(sources) == 0 {
		return "", nil, fmt.Errorf("no .gsh source files found in %s", abs)
	}
	return abs, sources, nil
}
func Init(path, module string) error {
	if module == "" {
		module = filepath.Base(path)
		if module == "." {
			cwd, e := os.Getwd()
			if e != nil {
				return e
			}
			module = filepath.Base(cwd)
		}
	}
	manifest := "module " + module + "\n\ngsharp 0.1\n\ndependencies {\n}\n"
	if _, err := ParseManifest(manifest); err != nil {
		return err
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(path, "gsharp.mod"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return fmt.Errorf("initialize project: %w", err)
	}
	_, err = f.WriteString(manifest)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	src := filepath.Join(path, "src")
	existing, err := filepath.Glob(filepath.Join(src, "*.gsh"))
	if err != nil {
		return err
	}
	rootFiles, err := filepath.Glob(filepath.Join(path, "*.gsh"))
	if err != nil {
		return err
	}
	if len(existing) > 0 || len(rootFiles) > 0 {
		return nil
	}
	if err = os.MkdirAll(src, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(src, "main.gsh"), []byte("class Program {\n    static void Main() {\n        Console.WriteLine(\"Hello from G-Sharp!\");\n    }\n}\n"), 0644)
}
