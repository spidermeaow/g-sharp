package cli

import (
	"errors"
	"fmt"
	"g-sharp/compiler"
	"g-sharp/compiler/codegen/golang"
	"g-sharp/compiler/diagnostics"
	"g-sharp/compiler/lowering"
	"g-sharp/formatter"
	"g-sharp/project"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const Version = "0.1.0-dev"

func Run(args []string, in io.Reader, out, errout io.Writer) int {
	err := run(args, in, out, errout)
	if err == nil {
		return 0
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode()
	}
	fmt.Fprintln(errout, err)
	return 1
}
func run(args []string, in io.Reader, out, errout io.Writer) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		fmt.Fprintln(out, "gsharp — G-Sharp bootstrap compiler\n\nCommands: new console NAME, init [MODULE], check [PATH], build [PATH] [-o FILE] [--emit-go], run [PATH] [-- ARGS...], fmt [PATH] [--check], clean, version\nPATH defaults to the current directory; it may be a project or .gsh file.")
		return nil
	}
	command := args[0]
	args = args[1:]
	switch command {
	case "version":
		fmt.Fprintln(out, "gsharp "+Version+" (Go bootstrap backend)")
		return nil
	case "init":
		if len(args) > 1 {
			return fmt.Errorf("usage: gsharp init [MODULE]")
		}
		module := ""
		if len(args) == 1 {
			module = args[0]
		}
		if err := project.Init(".", module); err != nil {
			return err
		}
		fmt.Fprintln(out, "Initialized G-Sharp project.")
		return nil
	case "new":
		if len(args) != 2 || args[0] != "console" {
			return fmt.Errorf("usage: gsharp new console NAME")
		}
		if _, err := os.Stat(args[1]); err == nil {
			return fmt.Errorf("destination already exists: %s", args[1])
		}
		if err := project.Init(args[1], filepath.Base(args[1])); err != nil {
			return err
		}
		fmt.Fprintln(out, "Created "+args[1])
		return nil
	case "clean":
		if len(args) != 0 {
			return fmt.Errorf("usage: gsharp clean")
		}
		root, err := filepath.Abs(".gsharp")
		if err != nil {
			return err
		}
		if _, err = os.Stat(root); os.IsNotExist(err) {
			return nil
		}
		if info, e := os.Lstat(root); e != nil || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to clean a linked or inaccessible build directory")
		}
		marker, err := os.ReadFile(filepath.Join(root, "compiler-owned"))
		if err != nil || string(marker) != "gsharp\n" {
			return fmt.Errorf("refusing to clean an unrecognized .gsharp directory")
		}
		return os.RemoveAll(root)
	}
	if command != "check" && command != "build" && command != "run" && command != "fmt" {
		return fmt.Errorf("unknown command %q; use gsharp help", command)
	}
	path := "."
	pathSet := false
	emit, checkFmt := false, false
	output := ""
	var programArgs []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--":
			if command != "run" {
				return fmt.Errorf("-- is only supported by run")
			}
			programArgs = args[i+1:]
			i = len(args)
		case "--emit-go":
			if command != "build" {
				return fmt.Errorf("--emit-go is only supported by build")
			}
			emit = true
		case "--check":
			if command != "fmt" {
				return fmt.Errorf("--check is only supported by fmt")
			}
			checkFmt = true
		case "-o":
			if command != "build" || i+1 == len(args) {
				return fmt.Errorf("build -o requires an output filename")
			}
			i++
			output = args[i]
		default:
			if strings.HasPrefix(a, "-") || pathSet {
				return fmt.Errorf("unexpected argument %q", a)
			}
			path = a
			pathSet = true
		}
	}
	root, sources, err := project.Sources(path)
	if err != nil {
		return err
	}
	if command == "fmt" {
		formatted := make([]string, len(sources))
		changed := false
		for i, s := range sources {
			f, ds := formatter.Format(s.Name, s.Text)
			if len(ds) > 0 {
				return render(ds, sources)
			}
			formatted[i] = f
			changed = changed || f != s.Text
		}
		if checkFmt && changed {
			return fmt.Errorf("source files need formatting")
		}
		if !checkFmt {
			for i, s := range sources {
				if formatted[i] != s.Text {
					info, e := os.Stat(s.Name)
					if e != nil {
						return e
					}
					if e = os.WriteFile(s.Name, []byte(formatted[i]), info.Mode().Perm()); e != nil {
						return e
					}
					fmt.Fprintln(out, s.Name)
				}
			}
		}
		return nil
	}
	p, ds := compiler.Check(sources)
	if len(ds) > 0 {
		return render(ds, sources)
	}
	if command == "check" {
		fmt.Fprintln(out, "Check succeeded.")
		return nil
	}
	source, err := golang.Generate(lowering.Lower(p))
	if err != nil {
		return err
	}
	cache := filepath.Join(root, ".gsharp")
	if info, e := os.Lstat(cache); e == nil && (info.Mode()&os.ModeSymlink != 0 || !info.IsDir()) {
		return fmt.Errorf("build directory must be a real directory")
	}
	if entries, e := os.ReadDir(cache); e == nil && len(entries) > 0 {
		marker, e := os.ReadFile(filepath.Join(cache, "compiler-owned"))
		if e != nil || string(marker) != "gsharp\n" {
			return fmt.Errorf("build directory contains unrecognized files: %s", cache)
		}
	}
	if err = os.MkdirAll(filepath.Join(cache, "generated"), 0755); err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(cache, "compiler-owned"), []byte("gsharp\n"), 0644); err != nil {
		return err
	}
	goFile := filepath.Join(cache, "generated", "main.go")
	if err = os.WriteFile(goFile, source, 0644); err != nil {
		return err
	}
	if output == "" {
		output = filepath.Join(cache, "bin", "app")
		if runtime.GOOS == "windows" {
			output += ".exe"
		}
	}
	output, err = filepath.Abs(output)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	cmd := exec.Command("go", "build", "-trimpath", "-o", output, goFile)
	cmd.Env = append(os.Environ(), "GOWORK=off", "CGO_ENABLED=0")
	if b, e := cmd.CombinedOutput(); e != nil {
		return fmt.Errorf("Go backend build failed (compiler/toolchain error): %w\n%s", e, b)
	}
	if command == "build" {
		fmt.Fprintln(out, output)
		if emit {
			fmt.Fprintln(out, goFile)
		}
		return nil
	}
	cmd = exec.Command(output, programArgs...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = errout
	return cmd.Run()
}
func render(ds diagnostics.List, sources []compiler.Source) error {
	var b strings.Builder
	for _, d := range ds {
		src := ""
		for _, s := range sources {
			if s.Name == d.Span.Start.File {
				src = s.Text
				break
			}
		}
		b.WriteString(diagnostics.Render(d, src))
		b.WriteByte('\n')
	}
	return errors.New(strings.TrimSpace(b.String()))
}
