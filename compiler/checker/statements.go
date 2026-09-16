package checker

import (
	"g-sharp/compiler/ast"
	"g-sharp/compiler/ir"
	"g-sharp/compiler/types"
)

func (c *Checker) statements(ss []ast.Stmt) []*ir.Stmt {
	var out []*ir.Stmt
	ended := false
	for _, s := range ss {
		if ended {
			c.error(s.Location(), "unreachable statement")
		}
		r := c.stmt(s)
		out = append(out, r)
		ended = terminates(r)
	}
	return out
}
func (c *Checker) nested(s ast.Stmt) []*ir.Stmt {
	if s == nil {
		return nil
	}
	c.push()
	defer c.pop()
	if b, ok := s.(*ast.Block); ok {
		return c.statements(b.Statements)
	}
	return []*ir.Stmt{c.stmt(s)}
}
func (c *Checker) stmt(s ast.Stmt) *ir.Stmt {
	if s == nil {
		return nil
	}
	r := &ir.Stmt{Span: s.Location()}
	switch x := s.(type) {
	case *ast.Block:
		r.Op = "block"
		r.Body = c.nested(x)
	case *ast.Variable:
		r.Op = "var"
		if x.Constant {
			r.Op = "const"
		}
		var ty *types.Type
		if x.Type.Name != "var" {
			ty = c.typ(x.Type)
		}
		r.Expr = c.expr(x.Init, ty)
		if ty == nil {
			ty = r.Expr.Type
		}
		if types.Equal(ty, types.Void) {
			c.error(r.Span, "variable cannot have type void")
		}
		c.require(r.Expr, ty)
		if x.Constant && !r.Expr.Constant {
			c.error(r.Span, "const requires a constant expression")
		}
		r.Type = ty
		r.Name = c.fresh(x.Name)
		c.define(x.Name, binding{name: r.Name, typ: ty, constant: x.Constant, value: r.Expr}, r.Span)
	case *ast.ExpressionStmt:
		r.Op = "expr"
		r.Expr = c.expr(x.Expr, nil)
		switch r.Expr.Op {
		case "call", "method", "print", "list_add", "sleep", "read_file", "http_listen", "receive":
		default:
			c.error(r.Span, "expression statement must be a call or receive")
		}
	case *ast.Assign:
		r.Op = "assign"
		r.Name = x.Op
		r.Left = c.expr(x.Left, nil)
		if x.Op == "<-" {
			r.Op = "send"
			if r.Left.Type.Name != "chan" {
				c.error(r.Span, "send requires a channel")
				r.Expr = c.expr(x.Right, nil)
			} else {
				r.Expr = c.expr(x.Right, r.Left.Type.Args[0])
				c.require(r.Expr, r.Left.Type.Args[0])
			}
			break
		}
		if !r.Left.Mutable {
			c.error(r.Span, "assignment target is not mutable storage")
		}
		if x.Right != nil {
			r.Expr = c.expr(x.Right, r.Left.Type)
			c.require(r.Expr, r.Left.Type)
			if x.Op == "/=" {
				if v := constValue(r.Expr); v != nil && v.ExactString() == "0" {
					c.error(r.Span, "division by zero")
				}
			}
		}
		if x.Op != "=" {
			if !types.Numeric(r.Left.Type) && !(x.Op == "+=" && types.Equal(r.Left.Type, types.String)) {
				c.error(r.Span, "compound assignment requires numeric operands")
			}
		}
	case *ast.If:
		r.Op = "if"
		r.Expr = c.expr(x.Condition, types.Bool)
		c.require(r.Expr, types.Bool)
		r.Body = c.nested(x.Then)
		r.Else = c.nested(x.Else)
	case *ast.For:
		r.Op = "for"
		c.push()
		r.Init = c.stmt(x.Init)
		if x.Condition != nil {
			r.Expr = c.expr(x.Condition, types.Bool)
			c.require(r.Expr, types.Bool)
		}
		c.loops++
		c.breaks++
		r.Body = c.nested(x.Body)
		r.Post = c.stmt(x.Post)
		if r.Post != nil && r.Post.Op != "assign" && r.Post.Op != "expr" {
			c.error(r.Span, "for update must be assignment or call")
		}
		c.loops--
		c.breaks--
		c.pop()
	case *ast.Foreach:
		r.Op = "foreach"
		r.Expr = c.expr(x.Collection, nil)
		ty := types.Invalid
		switch r.Expr.Type.Name {
		case "Array", "List", "chan":
			ty = r.Expr.Type.Args[0]
		case "Map":
			ty = r.Expr.Type.Args[0]
		default:
			c.error(r.Span, "foreach requires a collection or channel")
		}
		if x.Type.Name != "var" {
			decl := c.typ(x.Type)
			if !types.Equal(decl, ty) {
				c.error(r.Span, "foreach element type mismatch")
			}
		}
		r.Type = ty
		r.Name = c.fresh(x.Name)
		c.push()
		c.define(x.Name, binding{name: r.Name, typ: ty}, r.Span)
		c.loops++
		c.breaks++
		r.Body = c.nested(x.Body)
		c.loops--
		c.breaks--
		c.pop()
	case *ast.Return:
		r.Op = "return"
		if x.Value == nil {
			if !types.Equal(c.fn.Return, types.Void) {
				c.error(r.Span, "return requires a value")
			}
		} else {
			r.Expr = c.expr(x.Value, c.fn.Return)
			c.require(r.Expr, c.fn.Return)
			if types.Equal(c.fn.Return, types.Void) {
				c.error(r.Span, "void function cannot return a value")
			}
		}
	case *ast.Branch:
		r.Op = x.Kind
		if (x.Kind == "break" && c.breaks == 0) || (x.Kind == "continue" && c.loops == 0) {
			c.error(r.Span, x.Kind+" is not valid in this context")
		}
	case *ast.Go:
		r.Op = "go"
		if x.Call != nil {
			if _, ok := x.Call.(*ast.Call); !ok {
				c.error(r.Span, "go requires a call")
			}
			r.Expr = c.expr(x.Call, nil)
			if r.Expr.Op == "result" || r.Expr.Op == "channel" {
				c.error(r.Span, "go requires an executable function call")
			}
		} else {
			old, loops, breaks := c.fn, c.loops, c.breaks
			c.fn = &ir.Function{Return: types.Void, Static: old.Static}
			c.loops = 0
			c.breaks = 0
			r.Body = c.nested(x.Body)
			c.fn = old
			c.loops = loops
			c.breaks = breaks
		}
	case *ast.Switch:
		r.Op = "switch"
		if x.Select {
			r.Op = "select"
		} else {
			r.Expr = c.expr(x.Value, nil)
			if !c.comparable(r.Expr.Type) {
				c.error(r.Span, "switch requires a comparable value")
			}
		}
		defaults := 0
		seen := map[string]bool{}
		c.breaks++
		for _, a := range x.Cases {
			cs := ir.Case{Default: a.Default}
			c.push()
			if a.Default {
				defaults++
				if defaults > 1 {
					c.error(a.Span, "duplicate default case")
				}
			} else if x.Select {
				cs.Comm = c.selectComm(a.Comm)
			} else {
				cs.Expr = c.expr(a.Value, r.Expr.Type)
				c.require(cs.Expr, r.Expr.Type)
				if !cs.Expr.Constant {
					c.error(a.Span, "switch case must be constant")
				}
				key := ""
				if v := constValue(cs.Expr); v != nil {
					key = v.ExactString()
				}
				if key != "" && seen[key] {
					c.error(a.Span, "duplicate switch case")
				}
				seen[key] = true
			}
			cs.Body = c.statements(a.Body)
			c.pop()
			r.Cases = append(r.Cases, cs)
		}
		c.breaks--
	}
	return r
}
func (c *Checker) selectComm(s ast.Stmt) *ir.Stmt {
	if a, ok := s.(*ast.Assign); ok && a.Op == "=" {
		if name, ok := a.Left.(*ast.Name); ok {
			if u, ok := a.Right.(*ast.Unary); ok && u.Op == "<-" {
				v := c.expr(u, nil)
				n := c.fresh(name.Value)
				c.define(name.Value, binding{name: n, typ: v.Type}, a.Location())
				return &ir.Stmt{Op: "select_receive", Name: n, Type: v.Type, Expr: v.Args[0], Span: a.Location()}
			}
		}
	}
	r := c.stmt(s)
	if r.Op == "send" {
		return r
	}
	if r.Op == "expr" && r.Expr.Op == "receive" {
		return &ir.Stmt{Op: "select_receive", Expr: r.Expr.Args[0], Span: s.Location()}
	}
	c.error(s.Location(), "select case must send or receive a channel value")
	return r
}
func terminates(s *ir.Stmt) bool {
	if s == nil {
		return false
	}
	switch s.Op {
	case "return", "break", "continue":
		return true
	case "block":
		return ends(s.Body)
	case "if":
		return len(s.Else) > 0 && ends(s.Body) && ends(s.Else)
	}
	return false
}
func ends(ss []*ir.Stmt) bool { return len(ss) > 0 && terminates(ss[len(ss)-1]) }
func returns(ss []*ir.Stmt) bool {
	if len(ss) == 0 {
		return false
	}
	s := ss[len(ss)-1]
	switch s.Op {
	case "return":
		return true
	case "block":
		return returns(s.Body)
	case "if":
		return returns(s.Body) && returns(s.Else)
	case "switch", "select":
		hasDefault := false
		for _, cs := range s.Cases {
			hasDefault = hasDefault || cs.Default
			if !returns(cs.Body) {
				return false
			}
		}
		return hasDefault
	}
	return false
}
