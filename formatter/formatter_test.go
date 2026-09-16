package formatter

import (
	"strings"
	"testing"
)

func TestIdempotent(t *testing.T) {
	src := `// leading
class Program{static void Main(){var xs=new List<int>();/*keep*/for(var i=0;i<2;i++){xs.Add(i);}Console.WriteLine(xs.Count);}}`
	once, ds := Format("x", src)
	if len(ds) > 0 {
		t.Fatal(ds)
	}
	twice, ds := Format("x", once)
	if len(ds) > 0 || once != twice {
		t.Fatalf("%v\n%s\n%s", ds, once, twice)
	}
	if !strings.Contains(once, "/*keep*/") || !strings.Contains(once, "// leading") {
		t.Fatal(once)
	}
}
