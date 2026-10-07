package checker_test

import (
	"strings"
	"testing"

	"github.com/Nehonix-Team/xru/pkg/checker"
	"github.com/Nehonix-Team/xru/pkg/lexer"
	"github.com/Nehonix-Team/xru/pkg/parser"
)

func TestCheckerStrictRules(t *testing.T) {
	// 1. Test violation : &prop hors d'une #UNIT
	badCode1 := `
let x = &baseDir
`
	l1 := lexer.New("bad1.xru", badCode1)
	p1 := parser.New(l1)
	prog1 := p1.ParseProgram()

	c1 := checker.New()
	c1.Check(prog1)

	if len(c1.Errors()) == 0 {
		t.Fatal("Expected ERR_INVALID_PROP_ACCESS error, got 0 errors")
	}

	foundPropErr := false
	for _, err := range c1.Errors() {
		if strings.Contains(err, "ERR_INVALID_PROP_ACCESS") {
			foundPropErr = true
		}
	}
	if !foundPropErr {
		t.Errorf("Expected ERR_INVALID_PROP_ACCESS, got: %v", c1.Errors())
	}

	// 2. Test violation : Directive orpheline dans #WITH Net
	badCode2 := `
#WITH Net.Fetch("https://example.com")
    #UNKNOWN_DIR "arg" -> "val"
#END
`
	l2 := lexer.New("bad2.xru", badCode2)
	p2 := parser.New(l2)
	prog2 := p2.ParseProgram()

	c2 := checker.New()
	c2.Check(prog2)

	foundDirErr := false
	for _, err := range c2.Errors() {
		if strings.Contains(err, "ERR_ORPHAN_DIRECTIVE") {
			foundDirErr = true
		}
	}
	if !foundDirErr {
		t.Errorf("Expected ERR_ORPHAN_DIRECTIVE, got: %v", c2.Errors())
	}
}
