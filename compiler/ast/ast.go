package ast

import "g-sharp/compiler/token"

type Node struct{ Span token.Span }
type TypeRef struct {
	Node
	Name string
	Args []TypeRef
}
type File struct {
	Node
	Namespace  string
	Usings     []string
	Types      []*TypeDecl
	Functions  []*Function
	Statements []Stmt
}
type TypeDecl struct {
	Node
	Kind, Name string
	Parameters []string
	Fields     []*Field
	Methods    []*Function
	Members    []string
}
type Field struct {
	Node
	Name                                    string
	Type                                    TypeRef
	Init                                    Expr
	Property, Private, PrivateSet, ReadOnly bool
}
type Parameter struct {
	Node
	Name string
	Type TypeRef
}
type Function struct {
	Node
	Name            string
	Return          TypeRef
	Parameters      []Parameter
	TypeParameters  []string
	Body            *Block
	Static, Private bool
}
type Expr interface {
	Expression()
	Location() token.Span
}
type ExprBase struct{ Node }

func (ExprBase) Expression()            {}
func (e ExprBase) Location() token.Span { return e.Span }

type Literal struct {
	ExprBase
	Kind, Value string
}
type Name struct {
	ExprBase
	Value string
}
type Unary struct {
	ExprBase
	Op    string
	Value Expr
}
type Binary struct {
	ExprBase
	Op          string
	Left, Right Expr
}
type Member struct {
	ExprBase
	Object Expr
	Name   string
}
type Call struct {
	ExprBase
	Callee   Expr
	Args     []Expr
	TypeArgs []TypeRef
}
type New struct {
	ExprBase
	Type TypeRef
	Args []Expr
}
type Index struct {
	ExprBase
	Object, Index Expr
}
type Stmt interface {
	Statement()
	Location() token.Span
}
type StmtBase struct{ Node }

func (StmtBase) Statement()             {}
func (s StmtBase) Location() token.Span { return s.Span }

type Block struct {
	StmtBase
	Statements []Stmt
}
type Variable struct {
	StmtBase
	Name     string
	Type     TypeRef
	Init     Expr
	Constant bool
}
type ExpressionStmt struct {
	StmtBase
	Expr Expr
}
type Assign struct {
	StmtBase
	Op          string
	Left, Right Expr
}
type If struct {
	StmtBase
	Condition  Expr
	Then, Else Stmt
}
type For struct {
	StmtBase
	Init, Post Stmt
	Condition  Expr
	Body       Stmt
}
type Foreach struct {
	StmtBase
	Name       string
	Type       TypeRef
	Collection Expr
	Body       Stmt
}
type Return struct {
	StmtBase
	Value Expr
}
type Branch struct {
	StmtBase
	Kind string
}
type Go struct {
	StmtBase
	Call Expr
	Body *Block
}
type Case struct {
	Node
	Value   Expr
	Comm    Stmt
	Body    []Stmt
	Default bool
}
type Switch struct {
	StmtBase
	Value  Expr
	Cases  []Case
	Select bool
}
