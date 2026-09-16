package types

import "strings"

type Type struct {
	Name string
	Args []*Type
}

func (t *Type) String() string {
	if t == nil {
		return "<invalid>"
	}
	if len(t.Args) == 0 {
		return t.Name
	}
	var a []string
	for _, x := range t.Args {
		a = append(a, x.String())
	}
	return t.Name + "<" + strings.Join(a, ", ") + ">"
}
func Equal(a, b *Type) bool {
	if a == nil || b == nil {
		return false
	}
	if a.Name != b.Name || len(a.Args) != len(b.Args) {
		return false
	}
	for i := range a.Args {
		if !Equal(a.Args[i], b.Args[i]) {
			return false
		}
	}
	return true
}

var Invalid = &Type{Name: "<invalid>"}
var Void = &Type{Name: "void"}
var Int = &Type{Name: "int"}
var Bool = &Type{Name: "bool"}
var String = &Type{Name: "string"}
var Primitives = map[string]bool{"bool": true, "byte": true, "int8": true, "int16": true, "int32": true, "int64": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true, "int": true, "uint": true, "float32": true, "float64": true, "char": true, "string": true, "void": true, "Error": true}

func Numeric(t *Type) bool {
	return t != nil && (Integer(t) || t.Name == "float32" || t.Name == "float64")
}
func Integer(t *Type) bool {
	if t == nil {
		return false
	}
	switch t.Name {
	case "int", "uint", "int8", "int16", "int32", "int64", "uint8", "uint16", "uint32", "uint64", "char":
		return true
	}
	return false
}
