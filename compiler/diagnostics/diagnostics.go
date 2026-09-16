package diagnostics

import (
	"fmt"
	"g-sharp/compiler/token"
	"strings"
)

type Diagnostic struct {
	Code, Message string
	Span          token.Span
}

func (d Diagnostic) Error() string {
	return fmt.Sprintf("%s:%d:%d: error %s: %s", d.Span.Start.File, d.Span.Start.Line, d.Span.Start.Column, d.Code, d.Message)
}

type List []Diagnostic

func (l List) Error() string {
	var out []string
	for _, d := range l {
		out = append(out, d.Error())
	}
	return strings.Join(out, "\n")
}
func Render(d Diagnostic, source string) string {
	lines := strings.Split(source, "\n")
	p := d.Span.Start
	if p.Line < 1 || p.Line > len(lines) {
		return d.Error()
	}
	line := strings.TrimSuffix(lines[p.Line-1], "\r")
	n := d.Span.End.Column - p.Column
	if d.Span.End.Line != p.Line || n < 1 {
		n = 1
	}
	if n > 80 {
		n = 80
	}
	return fmt.Sprintf("%s\n\n%d | %s\n%s%s\n", d.Error(), p.Line, line, strings.Repeat(" ", len(fmt.Sprint(p.Line))+3+p.Column-1), strings.Repeat("^", n))
}
