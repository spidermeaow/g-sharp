package lexer

import (
	"g-sharp/compiler/diagnostics"
	"g-sharp/compiler/token"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Lexer struct {
	file, src      string
	off, line, col int
	tokens         []token.Token
	errors         diagnostics.List
}

func Scan(file, src string) ([]token.Token, diagnostics.List) {
	l := &Lexer{file: file, src: src, line: 1, col: 1}
	l.scan()
	return l.tokens, l.errors
}
func (l *Lexer) pos() token.Position {
	return token.Position{File: l.file, Offset: l.off, Line: l.line, Column: l.col}
}
func (l *Lexer) peek() rune {
	if l.off >= len(l.src) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.src[l.off:])
	return r
}
func (l *Lexer) next() rune {
	r, n := utf8.DecodeRuneInString(l.src[l.off:])
	l.off += n
	if r == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return r
}
func (l *Lexer) emit(k token.Kind, p token.Position) {
	l.tokens = append(l.tokens, token.Token{Kind: k, Text: l.src[p.Offset:l.off], Span: token.Span{Start: p, End: l.pos()}})
}
func (l *Lexer) err(p token.Position, msg string) {
	l.errors = append(l.errors, diagnostics.Diagnostic{Code: "GS1001", Message: msg, Span: token.Span{Start: p, End: l.pos()}})
}
func (l *Lexer) scan() {
	for l.off < len(l.src) {
		r := l.peek()
		if unicode.IsSpace(r) {
			l.next()
			continue
		}
		p := l.pos()
		if unicode.IsLetter(r) || r == '_' {
			l.next()
			for unicode.IsLetter(l.peek()) || unicode.IsDigit(l.peek()) || l.peek() == '_' {
				l.next()
			}
			k := token.Ident
			if token.Keywords[l.src[p.Offset:l.off]] {
				k = token.Kind(l.src[p.Offset:l.off])
			}
			l.emit(k, p)
			continue
		}
		if r >= '0' && r <= '9' {
			l.next()
			for l.peek() >= '0' && l.peek() <= '9' {
				l.next()
			}
			k := token.Integer
			if l.peek() == '.' && l.off+1 < len(l.src) && l.src[l.off+1] >= '0' && l.src[l.off+1] <= '9' {
				l.next()
				k = token.Float
				for l.peek() >= '0' && l.peek() <= '9' {
					l.next()
				}
			}
			l.emit(k, p)
			continue
		}
		if r == '"' || r == '\'' {
			quote := l.next()
			closed := false
			for l.off < len(l.src) {
				c := l.next()
				if c == quote {
					closed = true
					break
				}
				if c == '\n' {
					break
				}
				if c == '\\' && l.off < len(l.src) {
					l.next()
				}
			}
			if !closed {
				l.err(p, "unterminated literal")
			} else if _, err := strconv.Unquote(l.src[p.Offset:l.off]); err != nil {
				l.err(p, "invalid literal escape or character")
			}
			k := token.String
			if quote == '\'' {
				k = token.Char
			}
			l.emit(k, p)
			continue
		}
		if strings.HasPrefix(l.src[l.off:], "//") {
			for l.off < len(l.src) && l.peek() != '\n' {
				l.next()
			}
			l.emit(token.Comment, p)
			continue
		}
		if strings.HasPrefix(l.src[l.off:], "/*") {
			l.next()
			l.next()
			for l.off < len(l.src) && !strings.HasPrefix(l.src[l.off:], "*/") {
				l.next()
			}
			if l.off == len(l.src) {
				l.err(p, "unterminated block comment")
			} else {
				l.next()
				l.next()
			}
			l.emit(token.Comment, p)
			continue
		}
		found := false
		for _, op := range []string{"==", "!=", "<=", ">=", "&&", "||", "++", "--", "+=", "-=", "*=", "/=", "<-"} {
			if strings.HasPrefix(l.src[l.off:], op) {
				l.next()
				l.next()
				l.emit(token.Kind(op), p)
				found = true
				break
			}
		}
		if found {
			continue
		}
		l.next()
		if strings.ContainsRune("{}()[];,:.+-*/%!=<>?", r) {
			l.emit(token.Kind(string(r)), p)
		} else {
			l.err(p, "unexpected character "+strconv.QuoteRune(r))
		}
	}
	p := l.pos()
	l.tokens = append(l.tokens, token.Token{Kind: token.EOF, Span: token.Span{Start: p, End: p}})
}
