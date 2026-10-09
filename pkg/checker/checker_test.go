package checker_test

import (
	"strings"
	"testing"

	"github.com/Nehonix-Team/xru/pkg/checker"
	"github.com/Nehonix-Team/xru/pkg/lexer"
	"github.com/Nehonix-Team/xru/pkg/parser"
)

func TestCheckerStrictRules(t *testing.T) {
	// 1. Test violation : &prop hors d'une unit
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

	// 2. Test violation : Directive orpheline dans with Net
	badCode2 := `
with Net.Fetch("https://example.com") {
    unknown_dir "arg" -> "val"
}
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

func TestTypeMismatch(t *testing.T) {
	code := `
let a: string = 3
`
	l := lexer.New("type_err.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	c := checker.New()
	c.Check(prog)

	if len(c.Errors()) == 0 {
		t.Fatal("Expected ERR_TYPE_MISMATCH, got 0 errors")
	}

	foundTypeErr := false
	for _, err := range c.Errors() {
		if strings.Contains(err, "ERR_TYPE_MISMATCH") {
			foundTypeErr = true
		}
	}
	if !foundTypeErr {
		t.Errorf("Expected ERR_TYPE_MISMATCH, got: %v", c.Errors())
	}
}
