// Package ir defines backend-neutral typed operations. It does not import AST.
package ir

import (
	"g-sharp/compiler/token"
	"g-sharp/compiler/types"
)

type Expr struct {
	Span              token.Span
	Type              *types.Type
	Op, Name, Value   string
	Args              []*Expr
	Constant, Mutable bool
}
type Stmt struct {
	Span       token.Span
	Op, Name   string
	Type       *types.Type
	Expr, Left *Expr
	Body, Else []*Stmt
	Init, Post *Stmt
	Cases      []Case
}
type Case struct {
	Default bool
	Expr    *Expr
	Comm    *Stmt
	Body    []*Stmt
}
type Parameter struct {
	Name string
	Type *types.Type
}
type Function struct {
	Name, SourceName, Owner string
	Return                  *types.Type
	Parameters              []Parameter
	Body                    []*Stmt
	Static, Private         bool
}
type Field struct {
	Name, SourceName              string
	Type                          *types.Type
	Init                          *Expr
	Private, PrivateSet, ReadOnly bool
}
type TypeDecl struct {
	Name, SourceName, Kind string
	Fields                 []*Field
	Methods                []*Function
	Members                []string
}
type Program struct {
	Types     []*TypeDecl
	Functions []*Function
	Entry     *Function
}
