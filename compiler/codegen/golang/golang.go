// Package golang lowers typed IR operations to Go. No frontend syntax enters here.
package golang

import (
	"fmt"
	"g-sharp/compiler/ir"
	"g-sharp/compiler/types"
	"go/format"
	"strings"
)

type generator struct {
	out        strings.Builder
	level, seq int
	decls      map[string]*ir.TypeDecl
	ret        *types.Type
}

func Generate(p *ir.Program) ([]byte, error) {
	g := &generator{decls: map[string]*ir.TypeDecl{}}
	for _, d := range p.Types {
		g.decls[d.SourceName] = d
	}
	g.out.WriteString(runtimeSource)
	for _, d := range p.Types {
		g.decl(d)
	}
	for _, d := range p.Types {
		if d.Kind != "interface" {
			for _, f := range d.Methods {
				g.function(f, d)
			}
		}
	}
	for _, f := range p.Functions {
		g.function(f, nil)
	}
	g.line("func main() { %s() }", p.Entry.Name)
	b, err := format.Source([]byte(g.out.String()))
	if err != nil {
		return nil, fmt.Errorf("internal backend error: %w\n%s", err, g.out.String())
	}
	return b, nil
}
func (g *generator) line(f string, a ...any) {
	g.out.WriteString(strings.Repeat("\t", g.level))
	fmt.Fprintf(&g.out, f, a...)
	g.out.WriteByte('\n')
}
func (g *generator) temp() string { g.seq++; return fmt.Sprintf("gs_tmp_%d", g.seq) }
func (g *generator) typ(t *types.Type) string {
	switch t.Name {
	case "char":
		return "rune"
	case "Error":
		return "gsError"
	case "void":
		return ""
	case "Array":
		return "[]" + g.typ(t.Args[0])
	case "List":
		return "*gsList[" + g.typ(t.Args[0]) + "]"
	case "Map":
		return "map[" + g.typ(t.Args[0]) + "]" + g.typ(t.Args[1])
	case "chan":
		return "chan " + g.typ(t.Args[0])
	case "Result":
		return "gsResult[" + g.typ(t.Args[0]) + ", " + g.typ(t.Args[1]) + "]"
	}
	if d := g.decls[t.Name]; d != nil {
		if d.Kind == "class" {
			return "*" + d.Name
		}
		return d.Name
	}
	return t.Name
}
func (g *generator) params(f *ir.Function) string {
	var a []string
	for _, p := range f.Parameters {
		a = append(a, p.Name+" "+g.typ(p.Type))
	}
	return strings.Join(a, ", ")
}
func (g *generator) decl(d *ir.TypeDecl) {
	switch d.Kind {
	case "enum":
		g.line("type %s int", d.Name)
	case "interface":
		g.line("type %s interface {", d.Name)
		g.level++
		for _, m := range d.Methods {
			g.line("M_%s(%s) %s", m.SourceName, g.params(m), g.typ(m.Return))
		}
		g.level--
		g.line("}")
	default:
		g.line("type %s struct {", d.Name)
		g.level++
		for _, f := range d.Fields {
			g.line("%s %s", f.Name, g.typ(f.Type))
		}
		g.level--
		g.line("}")
	}
}
func (g *generator) function(f *ir.Function, d *ir.TypeDecl) {
	g.ret = f.Return
	name := f.Name
	recv := ""
	if d != nil && !f.Static {
		recv = "(self " + g.typ(&types.Type{Name: d.SourceName}) + ") "
		name = "M_" + f.SourceName
	}
	g.line("func %s%s(%s) %s {", recv, name, g.params(f), g.typ(f.Return))
	g.level++
	g.statements(f.Body)
	g.level--
	g.line("}")
}
func (g *generator) freeze(e *ir.Expr) string {
	v := g.expr(e)
	if e.Constant {
		return v
	}
	if types.Equal(e.Type, types.Void) {
		return v
	}
	n := g.temp()
	g.line("var %s %s = %s", n, g.typ(e.Type), v)
	return n
}
func (g *generator) store(e *ir.Expr, value string) string {
	if types.Equal(e.Type, types.Void) {
		g.line("%s", value)
		return ""
	}
	n := g.temp()
	g.line("var %s %s = %s", n, g.typ(e.Type), value)
	return n
}
func (g *generator) expr(e *ir.Expr) string {
	if e == nil {
		return ""
	}
	switch e.Op {
	case "constant":
		return g.expr(e.Args[0])
	case "literal":
		return g.typ(e.Type) + "(" + e.Value + ")"
	case "local":
		return e.Name
	case "field":
		return "(" + g.expr(e.Args[0]) + ")." + e.Name
	case "binary":
		if e.Constant {
			return "(" + g.expr(e.Args[0]) + " " + e.Name + " " + g.expr(e.Args[1]) + ")"
		}
		l := g.freeze(e.Args[0])
		if e.Name == "&&" || e.Name == "||" {
			n := g.temp()
			g.line("%s := %s", n, l)
			cond := n
			if e.Name == "||" {
				cond = "!" + n
			}
			g.line("if %s {", cond)
			g.level++
			r := g.expr(e.Args[1])
			g.line("%s = %s", n, r)
			g.level--
			g.line("}")
			return n
		}
		r := g.freeze(e.Args[1])
		return g.store(e, "("+l+" "+e.Name+" "+r+")")
	case "unary":
		v := g.expr(e.Args[0])
		return "(" + e.Name + v + ")"
	case "receive":
		v := g.expr(e.Args[0])
		return g.store(e, "<-"+v)
	case "propagate":
		v := g.freeze(e.Args[0])
		g.line("if !%s.ok {", v)
		g.level++
		g.line("return %s{err: %s.err}", g.typ(g.ret), v)
		g.level--
		g.line("}")
		return v + ".value"
	case "index":
		obj := g.freeze(e.Args[0])
		idx := g.freeze(e.Args[1])
		if e.Args[0].Type.Name == "List" {
			obj += ".items"
		}
		return g.store(e, obj+"["+idx+"]")
	case "count":
		obj := g.expr(e.Args[0])
		if e.Args[0].Type.Name == "List" {
			obj += ".items"
		}
		return g.store(e, "len("+obj+")")
	case "error_message":
		return g.expr(e.Args[0]) + ".Message"
	case "result_ok":
		return g.expr(e.Args[0]) + ".ok"
	case "result_value":
		return g.store(e, g.expr(e.Args[0])+".Value()")
	case "result_error":
		return g.store(e, g.expr(e.Args[0])+".Error()")
	case "new":
		var args []string
		for _, a := range e.Args {
			args = append(args, g.freeze(a))
		}
		t := g.typ(e.Type)
		switch e.Type.Name {
		case "Error":
			return g.store(e, "gsError{Message: "+args[0]+"}")
		case "List":
			return g.store(e, "&"+strings.TrimPrefix(t, "*")+"{}")
		case "Map":
			return g.store(e, "make("+t+")")
		case "Array":
			return g.store(e, "make("+t+", "+args[0]+")")
		}
		d := g.decls[e.Type.Name]
		var fs []string
		for _, f := range d.Fields {
			if f.Init != nil {
				fs = append(fs, f.Name+": "+g.expr(f.Init))
			}
		}
		v := d.Name + "{" + strings.Join(fs, ", ") + "}"
		if d.Kind == "class" {
			v = "&" + v
		}
		return g.store(e, v)
	case "channel":
		size := "0"
		if len(e.Args) > 0 {
			size = g.expr(e.Args[0])
		}
		return g.store(e, "make("+g.typ(e.Type)+", "+size+")")
	case "result":
		v := g.expr(e.Args[0])
		field := "ok: true, value"
		if e.Name == "Err" {
			field = "err"
		}
		return g.store(e, g.typ(e.Type)+"{"+field+": "+v+"}")
	default:
		return g.store(e, g.call(e))
	}
}
func (g *generator) call(e *ir.Expr) string {
	var args []string
	for _, a := range e.Args {
		args = append(args, g.freeze(a))
	}
	name := e.Name
	switch e.Op {
	case "method":
		name = args[0] + "." + e.Name
		args = args[1:]
	case "print":
		name = "fmt.Println"
		if e.Name == "Write" {
			name = "fmt.Print"
		}
	case "list_add":
		name = args[0] + ".Add"
		args = args[1:]
	case "sleep":
		name = "gsSleep"
	case "read_file":
		name = "gsReadFile"
	case "http_listen":
		name = "gsHTTP"
	}
	return name + "(" + strings.Join(args, ", ") + ")"
}
func (g *generator) lvalue(e *ir.Expr) string {
	switch e.Op {
	case "local":
		return e.Name
	case "field":
		obj := e.Args[0]
		if d := g.decls[obj.Type.Name]; d != nil && d.Kind == "struct" {
			return "(" + g.lvalue(obj) + ")." + e.Name
		}
		return "(" + g.freeze(obj) + ")." + e.Name
	case "index":
		obj := g.freeze(e.Args[0])
		idx := g.freeze(e.Args[1])
		if e.Args[0].Type.Name == "List" {
			obj += ".items"
		}
		return obj + "[" + idx + "]"
	}
	panic("invalid checked lvalue")
}
func (g *generator) statements(ss []*ir.Stmt) {
	for _, s := range ss {
		g.stmt(s)
	}
}
func (g *generator) stmt(s *ir.Stmt) {
	if s == nil {
		return
	}
	switch s.Op {
	case "block":
		g.line("{")
		g.level++
		g.statements(s.Body)
		g.level--
		g.line("}")
	case "loop":
		g.line("for {")
		g.level++
		g.statements(s.Body)
		g.level--
		g.line("}")
	case "jump":
		g.line("goto %s", s.Name)
	case "label":
		g.line("%s:", s.Name)
		g.line(";")
	case "var", "const":
		v := g.expr(s.Expr)
		g.line("%s %s %s = %s", s.Op, s.Name, g.typ(s.Type), v)
		g.line("_ = %s", s.Name)
	case "expr":
		v := g.expr(s.Expr)
		if v != "" {
			g.line("_ = %s", v)
		}
	case "assign":
		l := g.lvalue(s.Left)
		if s.Expr == nil {
			g.line("%s%s", l, s.Name)
		} else {
			v := g.expr(s.Expr)
			g.line("%s %s %s", l, s.Name, v)
		}
	case "send":
		l := g.freeze(s.Left)
		r := g.expr(s.Expr)
		g.line("%s <- %s", l, r)
	case "return":
		v := g.expr(s.Expr)
		g.line("return %s", v)
	case "if":
		v := g.expr(s.Expr)
		g.line("if %s {", v)
		g.level++
		g.statements(s.Body)
		g.level--
		if len(s.Else) > 0 {
			g.line("} else {")
			g.level++
			g.statements(s.Else)
			g.level--
		}
		g.line("}")
	case "foreach":
		v := g.freeze(s.Expr)
		header := s.Name + " := range " + v
		if s.Expr.Type.Name == "List" {
			v += ".items"
		}
		if s.Expr.Type.Name == "List" || s.Expr.Type.Name == "Array" {
			header = "_, " + s.Name + " := range " + v
		}
		g.line("for %s {", header)
		g.level++
		g.line("_ = %s", s.Name)
		g.statements(s.Body)
		g.level--
		g.line("}")
	case "break":
		g.line("break")
	case "continue":
		g.line("continue")
	case "go":
		if s.Expr != nil {
			g.line("go %s", g.call(s.Expr))
		} else {
			g.line("go func() {")
			g.level++
			old := g.ret
			g.ret = types.Void
			g.statements(s.Body)
			g.ret = old
			g.level--
			g.line("}()")
		}
	case "switch":
		v := g.freeze(s.Expr)
		g.line("switch %s {", v)
		g.level++
		for _, cs := range s.Cases {
			if cs.Default {
				g.line("default:")
			} else {
				g.line("case %s:", g.expr(cs.Expr))
			}
			g.level++
			g.statements(cs.Body)
			g.level--
		}
		g.level--
		g.line("}")
	case "select": // Channel operands and send values are evaluated once, in source order.
		headers := make([]string, len(s.Cases))
		for i, cs := range s.Cases {
			if cs.Default {
				headers[i] = "default:"
				continue
			}
			comm := cs.Comm
			if comm.Op == "send" {
				ch := g.freeze(comm.Left)
				v := g.freeze(comm.Expr)
				headers[i] = "case " + ch + " <- " + v + ":"
			} else {
				ch := g.freeze(comm.Expr)
				prefix := ""
				if comm.Name != "" {
					prefix = comm.Name + " := "
				}
				headers[i] = "case " + prefix + "<-" + ch + ":"
			}
		}
		g.line("select {")
		g.level++
		for i, cs := range s.Cases {
			g.line("%s", headers[i])
			g.level++
			if cs.Comm != nil && cs.Comm.Name != "" && cs.Comm.Op == "select_receive" {
				g.line("_ = %s", cs.Comm.Name)
			}
			g.statements(cs.Body)
			g.level--
		}
		g.level--
		g.line("}")
	}
}
