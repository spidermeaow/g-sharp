// Package formatter provides a comment-preserving, deterministic token formatter.
package formatter

import (
	"g-sharp/compiler/diagnostics"
	"g-sharp/compiler/lexer"
	"g-sharp/compiler/parser"
	"g-sharp/compiler/token"
	"strings"
)

func Format(file, source string) (string, diagnostics.List) {
	ts, ds := lexer.Scan(file, source)
	if len(ds) > 0 {
		return "", ds
	}
	_, ds = parser.Parse(ts)
	if len(ds) > 0 {
		return "", ds
	}
	var out strings.Builder
	line := ""
	indent, paren := 0, 0
	prev := ""
	angles := genericAngles(ts)
	var cases []bool
	flush := func() {
		if strings.TrimSpace(line) != "" {
			out.WriteString(strings.Repeat("    ", max(indent, 0)))
			out.WriteString(strings.TrimSpace(line))
			out.WriteByte('\n')
			line = ""
		}
	}
	space := func() {
		if line != "" && !strings.HasSuffix(line, " ") {
			line += " "
		}
	}
	for i, t := range ts {
		if t.Kind == token.EOF {
			break
		}
		s := t.Text
		if (t.Kind == "case" || t.Kind == "default") && len(cases) > 0 && cases[len(cases)-1] {
			flush()
			indent--
			cases[len(cases)-1] = false
		}
		next := ""
		if i+1 < len(ts) {
			next = ts[i+1].Text
		}
		switch t.Kind {
		case token.Comment:
			if line != "" {
				space()
			}
			line += s
			flush()
		case "{":
			space()
			line += "{"
			flush()
			indent++
			cases = append(cases, false)
		case "}":
			flush()
			if len(cases) > 0 {
				if cases[len(cases)-1] {
					indent--
				}
				cases = cases[:len(cases)-1]
			}
			indent--
			line = "}"
			if next != ";" && next != "," && next != ")" && next != "else" {
				flush()
			}
		case "(":
			if prev == "if" || prev == "for" || prev == "foreach" || prev == "switch" {
				space()
			}
			line += "("
			paren++
		case ")":
			line = strings.TrimRight(line, " ") + ")"
			paren--
		case "[":
			line += "["
		case "]":
			line = strings.TrimRight(line, " ") + "]"
		case ".":
			line = strings.TrimRight(line, " ") + "."
		case ",":
			line = strings.TrimRight(line, " ") + ", "
		case ";":
			line = strings.TrimRight(line, " ") + ";"
			if paren > 0 {
				space()
			} else {
				flush()
			}
		case ":":
			line = strings.TrimRight(line, " ") + ":"
			flush()
			if len(cases) > 0 {
				indent++
				cases[len(cases)-1] = true
			}
		case "++", "--", "?":
			line = strings.TrimRight(line, " ") + s
		default:
			if angles[i] && (s == "<" || s == ">") {
				line = strings.TrimRight(line, " ") + s
				break
			}
			if line != "" && !strings.HasSuffix(line, "(") && !strings.HasSuffix(line, "[") && !strings.HasSuffix(line, ".") && !(i > 0 && angles[i-1] && prev == "<") {
				space()
			}
			line += s
		}
		prev = s
	}
	flush()
	return out.String(), nil
}

// Angle brackets are type delimiters only when their contents form a type list.
// All other comparisons retain ordinary operator spacing.
func genericAngles(ts []token.Token) map[int]bool {
	result := map[int]bool{}
	for i, t := range ts {
		if t.Kind != "<" || i == 0 || ts[i-1].Kind != token.Ident {
			continue
		}
		depth := 0
		var indices []int
		valid := true
		for j := i; j < len(ts); j++ {
			switch ts[j].Kind {
			case "<":
				depth++
				indices = append(indices, j)
			case ">":
				depth--
				indices = append(indices, j)
			case token.Ident, ",":
			default:
				valid = false
			}
			if !valid {
				break
			}
			if depth == 0 {
				for _, idx := range indices {
					result[idx] = true
				}
				break
			}
		}
	}
	return result
}
