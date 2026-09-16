package checker

import (
	"g-sharp/compiler/ir"
	"g-sharp/compiler/types"
	"go/constant"
	gotoken "go/token"
	"math"
	"strings"
)

func (c *Checker) literalRange(e *ir.Expr) {
	v := constValue(e)
	if v == nil {
		return
	}
	if types.Integer(e.Type) {
		v = constant.ToInt(v)
		if v.Kind() == constant.Unknown {
			c.error(e.Span, "integer literal required")
			return
		}
		bits := 64
		signed := true
		switch e.Type.Name {
		case "int8":
			bits = 8
		case "int16":
			bits = 16
		case "int32", "char":
			bits = 32
		case "uint8":
			bits = 8
			signed = false
		case "uint16":
			bits = 16
			signed = false
		case "uint32":
			bits = 32
			signed = false
		case "uint", "uint64":
			signed = false
		}
		if signed {
			i, ok := constant.Int64Val(v)
			if !ok || (bits < 64 && (i < -(int64(1)<<(bits-1)) || i > (int64(1)<<(bits-1))-1)) {
				c.error(e.Span, "integer literal out of range for "+e.Type.String())
			}
		} else {
			u, ok := constant.Uint64Val(v)
			if !ok || (bits < 64 && u > uint64(1)<<bits-1) {
				c.error(e.Span, "integer literal out of range for "+e.Type.String())
			}
		}
	} else if e.Type.Name == "float32" {
		f, _ := constant.Float64Val(v)
		if f > 3.4028234663852886e38 || f < -3.4028234663852886e38 {
			c.error(e.Span, "float literal out of range")
		}
	} else if e.Type.Name == "float64" {
		f, _ := constant.Float64Val(v)
		if math.IsInf(f, 0) {
			c.error(e.Span, "float literal out of range")
		}
	}
}
func constValue(e *ir.Expr) (result constant.Value) {
	// Invalid expressions already have diagnostics. Constant evaluation must not
	// turn an erroneous source program into a compiler panic.
	defer func() {
		if recover() != nil {
			result = nil
		}
	}()
	if e == nil || !e.Constant {
		return nil
	}
	if e.Op == "constant" {
		return constValue(e.Args[0])
	}
	if e.Op == "unary" {
		v := constValue(e.Args[0])
		if v == nil {
			return nil
		}
		op := map[string]gotoken.Token{"-": gotoken.SUB, "+": gotoken.ADD, "!": gotoken.NOT}[e.Name]
		if op == 0 {
			return nil
		}
		return constant.UnaryOp(op, v, 0)
	}
	if e.Op == "binary" {
		a, b := constValue(e.Args[0]), constValue(e.Args[1])
		if a == nil || b == nil {
			return nil
		}
		op := map[string]gotoken.Token{"+": gotoken.ADD, "-": gotoken.SUB, "*": gotoken.MUL, "/": gotoken.QUO, "%": gotoken.REM, "&&": gotoken.LAND, "||": gotoken.LOR, "==": gotoken.EQL, "!=": gotoken.NEQ, "<": gotoken.LSS, "<=": gotoken.LEQ, ">": gotoken.GTR, ">=": gotoken.GEQ}[e.Name]
		if op == 0 {
			return nil
		}
		if op >= gotoken.EQL && op <= gotoken.GEQ {
			return constant.MakeBool(constant.Compare(a, op, b))
		}
		if (op == gotoken.QUO || op == gotoken.REM) && constant.Sign(b) == 0 {
			return nil
		}
		if op == gotoken.QUO && types.Integer(e.Type) {
			op = gotoken.QUO_ASSIGN
		}
		return constant.BinaryOp(a, op, b)
	}
	if e.Op != "literal" {
		return nil
	}
	k := gotoken.INT
	if strings.HasPrefix(e.Value, "\"") {
		k = gotoken.STRING
	} else if strings.HasPrefix(e.Value, "'") {
		k = gotoken.CHAR
	} else if strings.Contains(e.Value, ".") {
		k = gotoken.FLOAT
	}
	if e.Value == "true" || e.Value == "false" {
		return constant.MakeBool(e.Value == "true")
	}
	value := e.Value
	negative := strings.HasPrefix(value, "-")
	if negative {
		value = strings.TrimPrefix(value, "-")
	}
	v := constant.MakeFromLiteral(value, k, 0)
	if negative {
		v = constant.UnaryOp(gotoken.SUB, v, 0)
	}
	return v
}
