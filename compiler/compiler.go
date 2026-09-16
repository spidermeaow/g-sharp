package compiler

import (
	"g-sharp/compiler/ast"
	"g-sharp/compiler/checker"
	"g-sharp/compiler/diagnostics"
	"g-sharp/compiler/ir"
	"g-sharp/compiler/lexer"
	"g-sharp/compiler/parser"
)

type Source struct{ Name, Text string }

func Check(sources []Source) (*ir.Program, diagnostics.List) {
	var files []*ast.File
	var errors diagnostics.List
	for _, s := range sources {
		ts, ds := lexer.Scan(s.Name, s.Text)
		errors = append(errors, ds...)
		if len(ds) > 0 {
			continue
		}
		f, ds := parser.Parse(ts)
		errors = append(errors, ds...)
		files = append(files, f)
	}
	if len(errors) > 0 {
		return nil, errors
	}
	return checker.Check(files)
}
