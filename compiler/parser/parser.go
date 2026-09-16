package parser

import (
	"fmt"
	"g-sharp/compiler/ast"
	"g-sharp/compiler/diagnostics"
	"g-sharp/compiler/token"
)

type Parser struct {
	ts     []token.Token
	i      int
	errors diagnostics.List
}
type syntaxError struct{}

func Parse(ts []token.Token) (*ast.File, diagnostics.List) {
	p := &Parser{}
	for _, t := range ts {
		if t.Kind != token.Comment {
			p.ts = append(p.ts, t)
		}
	}
	if len(p.ts) == 0 {
		return &ast.File{}, nil
	}
	f := &ast.File{}
	declarationsStarted := false
	for !p.at("eof") {
		start := p.i
		func() {
			defer func() {
				if r := recover(); r != nil {
					if _, ok := r.(syntaxError); !ok {
						panic(r)
					}
					p.sync()
				}
			}()
			switch p.cur().Text {
			case "namespace":
				if f.Namespace != "" || declarationsStarted {
					p.fail("namespace must appear once, before declarations")
				}
				p.take()
				f.Namespace = p.qualified()
				p.want(";")
			case "using":
				if declarationsStarted {
					p.fail("using directives must precede declarations")
				}
				p.take()
				f.Usings = append(f.Usings, p.qualified())
				p.want(";")
			case "class", "struct", "interface", "enum":
				declarationsStarted = true
				f.Types = append(f.Types, p.typeDecl())
			default:
				declarationsStarted = true
				if p.functionAhead() {
					f.Functions = append(f.Functions, p.function(false, false, false))
				} else {
					f.Statements = append(f.Statements, p.stmt())
				}
			}
		}()
		if start == p.i {
			p.take()
		}
	}
	f.Node = ast.Node{Span: token.Span{Start: p.ts[0].Span.Start, End: p.cur().Span.End}}
	return f, p.errors
}
func (p *Parser) cur() token.Token { return p.ts[p.i] }
func (p *Parser) at(s string) bool { return string(p.cur().Kind) == s }
func (p *Parser) take() token.Token {
	t := p.cur()
	if p.i < len(p.ts)-1 {
		p.i++
	}
	return t
}
func (p *Parser) eat(s string) bool {
	if p.at(s) {
		p.take()
		return true
	}
	return false
}
func (p *Parser) want(s string) token.Token {
	if !p.at(s) {
		p.fail("expected " + s + ", found " + fmt.Sprintf("%q", p.cur().Text))
	}
	return p.take()
}
func (p *Parser) fail(msg string) {
	p.errors = append(p.errors, diagnostics.Diagnostic{Code: "GS1101", Message: msg, Span: p.cur().Span})
	panic(syntaxError{})
}
func (p *Parser) sync() {
	for !p.at("eof") {
		if p.eat(";") || p.eat("}") {
			return
		}
		p.take()
	}
}
func (p *Parser) span(start token.Token) token.Span {
	end := start.Span.End
	if p.i > 0 {
		end = p.ts[p.i-1].Span.End
	}
	return token.Span{Start: start.Span.Start, End: end}
}
func (p *Parser) node(t token.Token) ast.Node   { return ast.Node{Span: p.span(t)} }
func (p *Parser) sb(t token.Token) ast.StmtBase { return ast.StmtBase{Node: p.node(t)} }
func (p *Parser) eb(t token.Token) ast.ExprBase { return ast.ExprBase{Node: p.node(t)} }
func (p *Parser) qualified() string {
	s := p.want("identifier").Text
	for p.eat(".") {
		s += "." + p.want("identifier").Text
	}
	return s
}
func (p *Parser) typ() ast.TypeRef {
	t := p.want("identifier")
	r := ast.TypeRef{Name: t.Text}
	if p.eat("<") {
		for {
			r.Args = append(r.Args, p.typ())
			if !p.eat(",") {
				break
			}
		}
		p.want(">")
	}
	r.Node = p.node(t)
	return r
}
func (p *Parser) typeParams() []string {
	var r []string
	if p.eat("<") {
		for {
			r = append(r, p.want("identifier").Text)
			if !p.eat(",") {
				break
			}
		}
		p.want(">")
	}
	return r
}
func (p *Parser) typeDecl() *ast.TypeDecl {
	t := p.take()
	d := &ast.TypeDecl{Kind: t.Text, Name: p.want("identifier").Text}
	d.Parameters = p.typeParams()
	p.want("{")
	if d.Kind == "enum" {
		for !p.at("}") && !p.at("eof") {
			d.Members = append(d.Members, p.want("identifier").Text)
			if !p.eat(",") {
				break
			}
		}
	} else {
		for !p.at("}") && !p.at("eof") {
			private, static := false, false
			for p.at("public") || p.at("private") || p.at("static") {
				m := p.take().Text
				private = private || m == "private"
				static = static || m == "static"
			}
			if p.functionAhead() {
				d.Methods = append(d.Methods, p.function(static, private, d.Kind == "interface"))
				continue
			}
			if d.Kind == "interface" {
				p.fail("interfaces contain method signatures")
			}
			ft := p.cur()
			ty := p.typ()
			name := p.want("identifier").Text
			f := &ast.Field{Type: ty, Name: name, Private: private}
			if static {
				p.fail("static fields are not supported")
			}
			if p.eat("{") {
				f.Property = true
				p.want("get")
				p.want(";")
				if !p.at("}") {
					f.PrivateSet = p.eat("private")
					p.want("set")
					p.want(";")
				} else {
					f.ReadOnly = true
				}
				p.want("}")
			} else {
				if p.eat("=") {
					f.Init = p.expr(0)
				}
				p.want(";")
			}
			f.Node = p.node(ft)
			d.Fields = append(d.Fields, f)
		}
	}
	p.want("}")
	d.Node = p.node(t)
	return d
}

