// Package lowering converts checked high-level control flow into core IR.
// It is independent of any output language and never mutates its input program.
package lowering

import (
	"fmt"
	"g-sharp/compiler/ir"
	"g-sharp/compiler/types"
)

type lowerer struct{ sequence int }

func Lower(p *ir.Program) *ir.Program {
	l := &lowerer{}
	out := &ir.Program{}
	functions := map[*ir.Function]*ir.Function{}
	lowerFunc := func(f *ir.Function) *ir.Function {
		r := *f
		r.Body = l.statements(f.Body, "")
		functions[f] = &r
		return &r
	}
	for _, d := range p.Types {
		r := *d
		r.Methods = nil
		for _, f := range d.Methods {
			r.Methods = append(r.Methods, lowerFunc(f))
		}
		out.Types = append(out.Types, &r)
	}
	for _, f := range p.Functions {
		out.Functions = append(out.Functions, lowerFunc(f))
	}
	out.Entry = functions[p.Entry]
	return out
}
func (l *lowerer) statements(ss []*ir.Stmt, target string) []*ir.Stmt {
	var out []*ir.Stmt
	for _, s := range ss {
		out = append(out, l.stmt(s, target))
	}
	return out
}
func (l *lowerer) stmt(s *ir.Stmt, target string) *ir.Stmt {
	if s == nil {
		return nil
	}
	r := *s
	switch s.Op {
	case "for":
		l.sequence++
		label := fmt.Sprintf("gs_loop_post_%d", l.sequence)
		var body []*ir.Stmt
		if s.Expr != nil {
			body = append(body, &ir.Stmt{Op: "if", Span: s.Span, Expr: &ir.Expr{Op: "unary", Name: "!", Type: types.Bool, Args: []*ir.Expr{s.Expr}}, Body: []*ir.Stmt{{Op: "break"}}})
		}
		body = append(body, &ir.Stmt{Op: "block", Body: l.statements(s.Body, label)}, &ir.Stmt{Op: "jump", Name: label}, &ir.Stmt{Op: "label", Name: label})
		if s.Post != nil {
			body = append(body, l.stmt(s.Post, target))
		}
		r.Op = "block"
		r.Expr = nil
		r.Init = nil
		r.Post = nil
		r.Body = nil
		if s.Init != nil {
			r.Body = append(r.Body, l.stmt(s.Init, target))
		}
		r.Body = append(r.Body, &ir.Stmt{Op: "loop", Span: s.Span, Body: body})
		return &r
	case "continue":
		if target != "" {
			r.Op = "jump"
			r.Name = target
		}
	case "foreach", "go":
		r.Body = l.statements(s.Body, "")
	default:
		r.Body = l.statements(s.Body, target)
	}
	r.Else = l.statements(s.Else, target)
	r.Cases = nil
	for _, cs := range s.Cases {
		copy := cs
		copy.Body = l.statements(cs.Body, target)
		r.Cases = append(r.Cases, copy)
	}
	return &r
}
