package lowering

import (
	"g-sharp/compiler/ir"
	"testing"
)

func TestContinueTargets(t *testing.T) {
	inner := &ir.Stmt{Op: "foreach", Body: []*ir.Stmt{{Op: "continue"}}}
	f := &ir.Function{Body: []*ir.Stmt{{Op: "for", Body: []*ir.Stmt{{Op: "continue"}, inner}}}}
	p := &ir.Program{Functions: []*ir.Function{f}, Entry: f}
	out := Lower(p)
	loop := out.Entry.Body[0].Body[0]
	body := loop.Body[0].Body
	if body[0].Op != "jump" || body[1].Body[0].Op != "continue" {
		t.Fatal(body)
	}
	if f.Body[0].Op != "for" || f.Body[0].Body[0].Op != "continue" {
		t.Fatal("mutated checked IR")
	}
}
