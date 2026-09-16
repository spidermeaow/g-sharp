package lexer

import "testing"

func TestScan(t *testing.T) {
	ts, ds := Scan("x.gsh", "// hello\nvar café = 42; /* ok */ café <- 1.5; \"a\\n\" 'λ'")
	if len(ds) > 0 {
		t.Fatal(ds)
	}
	want := []string{"comment", "var", "identifier", "=", "integer", ";", "comment", "identifier", "<-", "float", ";", "string", "char", "eof"}
	if len(ts) != len(want) {
		t.Fatalf("%#v", ts)
	}
	for i, w := range want {
		if string(ts[i].Kind) != w {
			t.Fatalf("%d: %s", i, ts[i].Kind)
		}
	}
	if ts[2].Span.Start.Line != 2 || ts[2].Span.Start.Column != 5 {
		t.Fatal(ts[2].Span)
	}
}
func TestBadLiterals(t *testing.T) {
	for _, s := range []string{"\"bad", "'ab'", "/*bad", "@"} {
		_, ds := Scan("x", s)
		if len(ds) == 0 {
			t.Fatal(s)
		}
	}
}
