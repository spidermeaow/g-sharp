package checker

import (
	"fmt"
	"g-sharp/compiler/ast"
	"g-sharp/compiler/ir"
	"g-sharp/compiler/types"
	"go/constant"
	"strconv"
	"strings"
)

func (c *Checker) expr(e ast.Expr, expected *types.Type) *ir.Expr {
	if e == nil {
		return nil
	}
	r := &ir.Expr{Span: e.Location(), Type: types.Invalid}
	before := len(c.errors)
	defer func() {
		if len(c.errors) == before && r.Constant && types.Numeric(r.Type) {
			c.literalRange(r)
		}
	}()
	switch x := e.(type) {
	case *ast.Literal:
		r.Op = "literal"
		r.Value = x.Value
		if x.Kind == "integer" {
			r.Value = strings.TrimLeft(r.Value, "0")
			if r.Value == "" {
				r.Value = "0"
			}
		}
		r.Constant = true
		switch x.Kind {
		case "integer":
			r.Type = types.Int
		case "float":
			r.Type = &types.Type{Name: "float64"}
		case "string":
			r.Type = types.String
		case "char":
			r.Type = &types.Type{Name: "char"}
		default:
			r.Type = types.Bool
		}
		if types.Numeric(r.Type) {
			if expected != nil && types.Numeric(expected) {
				r.Type = expected
			}
		}
	case *ast.Name:
		if b, ok := c.scope.find(x.Value); ok {
			r.Op = "local"
			r.Name = b.name
			r.Type = b.typ
			r.Mutable = !b.constant
			r.Constant = b.constant
			if b.constant && b.value != nil {
				r.Op = "constant"
				r.Args = []*ir.Expr{b.value}
			}
			return r
		}
		if x.Value == "this" && c.owner != nil && c.fn != nil && !c.fn.Static {
			r.Op = "local"
			r.Name = "self"
			r.Type = &types.Type{Name: c.owner.SourceName}
			return r
		}
		if c.owner != nil && c.fn != nil && !c.fn.Static {
			for _, f := range c.owner.Fields {
				if f.SourceName == x.Value {
					return c.field(r, &ir.Expr{Op: "local", Name: "self", Type: &types.Type{Name: c.owner.SourceName}}, f, c.owner)
				}
			}
		}
		c.error(r.Span, "unknown variable '"+x.Value+"'")
	case *ast.New:
		r.Type = c.typ(x.Type)
		r.Op = "new"
		switch r.Type.Name {
		case "Error":
			c.arguments(r, x.Args, []*types.Type{types.String})
		case "List", "Map":
			c.arguments(r, x.Args, nil)
		case "Array":
			c.arguments(r, x.Args, []*types.Type{types.Int})
			if len(r.Args) > 0 {
				c.capacity(r.Args[0])
			}
			if !c.safeZero(r.Type.Args[0], map[string]bool{}) {
				c.error(r.Span, "array elements must have safe zero values")
			}
		default:
			d := c.decls[r.Type.Name]
			if d == nil || d.Kind == "interface" || d.Kind == "enum" {
				c.error(r.Span, "type cannot be constructed")
			}
			c.arguments(r, x.Args, nil)
		}
	case *ast.Unary:
		if x.Op == "-" {
			if lit, ok := x.Value.(*ast.Literal); ok && (lit.Kind == "integer" || lit.Kind == "float") {
				copy := *lit
				copy.Value = "-" + strings.TrimLeft(lit.Value, "0")
				if copy.Value == "-" {
					copy.Value = "0"
				}
				copy.ExprBase = x.ExprBase
				return c.expr(&copy, expected)
			}
		}
		var exp *types.Type
		if x.Op == "-" || x.Op == "+" {
			exp = expected
		}
		v := c.expr(x.Value, exp)
		r.Args = []*ir.Expr{v}
		r.Op = "unary"
		r.Name = x.Op
		r.Type = v.Type
		r.Constant = v.Constant
		switch x.Op {
		case "!":
			c.require(v, types.Bool)
			r.Type = types.Bool
		case "+", "-":
			if !types.Numeric(v.Type) {
				c.error(r.Span, "unary operator requires numeric operand")
			}
		case "<-":
			r.Op = "receive"
			r.Constant = false
			if v.Type.Name != "chan" {
				c.error(r.Span, "receive requires a channel")
			} else {
				r.Type = v.Type.Args[0]
			}
		case "?":
			r.Op = "propagate"
			r.Constant = false
			if v.Type.Name != "Result" {
				c.error(r.Span, "? requires a Result")
			} else {
				r.Type = v.Type.Args[0]
				if c.fn == nil || c.fn.Return.Name != "Result" || !types.Equal(v.Type.Args[1], c.fn.Return.Args[1]) {
					c.error(r.Span, "? requires a function returning Result with the same error type")
				}
			}
		}
	case *ast.Binary:
		left := c.expr(x.Left, nil)
		right := c.expr(x.Right, left.Type)
		r.Op = "binary"
		r.Name = x.Op
		r.Args = []*ir.Expr{left, right}
		r.Type = left.Type
		r.Constant = left.Constant && right.Constant
		if !types.Equal(left.Type, right.Type) {
			c.error(r.Span, "operator operands must have the same type")
		}
		switch x.Op {
		case "&&", "||":
			c.require(left, types.Bool)
			r.Type = types.Bool
		case "==", "!=":
			if !c.comparable(left.Type) {
				c.error(r.Span, "equality requires primitive or enum values")
			}
			r.Type = types.Bool
		case "<", "<=", ">", ">=":
			if !types.Numeric(left.Type) && !types.Equal(left.Type, types.String) {
				c.error(r.Span, "comparison requires numeric or string operands")
			}
			r.Type = types.Bool
		case "+":
			if !types.Numeric(left.Type) && !types.Equal(left.Type, types.String) {
				c.error(r.Span, "+ requires numbers or strings")
			}
		default:
			if !types.Numeric(left.Type) || (x.Op == "%" && !types.Integer(left.Type)) {
				c.error(r.Span, "invalid numeric operator")
			}
		}
		if (x.Op == "/" || x.Op == "%") && right.Constant {
			if v := constValue(right); v != nil && (v.Kind() == constant.Int || v.Kind() == constant.Float) && constant.Sign(v) == 0 {
				c.error(r.Span, "division by zero")
			}
		}
	case *ast.Member:
		if n, ok := x.Object.(*ast.Name); ok && !c.valueName(n.Value) {
			if d := c.decls[c.lookup(n.Value)]; d != nil && d.Kind == "enum" {
				for i, m := range d.Members {
					if m == x.Name {
						r.Op = "literal"
						r.Value = strconv.Itoa(i)
						r.Type = &types.Type{Name: d.SourceName}
						r.Constant = true
						return r
					}
				}
				c.error(r.Span, "unknown enum member "+x.Name)
				return r
			}
		}
		obj := c.expr(x.Object, nil)
		r.Args = []*ir.Expr{obj}
		if d := c.decls[obj.Type.Name]; d != nil {
			for _, f := range d.Fields {
				if f.SourceName == x.Name {
					return c.field(r, obj, f, d)
				}
			}
		}
		if x.Name == "Count" && (obj.Type.Name == "List" || obj.Type.Name == "Array" || obj.Type.Name == "Map" || obj.Type.Name == "string") {
			r.Op = "count"
			r.Type = types.Int
			return r
		}
		if obj.Type.Name == "Error" && x.Name == "Message" {
			r.Op = "error_message"
			r.Type = types.String
			return r
		}
		if obj.Type.Name == "Result" {
			r.Name = x.Name
			switch x.Name {
			case "IsOk":
				r.Op = "result_ok"
				r.Type = types.Bool
				return r
			case "Value":
				r.Op = "result_value"
				r.Type = obj.Type.Args[0]
				return r
			case "Error":
				r.Op = "result_error"
				r.Type = obj.Type.Args[1]
				return r
			}
		}
		c.error(r.Span, "unknown member '"+x.Name+"' on "+obj.Type.String())
	case *ast.Index:
		obj := c.expr(x.Object, nil)
		idxType := types.Int
		if obj.Type.Name == "Map" {
			idxType = obj.Type.Args[0]
		}
		idx := c.expr(x.Index, idxType)
		c.require(idx, idxType)
		r.Op = "index"
		r.Args = []*ir.Expr{obj, idx}
		r.Mutable = true
		switch obj.Type.Name {
		case "List", "Array":
			r.Type = obj.Type.Args[0]
		case "Map":
			r.Type = obj.Type.Args[1]
		default:
			c.error(r.Span, "indexing requires Array, List, or Map")
		}
	case *ast.Call:
		return c.call(x, expected)
	}
	return r
}
func (c *Checker) field(r, obj *ir.Expr, f *ir.Field, d *ir.TypeDecl) *ir.Expr {
	if f.Private && c.owner != d {
		c.error(r.Span, "field is private")
	}
	r.Op = "field"
	r.Name = f.Name
	r.Type = f.Type
	r.Args = []*ir.Expr{obj}
	r.Mutable = !f.ReadOnly && (!f.PrivateSet || c.owner == d) && (d.Kind == "class" || obj.Mutable || obj.Name == "self")
	if d.Kind == "struct" && obj.Op == "index" && obj.Args[0].Type.Name == "Map" {
		r.Mutable = false
	}
	return r
}
func (c *Checker) arguments(r *ir.Expr, args []ast.Expr, params []*types.Type) {
	if len(args) != len(params) {
		c.error(r.Span, fmt.Sprintf("expected %d arguments, got %d", len(params), len(args)))
	}
	for i, a := range args {
		var t *types.Type
		if i < len(params) {
			t = params[i]
		}
		e := c.expr(a, t)
		if t != nil {
			c.require(e, t)
		}
		r.Args = append(r.Args, e)
	}
}
func (c *Checker) call(x *ast.Call, expected *types.Type) *ir.Expr {
	r := &ir.Expr{Span: x.Location(), Type: types.Invalid, Op: "call"}
	var f *ir.Function
	if n, ok := x.Callee.(*ast.Name); ok {
		if n.Value == "chan" {
			if len(x.TypeArgs) != 1 {
				c.error(r.Span, "chan requires an element type")
				return r
			}
			r.Op = "channel"
			r.Type = &types.Type{Name: "chan", Args: []*types.Type{c.typ(x.TypeArgs[0])}}
			if types.Equal(r.Type.Args[0], types.Void) {
				c.error(r.Span, "void cannot be a channel element")
			}
			if len(x.Args) > 1 {
				c.error(r.Span, "channel accepts zero or one capacity")
			}
			for _, a := range x.Args {
				e := c.expr(a, types.Int)
				c.require(e, types.Int)
				c.capacity(e)
				r.Args = append(r.Args, e)
			}
			return r
		}
		f = c.funcs[c.lookup(n.Value)]
		if _, ok := c.scope.find(n.Value); ok {
			c.error(r.Span, "local value is not callable")
			return r
		}
		if f == nil && c.owner != nil {
			for _, m := range c.owner.Methods {
				if m.SourceName == n.Value {
					f = m
					if !m.Static {
						if c.fn == nil || c.fn.Static {
							c.error(r.Span, "instance method requires an object")
						}
						r.Op = "method"
						r.Args = append(r.Args, &ir.Expr{Op: "local", Name: "self", Type: &types.Type{Name: c.owner.SourceName}})
					}
					break
				}
			}
		}
	} else if m, ok := x.Callee.(*ast.Member); ok {
		if n, ok := m.Object.(*ast.Name); ok && !c.valueName(n.Value) {
			switch n.Value {
			case "Console":
				if m.Name == "WriteLine" || m.Name == "Write" {
					r.Op = "print"
					r.Name = m.Name
					r.Type = types.Void
					for _, a := range x.Args {
						e := c.expr(a, nil)
						if !c.comparable(e.Type) {
							c.error(e.Span, "Console accepts primitive or enum values")
						}
						r.Args = append(r.Args, e)
					}
					return r
				}
			case "Result":
				if m.Name == "Ok" || m.Name == "Err" {
					r.Op = "result"
					r.Name = m.Name
					if expected == nil || expected.Name != "Result" {
						c.error(r.Span, "Result constructor requires an expected Result<T,E> type")
						return r
					}
					r.Type = expected
					idx := 0
					if m.Name == "Err" {
						idx = 1
					}
					c.arguments(r, x.Args, []*types.Type{expected.Args[idx]})
					return r
				}
			case "Time":
				if m.Name == "Sleep" {
					r.Op = "sleep"
					r.Type = types.Void
					c.arguments(r, x.Args, []*types.Type{types.Int})
					return r
				}
			case "File":
				if m.Name == "ReadAllText" {
					r.Op = "read_file"
					r.Type = &types.Type{Name: "Result", Args: []*types.Type{types.String, {Name: "Error"}}}
					c.arguments(r, x.Args, []*types.Type{types.String})
					return r
				}
			case "Http":
				if m.Name == "Listen" {
					r.Op = "http_listen"
					r.Type = &types.Type{Name: "Result", Args: []*types.Type{types.Int, {Name: "Error"}}}
					c.arguments(r, x.Args, []*types.Type{types.String, types.String})
					return r
				}
			}
			if d := c.decls[c.lookup(n.Value)]; d != nil {
				for _, fn := range d.Methods {
					if fn.SourceName == m.Name && fn.Static {
						f = fn
						break
					}
				}
				if f == nil {
					c.error(r.Span, "unknown static method "+m.Name)
					return r
				}
			}
		}
		if f == nil {
			obj := c.expr(m.Object, nil)
			if obj.Type.Name == "List" && m.Name == "Add" {
				r.Op = "list_add"
				r.Type = types.Void
				r.Args = []*ir.Expr{obj}
				c.arguments(r, x.Args, []*types.Type{obj.Type.Args[0]})
				return r
			}
			d := c.decls[obj.Type.Name]
			if d != nil {
				for _, fn := range d.Methods {
					if fn.SourceName == m.Name && !fn.Static {
						f = fn
						r.Op = "method"
						r.Args = []*ir.Expr{obj}
						break
					}
				}
			}
		}
	}
	if f == nil {
		c.error(r.Span, "unknown function or method")
		return r
	}
	if f.Private && (c.owner == nil || c.owner.SourceName != f.Owner) {
		c.error(r.Span, "method is private")
	}
	r.Name = f.Name
	if r.Op == "method" {
		r.Name = methodName(f.SourceName)
	}
	r.Type = f.Return
	var params []*types.Type
	for _, p := range f.Parameters {
		params = append(params, p.Type)
	}
	c.arguments(r, x.Args, params)
	return r
}

// A stable method identity enables structural interfaces across unrelated types.
func methodName(n string) string { return "M_" + n }
