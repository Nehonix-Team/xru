package formatter

import (
	"strings"
	"testing"
)

func TestFormatMinifiedCode(t *testing.T) {
	minified := `use std as s1;use fs as s2;unit e{declare private a: string = "secret-123";declare b: number = 4;func c(){let h = &a;&self::d()};func d(){&b=:&b+1}};let f = @e();f.c();let g = f.b;if s2.exists("test.txt"){s1.println("Size: $f{f.b}")};`

	formatted, err := Format(minified)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	t.Logf("Formatted output:\n%s", formatted)

	// Verify line count has expanded
	lines := strings.Split(strings.TrimSpace(formatted), "\n")
	if len(lines) < 10 {
		t.Errorf("Expected expanded multi-line output, got %d lines", len(lines))
	}

	// Verify indentation of unit members
	if !strings.Contains(formatted, "    declare private a: string = \"secret-123\";") {
		t.Errorf("Expected indented declare statement, got:\n%s", formatted)
	}
	if !strings.Contains(formatted, "    func c() {") {
		t.Errorf("Expected indented func c(), got:\n%s", formatted)
	}
	if !strings.Contains(formatted, "        let h = &a;") {
		t.Errorf("Expected double-indented body inside func c(), got:\n%s", formatted)
	}

	// Verify redundant semicolon after unit brace is not present
	if strings.Contains(formatted, "};\n") {
		t.Errorf("Expected no redundant semicolon after closing brace of unit, got:\n%s", formatted)
	}
}

func TestFormatMissingSemicolons(t *testing.T) {
	codeWithoutSemi := `use std
use fs as Fs

let x = 10
let y = 20

func test(a, b) {
    let sum = a + b
    return sum
}
`

	formatted, err := Format(codeWithoutSemi)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	t.Logf("Formatted output:\n%s", formatted)

	if !strings.Contains(formatted, "use std;") {
		t.Errorf("Expected 'use std;', got:\n%s", formatted)
	}
	if !strings.Contains(formatted, "use fs as Fs;") {
		t.Errorf("Expected 'use fs as Fs;', got:\n%s", formatted)
	}
	if !strings.Contains(formatted, "let x = 10;") {
		t.Errorf("Expected 'let x = 10;', got:\n%s", formatted)
	}
	if !strings.Contains(formatted, "let sum = a + b;") {
		t.Errorf("Expected 'let sum = a + b;', got:\n%s", formatted)
	}
	if !strings.Contains(formatted, "return sum;") {
		t.Errorf("Expected 'return sum;', got:\n%s", formatted)
	}
}

func TestFormatPreservesRawStringAndComments(t *testing.T) {
	code := `// Main entry
use std;

/*
 * Multi-line config block
 */
let doc = <<<
line 1
    indented line; without code
line 3
>>>;

std.println("OK"); // End of script
`

	formatted, err := Format(code)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	t.Logf("Formatted output:\n%s", formatted)

	// Comments preserved
	if !strings.Contains(formatted, "// Main entry") {
		t.Errorf("Expected line comment preserved, got:\n%s", formatted)
	}
	if !strings.Contains(formatted, "Multi-line config block") {
		t.Errorf("Expected block comment preserved, got:\n%s", formatted)
	}
	if !strings.Contains(formatted, "// End of script") {
		t.Errorf("Expected trailing comment preserved, got:\n%s", formatted)
	}

	// Raw heredoc string preserved exactly
	if !strings.Contains(formatted, "<<<\nline 1\n    indented line; without code\nline 3\n>>>") {
		t.Errorf("Expected raw chevron string preserved verbatim, got:\n%s", formatted)
	}
}

func TestStandaloneSnippet(t *testing.T) {
	code := `const o = if i -> "<green>Dependencies installation enabled</>" else "<yellow>Dependencies installation disabled</>";
interface EnvObj @map{XYPRISS_SHIELD "silent";PORT number;SERVER_TIMEOUT number;XEMS_SECRET string};
const q = @map{PORT:j,SERVER_TIMEOUT:k,XEMS_SECRET:p,XYPRISS_SHIELD:"silent"};
if s2.exists(".env"){file ".env",(ae,af)->{if af!=nil{return};if e=="minimal"{let ag = "";for ah,ai in q{if ai.type()=="number"{ag=:ag+"$f{ah}=$f{ai}\n"} else{ag=:ag+"$f{ah}=\"$f{ai}\"\n"}};ae.setContent(ag)} else{ae.removeLines("XEMS_SECRET");ae.append("\n");ae.append("XEMS_SECRET=$f{p}")}}};`

	formatted, err := Format(code)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}
	t.Logf("Formatted output:\n%s", formatted)
}
