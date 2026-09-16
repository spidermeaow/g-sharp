package token

type Position struct {
	File                 string
	Offset, Line, Column int
}
type Span struct{ Start, End Position }
type Kind string

const (
	EOF     Kind = "eof"
	Ident   Kind = "identifier"
	Integer Kind = "integer"
	Float   Kind = "float"
	String  Kind = "string"
	Char    Kind = "char"
	Comment Kind = "comment"
)

type Token struct {
	Kind Kind
	Text string
	Span Span
}

var Keywords = map[string]bool{}

func init() {
	for _, s := range []string{"namespace", "using", "class", "struct", "interface", "enum", "static", "private", "public", "get", "set", "var", "const", "if", "else", "for", "foreach", "in", "switch", "case", "default", "return", "break", "continue", "go", "select", "new", "true", "false", "null", "while"} {
		Keywords[s] = true
	}
}