// Look ahead through a type reference without changing parser state.
func (p *Parser) functionAhead() bool {
	j := p.i
	if j >= len(p.ts) || p.ts[j].Kind != token.Ident {
		return false
	}
	j++
	depth := 0
	if j < len(p.ts) && p.ts[j].Text == "<" {
		for j < len(p.ts) {
			s := p.ts[j].Text
			j++
			if s == "<" {
				depth++
			}
			if s == ">" {
				depth--
				if depth == 0 {
					break
				}
			}
		}
	}
	if j >= len(p.ts) || p.ts[j].Kind != token.Ident {
		return false
	}
	j++
	if j < len(p.ts) && p.ts[j].Text == "<" {
		depth = 0
		for j < len(p.ts) {
			s := p.ts[j].Text
			j++
			if s == "<" {
				depth++
			}
			if s == ">" {
				depth--
				if depth == 0 {
					break
				}
			}
		}
	}
	return j < len(p.ts) && p.ts[j].Text == "("
}
func (p *Parser) function(static, private, signature bool) *ast.Function {
	t := p.cur()
	f := &ast.Function{Return: p.typ(), Static: static, Private: private}
	f.Name = p.want("identifier").Text
	f.TypeParameters = p.typeParams()
	p.want("(")
	if !p.at(")") {
		for {
			pt := p.cur()
			ty := p.typ()
			name := p.want("identifier").Text
			f.Parameters = append(f.Parameters, ast.Parameter{Node: p.node(pt), Type: ty, Name: name})
			if !p.eat(",") {
				break
			}
		}
	}
	p.want(")")
	if signature {
		p.want(";")
	} else {
		f.Body = p.block()
	}
	f.Node = p.node(t)
	return f
}
func (p *Parser) block() *ast.Block {
	t := p.want("{")
	b := &ast.Block{}
	for !p.at("}") && !p.at("eof") {
		b.Statements = append(b.Statements, p.stmt())
	}
	p.want("}")
	b.StmtBase = p.sb(t)
	return b
}
func (p *Parser) stmt() ast.Stmt {
	t := p.cur()
	switch t.Text {
	case "{":
		return p.block()
	case "if":
		p.take()
		p.want("(")
		c := p.expr(0)
		p.want(")")
		a := p.stmt()
		var b ast.Stmt
		if p.eat("else") {
			b = p.stmt()
		}
		return &ast.If{StmtBase: p.sb(t), Condition: c, Then: a, Else: b}
	case "for":
		p.take()
		p.want("(")
		var init, post ast.Stmt
		var c ast.Expr
		if !p.at(";") {
			init = p.simple()
		}
		p.want(";")
		if !p.at(";") {
			c = p.expr(0)
		}
		p.want(";")
		if !p.at(")") {
			post = p.simple()
		}
		p.want(")")
		body := p.stmt()
		return &ast.For{StmtBase: p.sb(t), Init: init, Condition: c, Post: post, Body: body}
	case "foreach":
		p.take()
		p.want("(")
		var ty ast.TypeRef
		if p.eat("var") {
			ty.Name = "var"
		} else {
			ty = p.typ()
		}
		name := p.want("identifier").Text
		p.want("in")
		e := p.expr(0)
		p.want(")")
		b := p.stmt()
		return &ast.Foreach{StmtBase: p.sb(t), Name: name, Type: ty, Collection: e, Body: b}
	case "return":
		p.take()
		var e ast.Expr
		if !p.at(";") {
			e = p.expr(0)
		}
		p.want(";")
		return &ast.Return{StmtBase: p.sb(t), Value: e}
	case "break", "continue":
		p.take()
		p.want(";")
		return &ast.Branch{StmtBase: p.sb(t), Kind: t.Text}
	case "go":
		p.take()
		g := &ast.Go{}
		if p.at("{") {
			g.Body = p.block()
		} else {
			g.Call = p.expr(0)
		}
		p.want(";")
		g.StmtBase = p.sb(t)
		return g
	case "switch", "select":
		return p.switchStmt()
	}
	s := p.simple()
	p.want(";")
	return s
}
func (p *Parser) variableAhead() bool {
	if p.at("var") || p.at("const") {
		return true
	}
	j := p.i
	if p.ts[j].Kind != token.Ident {
		return false
	}
	j++
	if j < len(p.ts) && p.ts[j].Text == "<" {
		depth := 0
		for j < len(p.ts) {
			s := p.ts[j].Text
			j++
			if s == "<" {
				depth++
			}
			if s == ">" {
				depth--
				if depth == 0 {
					break
				}
			}
		}
	}
	return j < len(p.ts) && p.ts[j].Kind == token.Ident
}
func (p *Parser) simple() ast.Stmt {
	t := p.cur()
	if p.variableAhead() {
		constant := p.eat("const")
		var ty ast.TypeRef
		if p.eat("var") {
			ty.Name = "var"
		} else {
			ty = p.typ()
		}
		name := p.want("identifier").Text
		p.want("=")
		e := p.expr(0)
		return &ast.Variable{StmtBase: p.sb(t), Name: name, Type: ty, Init: e, Constant: constant}
	}
	e := p.expr(0)
	switch p.cur().Text {
	case "=", "+=", "-=", "*=", "/=", "<-":
		op := p.take().Text
		r := p.expr(0)
		return &ast.Assign{StmtBase: p.sb(t), Op: op, Left: e, Right: r}
	case "++", "--":
		op := p.take().Text
		return &ast.Assign{StmtBase: p.sb(t), Op: op, Left: e}
	}
	return &ast.ExpressionStmt{StmtBase: p.sb(t), Expr: e}
}
func (p *Parser) switchStmt() ast.Stmt {
	t := p.take()
	s := &ast.Switch{Select: t.Text == "select"}
	if !s.Select {
		p.want("(")
		s.Value = p.expr(0)
		p.want(")")
	}
	p.want("{")
	for !p.at("}") && !p.at("eof") {
		ct := p.cur()
		c := ast.Case{}
		if p.eat("default") {
			c.Default = true
		} else {
			p.want("case")
			if s.Select {
				c.Comm = p.simple()
			} else {
				c.Value = p.expr(0)
			}
		}
		p.want(":")
		for !p.at("case") && !p.at("default") && !p.at("}") && !p.at("eof") {
			c.Body = append(c.Body, p.stmt())
		}
		c.Node = p.node(ct)
		s.Cases = append(s.Cases, c)
	}
	p.want("}")
	s.StmtBase = p.sb(t)
	return s
}

