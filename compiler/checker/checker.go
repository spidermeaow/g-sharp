package checker

import (
	"fmt"
	"g-sharp/compiler/ast"
	"g-sharp/compiler/diagnostics"
	"g-sharp/compiler/ir"
	"g-sharp/compiler/token"
	"g-sharp/compiler/types"
)

type binding struct {
	name     string
	typ      *types.Type
	constant bool
	value    *ir.Expr
}
type scope struct {
	parent *scope
	names  map[string]binding
}

func (s *scope) find(n string) (binding, bool) {
	for ; s != nil; s = s.parent {
		if b, ok := s.names[n]; ok {
			return b, true
		}
	}
	return binding{}, false
}

type Checker struct {
	program                 *ir.Program
	errors                  diagnostics.List
	decls                   map[string]*ir.TypeDecl
	funcs                   map[string]*ir.Function
	astTypes                map[*ast.TypeDecl]*ir.TypeDecl
	astFuncs                map[*ast.Function]*ir.Function
	file                    *ast.File
	scope                   *scope
	fn                      *ir.Function
	owner                   *ir.TypeDecl
	sequence, loops, breaks int
}

func Check(files []*ast.File) (*ir.Program, diagnostics.List) {
	c := &Checker{program: &ir.Program{}, decls: map[string]*ir.TypeDecl{}, funcs: map[string]*ir.Function{}, astTypes: map[*ast.TypeDecl]*ir.TypeDecl{}, astFuncs: map[*ast.Function]*ir.Function{}}
	c.check(files)
	return c.program, c.errors
}
func (c *Checker) error(s token.Span, msg string) {
	c.errors = append(c.errors, diagnostics.Diagnostic{Code: "GS2001", Message: msg, Span: s})
}
func (c *Checker) fresh(n string) string { c.sequence++; return fmt.Sprintf("gs_%d_%s", c.sequence, n) }
func qualify(ns, n string) string {
	if ns == "" {
		return n
	}
	return ns + "." + n
}
func (c *Checker) lookup(n string) string {
	local := qualify(c.file.Namespace, n)
	if c.decls[local] != nil || c.funcs[local] != nil {
		return local
	}
	if c.decls[n] != nil || c.funcs[n] != nil {
		return n
	}
	found := ""
	for _, u := range c.file.Usings {
		k := qualify(u, n)
		if c.decls[k] != nil || c.funcs[k] != nil {
			if found != "" && found != k {
				return "<ambiguous>"
			}
			found = k
		}
	}
	if found != "" {
		return found
	}
	return local
}
func (c *Checker) typ(r ast.TypeRef) *types.Type {
	n := r.Name
	if n == "byte" {
		n = "uint8"
	}
	args := []*types.Type{}
	for _, a := range r.Args {
		args = append(args, c.typ(a))
	}
	arity := 0
	switch n {
	case "Array", "List", "chan":
		arity = 1
	case "Map", "Result":
		arity = 2
	default:
		if !types.Primitives[n] {
			n = c.lookup(n)
			if c.decls[n] == nil {
				c.error(r.Span, "unknown type '"+r.Name+"'")
				return types.Invalid
			}
		}
	}
	if len(args) != arity {
		c.error(r.Span, fmt.Sprintf("type %s requires %d type arguments", r.Name, arity))
		return types.Invalid
	}
	for _, a := range args {
		if types.Equal(a, types.Void) {
			c.error(r.Span, "void cannot be a type argument")
		}
	}
	t := &types.Type{Name: n, Args: args}
	if n == "Map" && len(args) == 2 {
		if !c.comparable(args[0]) {
			c.error(r.Span, "map keys must be comparable primitive or enum values")
		}
		if !c.safeZero(args[1], map[string]bool{}) {
			c.error(r.Span, "map values must have safe zero values")
		}
	}
	return t
}
func (c *Checker) comparable(t *types.Type) bool {
	if t == nil {
		return false
	}
	if types.Primitives[t.Name] && t.Name != "void" {
		return true
	}
	d := c.decls[t.Name]
	return d != nil && d.Kind == "enum"
}
func (c *Checker) safeZero(t *types.Type, seen map[string]bool) bool {
	if types.Primitives[t.Name] && t.Name != "void" {
		return true
	}
	d := c.decls[t.Name]
	if d == nil {
		return false
	}
	if d.Kind == "enum" {
		return true
	}
	if d.Kind != "struct" || seen[t.Name] {
		return false
	}
	seen[t.Name] = true
	defer delete(seen, t.Name)
	for _, f := range d.Fields {
		if !c.safeZero(f.Type, seen) {
			return false
		}
	}
	return true
}
func (c *Checker) check(files []*ast.File) {
	for _, f := range files {
		c.file = f
		for _, d := range f.Types {
			k := qualify(f.Namespace, d.Name)
			if types.Primitives[d.Name] || d.Name == "Result" || d.Name == "List" || d.Name == "Map" || d.Name == "Array" || d.Name == "chan" {
				c.error(d.Span, "reserved type name "+d.Name)
			}
			if c.decls[k] != nil {
				c.error(d.Span, "duplicate type "+k)
				continue
			}
			if len(d.Parameters) > 0 {
				c.error(d.Span, "user-defined generics are reserved for the next milestone")
			}
			r := &ir.TypeDecl{Name: c.fresh(d.Name), SourceName: k, Kind: d.Kind, Members: d.Members}
			c.decls[k] = r
			c.astTypes[d] = r
			c.program.Types = append(c.program.Types, r)
		}
	}
	for _, f := range files {
		c.file = f
		for _, d := range f.Types {
			r := c.astTypes[d]
			if r == nil {
				continue
			}
			names := map[string]bool{}
			for _, n := range d.Members {
				if names[n] {
					c.error(d.Span, "duplicate enum member "+n)
				}
				names[n] = true
			}
			for _, field := range d.Fields {
				if names[field.Name] {
					c.error(field.Span, "duplicate member "+field.Name)
				}
				names[field.Name] = true
				ty := c.typ(field.Type)
				if types.Equal(ty, types.Void) {
					c.error(field.Span, "field cannot have type void")
				}
				r.Fields = append(r.Fields, &ir.Field{Name: c.fresh(field.Name), SourceName: field.Name, Type: ty, Private: field.Private, PrivateSet: field.PrivateSet, ReadOnly: field.ReadOnly})
			}
			for _, fn := range d.Methods {
				if names[fn.Name] {
					c.error(fn.Span, "duplicate member "+fn.Name)
				}
				names[fn.Name] = true
				rf := c.signature(fn, r)
				r.Methods = append(r.Methods, rf)
			}
		}
		for _, fn := range f.Functions {
			k := qualify(f.Namespace, fn.Name)
			if c.funcs[k] != nil || c.decls[k] != nil {
				c.error(fn.Span, "duplicate symbol "+k)
			}
			r := c.signature(fn, nil)
			c.funcs[k] = r
			c.program.Functions = append(c.program.Functions, r)
		}
	}
	// Check fields after all signatures exist, then check executable bodies.
	for _, f := range files {
		c.file = f
		for _, d := range f.Types {
			r := c.astTypes[d]
			if r == nil {
				continue
			}
			c.owner = r
			for i, field := range d.Fields {
				c.scope = &scope{names: map[string]binding{}}
				if field.Init != nil {
					e := c.expr(field.Init, r.Fields[i].Type)
					c.require(e, r.Fields[i].Type)
					if !e.Constant {
						c.error(field.Span, "field initializers must be constant expressions")
					}
					r.Fields[i].Init = e
				} else if !c.safeZero(r.Fields[i].Type, map[string]bool{}) {
					c.error(field.Span, "field requires a safe initializer; nullable references are not supported")
				}
			}
			for _, fn := range d.Methods {
				if fn.Body != nil {
					c.body(fn, c.astFuncs[fn], r)
				}
			}
		}
		for _, fn := range f.Functions {
			c.body(fn, c.astFuncs[fn], nil)
		}
	}
	var top []ast.Stmt
	var topFile *ast.File
	for _, f := range files {
		if len(f.Statements) > 0 {
			if topFile != nil {
				c.error(f.Statements[0].Location(), "top-level statements may appear in only one file")
			}
			topFile = f
			top = append(top, f.Statements...)
		}
	}
	if len(top) > 0 {
		if c.program.Entry != nil {
			c.error(top[0].Location(), "cannot combine Main and top-level statements")
		}
		c.file = topFile
		c.owner = nil
		c.fn = &ir.Function{Name: c.fresh("entry"), Return: types.Void, Static: true}
		c.scope = &scope{names: map[string]binding{}}
		c.fn.Body = c.statements(top)
		c.program.Entry = c.fn
		c.program.Functions = append(c.program.Functions, c.fn)
	}
	if c.program.Entry == nil {
		c.error(token.Span{}, "program needs static void Main() or top-level statements")
	}
}
func (c *Checker) signature(f *ast.Function, owner *ir.TypeDecl) *ir.Function {
	if len(f.TypeParameters) > 0 {
		c.error(f.Span, "user-defined generics are reserved for the next milestone")
	}
	r := &ir.Function{Name: c.fresh(f.Name), SourceName: f.Name, Return: c.typ(f.Return), Static: f.Static, Private: f.Private}
	if owner != nil {
		r.Owner = owner.SourceName
		if owner.Kind == "interface" && (f.Static || f.Private) {
			c.error(f.Span, "interface methods must be public instance methods")
		}
	}
	for _, p := range f.Parameters {
		ty := c.typ(p.Type)
		if types.Equal(ty, types.Void) {
			c.error(p.Span, "parameter cannot have type void")
		}
		r.Parameters = append(r.Parameters, ir.Parameter{Name: c.fresh(p.Name), Type: ty})
	}
	c.astFuncs[f] = r
	return r
}
func (c *Checker) body(f *ast.Function, r *ir.Function, owner *ir.TypeDecl) {
	c.owner = owner
	c.fn = r
	c.scope = &scope{names: map[string]binding{}}
	c.loops = 0
	c.breaks = 0
	for i, p := range f.Parameters {
		c.define(p.Name, binding{name: r.Parameters[i].Name, typ: r.Parameters[i].Type}, p.Span)
	}
	if f.Name == "Main" && f.Static {
		if len(r.Parameters) != 0 || !types.Equal(r.Return, types.Void) {
			c.error(f.Span, "entry point must be static void Main()")
		}
		if c.program.Entry != nil {
			c.error(f.Span, "multiple entry points")
		}
		c.program.Entry = r
	}
	r.Body = c.statements(f.Body.Statements)
	if !types.Equal(r.Return, types.Void) && !returns(r.Body) {
		c.error(f.Span, "not all paths return a value")
	}
}
func (c *Checker) define(n string, b binding, s token.Span) {
	if _, ok := c.scope.names[n]; ok {
		c.error(s, "duplicate symbol "+n)
	}
	c.scope.names[n] = b
}
func (c *Checker) push() { c.scope = &scope{parent: c.scope, names: map[string]binding{}} }
func (c *Checker) pop()  { c.scope = c.scope.parent }
func (c *Checker) assignable(a, b *types.Type) bool {
	if types.Equal(a, b) || a == types.Invalid || b == types.Invalid {
		return true
	}
	d := c.decls[b.Name]
	src := c.decls[a.Name]
	if d == nil || d.Kind != "interface" || src == nil {
		return false
	}
	for _, want := range d.Methods {
		found := false
		for _, m := range src.Methods {
			if m.Static || m.Private || m.SourceName != want.SourceName || !types.Equal(m.Return, want.Return) || len(m.Parameters) != len(want.Parameters) {
				continue
			}
			good := true
			for i, p := range m.Parameters {
				good = good && types.Equal(p.Type, want.Parameters[i].Type)
			}
			if good {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}
func (c *Checker) require(e *ir.Expr, t *types.Type) {
	if e != nil && !c.assignable(e.Type, t) {
		c.error(e.Span, "cannot assign '"+e.Type.String()+"' to '"+t.String()+"'")
	}
}
