package checker

import (
	"g-sharp/compiler/ir"
	"go/constant"
)

func (c *Checker) capacity(e *ir.Expr) {
	v := constValue(e)
	if v != nil && v.Kind() == constant.Int && constant.Sign(v) < 0 {
		c.error(e.Span, "capacity must be nonnegative")
	}
}
func (c *Checker) valueName(name string) bool {
	if _, ok := c.scope.find(name); ok {
		return true
	}
	if name == "this" {
		return true
	}
	if c.owner != nil {
		for _, f := range c.owner.Fields {
			if f.SourceName == name {
				return true
			}
		}
	}
	return false
}