var prec = map[string]int{"||": 1, "&&": 2, "==": 3, "!=": 3, "<": 4, "<=": 4, ">": 4, ">=": 4, "+": 5, "-": 5, "*": 6, "/": 6, "%": 6}

func (p *Parser) expr(min int) ast.Expr {
	t := p.cur()
	var e ast.Expr
	switch t.Kind {
	case token.Integer, token.Float, token.String, token.Char, "true", "false":
		p.take()
		e = &ast.Literal{ExprBase: p.eb(t), Kind: string(t.Kind), Value: t.Text}
	case token.Ident:
		p.take()
		e = &ast.Name{ExprBase: p.eb(t), Value: t.Text}
	case "new":
		p.take()
		ty := p.typ()
		args := p.args()
		e = &ast.New{ExprBase: p.eb(t), Type: ty, Args: args}
	case "(":
		p.take()
		e = p.expr(0)
		p.want(")")
	case "!", "-", "+", "<-":
		p.take()
		v := p.expr(7)
		e = &ast.Unary{ExprBase: p.eb(t), Op: t.Text, Value: v}
	default:
		p.fail("expected expression")
	}
	for {
		if p.eat(".") {
			name := p.want("identifier").Text
			e = &ast.Member{ExprBase: p.eb(t), Object: e, Name: name}
			continue
		}
		if p.at("(") {
			args := p.args()
			e = &ast.Call{ExprBase: p.eb(t), Callee: e, Args: args}
			continue
		}
		if n, ok := e.(*ast.Name); ok && n.Value == "chan" && p.eat("<") {
			ty := p.typ()
			p.want(">")
			args := p.args()
			e = &ast.Call{ExprBase: p.eb(t), Callee: e, TypeArgs: []ast.TypeRef{ty}, Args: args}
			continue
		}
		if p.eat("[") {
			idx := p.expr(0)
			p.want("]")
			e = &ast.Index{ExprBase: p.eb(t), Object: e, Index: idx}
			continue
		}
		if p.eat("?") {
			e = &ast.Unary{ExprBase: p.eb(t), Op: "?", Value: e}
			continue
		}
		level := prec[p.cur().Text]
		if level == 0 || level < min {
			break
		}
		op := p.take().Text
		right := p.expr(level + 1)
		e = &ast.Binary{ExprBase: p.eb(t), Op: op, Left: e, Right: right}
	}
	return e
}
func (p *Parser) args() []ast.Expr {
	p.want("(")
	var args []ast.Expr
	if !p.at(")") {
		for {
			args = append(args, p.expr(0))
			if !p.eat(",") {
				break
			}
		}
	}
	p.want(")")
	return args
}
