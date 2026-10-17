package lsp_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nehonix-Team/xru/pkg/lsp"
)

func TestLSPServerDiagnostics(t *testing.T) {
	testInput := new(bytes.Buffer)
	testOutput := new(bytes.Buffer)

	// 1. Initialize
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(initReq), initReq))

	// 2. didOpen with test.xru content having 3 errors
	content := `use std
use net as Net
use undefinedModulefs
std.println(an)
with Net.Fetch("https://example.com") {
    set nonExistExtractTo = "./src"
}
`
	didOpenPayload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri":        "file:///test.xru",
				"languageId": "xru",
				"version":    1,
				"text":       content,
			},
		},
	}
	openData, _ := json.Marshal(didOpenPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(openData), string(openData)))

	// 3. Hover
	hoverPayload := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "textDocument/hover",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri": "file:///test.xru",
			},
			"position": map[string]any{
				"line":      0,
				"character": 4, // on 'std'
			},
		},
	}
	hoverData, _ := json.Marshal(hoverPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(hoverData), string(hoverData)))

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()

	// Verify diagnostics
	if !strings.Contains(outStr, "publishDiagnostics") {
		t.Fatalf("Expected publishDiagnostics in output, got: %s", outStr)
	}

	if !strings.Contains(outStr, "undefinedModulefs") {
		t.Errorf("Expected diagnostic for undefinedModulefs, got: %s", outStr)
	}
	if !strings.Contains(outStr, "an") {
		t.Errorf("Expected diagnostic for undefined variable 'an', got: %s", outStr)
	}
	if !strings.Contains(outStr, "nonExistExtractTo") {
		t.Errorf("Expected diagnostic for nonExistExtractTo, got: %s", outStr)
	}

	// Verify hover response
	if !strings.Contains(outStr, "unit std") {
		t.Errorf("Expected hover info for 'std', got: %s", outStr)
	}
}

func TestLSPAutoImportCompletion(t *testing.T) {
	testInput := new(bytes.Buffer)
	testOutput := new(bytes.Buffer)

	tmpDir := setupServicesWorkspace(t)

	// 1. Initialize
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(initReq), initReq))

	// 2. didOpen for test2.xru
	content := `use std

`
	uri := "file://" + filepath.Join(tmpDir, "test_auto.xru")
	didOpenPayload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri":        uri,
				"languageId": "xru",
				"version":    1,
				"text":       content,
			},
		},
	}
	openData, _ := json.Marshal(didOpenPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(openData), string(openData)))

	// 3. Completion request
	compPayload := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri": uri,
			},
			"position": map[string]any{
				"line":      1,
				"character": 0,
			},
		},
	}
	compData, _ := json.Marshal(compPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(compData), string(compData)))

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()

	// Verify completion contains auto import for login
	if !strings.Contains(outStr, "auth::login") {
		t.Errorf("Expected auto import for login in completion, got: %s", outStr)
	}

	// Verify additionalTextEdits
	if !strings.Contains(outStr, "additionalTextEdits") {
		t.Errorf("Expected additionalTextEdits for auto import, got: %s", outStr)
	}
}

func setupServicesWorkspace(t *testing.T) string {
	tmp := t.TempDir()
	modContent := "map {\n    services -> \"./services/*\"\n}\n"
	if err := os.WriteFile(filepath.Join(tmp, "_mod.xru"), []byte(modContent), 0644); err != nil {
		t.Fatal(err)
	}
	servicesDir := filepath.Join(tmp, "services")
	if err := os.MkdirAll(servicesDir, 0755); err != nil {
		t.Fatal(err)
	}

	authContent := `mod auth
use std

global func login(user: string, password: string) {
    return true
}
`
	if err := os.WriteFile(filepath.Join(servicesDir, "auth.xru"), []byte(authContent), 0644); err != nil {
		t.Fatal(err)
	}

	cdnContent := `mod cdn
use std

global let cacheDir = "/var/cache"
global func deploy() {
}
`
	if err := os.WriteFile(filepath.Join(servicesDir, "cdn.xru"), []byte(cdnContent), 0644); err != nil {
		t.Fatal(err)
	}
	return tmp
}

func TestLSPFromCompletion(t *testing.T) {
	testInput := new(bytes.Buffer)
	testOutput := new(bytes.Buffer)

	tmpDir := setupServicesWorkspace(t)

	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(initReq), initReq))

	content := `from "
from "services" use 
from "services" use auth::
`
	uri := "file://" + filepath.Join(tmpDir, "test_from.xru")
	didOpenPayload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri":        uri,
				"languageId": "xru",
				"version":    1,
				"text":       content,
			},
		},
	}
	openData, _ := json.Marshal(didOpenPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(openData), string(openData)))

	// Request 1: completion on line 0 (after `from "`)
	comp1 := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 0, "character": 6},
		},
	}
	c1Data, _ := json.Marshal(comp1)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(c1Data), string(c1Data)))

	// Request 2: completion on line 1 (after `from "services" use `)
	comp2 := map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 1, "character": 20},
		},
	}
	c2Data, _ := json.Marshal(comp2)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(c2Data), string(c2Data)))

	// Request 3: completion on line 2 (after `from "services" use auth::`)
	comp3 := map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 2, "character": 26},
		},
	}
	c3Data, _ := json.Marshal(comp3)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(c3Data), string(c3Data)))

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()

	// Verify request 1 suggested services namespace
	if !strings.Contains(outStr, `"insertText":"services\" use "`) {
		t.Errorf("Expected namespace completion with 'services\" use ', got: %s", outStr)
	}

	// Verify request 2 suggested auth module
	if !strings.Contains(outStr, `"insertText":"auth"`) {
		t.Errorf("Expected module completion with 'auth', got: %s", outStr)
	}

	// Verify request 3 suggested login exported function
	if !strings.Contains(outStr, `"insertText":"login"`) {
		t.Errorf("Expected member completion with 'login', got: %s", outStr)
	}
}

func TestLSPDiagnosticsClearedOnCorrection(t *testing.T) {
	testInput := new(bytes.Buffer)
	testOutput := new(bytes.Buffer)

	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(initReq), initReq))

	uri := "file:///test_diag.xru"
	// 1. Send file with error
	didOpenPayload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri":        uri,
				"languageId": "xru",
				"version":    1,
				"text":       "use std\nstd.println(unknownErrVar)\n",
			},
		},
	}
	openData, _ := json.Marshal(didOpenPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(openData), string(openData)))

	// 2. Send didChange fixing the error
	didChangePayload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didChange",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri":     uri,
				"version": 2,
			},
			"contentChanges": []map[string]any{
				{"text": "use std\nstd.println(\"fixed\")\n"},
			},
		},
	}
	changeData, _ := json.Marshal(didChangePayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(changeData), string(changeData)))

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()

	// Should contain publishDiagnostics with empty array []
	if !strings.Contains(outStr, `"diagnostics":[]`) {
		t.Errorf("Expected diagnostics to be empty array '[]' after fix, got: %s", outStr)
	}
	if strings.Contains(outStr, `"diagnostics":null`) {
		t.Errorf("Diagnostics should never be null, got: %s", outStr)
	}
}

func TestLSPGoToDefinitionModule(t *testing.T) {
	testInput := new(bytes.Buffer)
	testOutput := new(bytes.Buffer)

	tmpDir := setupServicesWorkspace(t)

	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(initReq), initReq))

	uri := "file://" + filepath.Join(tmpDir, "test_nav.xru")
	content := `from "services" use auth
from "services" use "auth"
`
	didOpenPayload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri":        uri,
				"languageId": "xru",
				"version":    1,
				"text":       content,
			},
		},
	}
	openData, _ := json.Marshal(didOpenPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(openData), string(openData)))

	// Definition request on `auth` (line 0, col 21)
	defPayload := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "textDocument/definition",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 0, "character": 21},
		},
	}
	defData, _ := json.Marshal(defPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(defData), string(defData)))

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()

	// Verify definition points to auth.xru
	if !strings.Contains(outStr, "auth.xru") {
		t.Errorf("Expected definition of 'auth' to point to auth.xru, got: %s", outStr)
	}
}

func TestLSPModuleDotCompletionStrict(t *testing.T) {
	testInput := new(bytes.Buffer)
	testOutput := new(bytes.Buffer)

	tmpDir := setupServicesWorkspace(t)

	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(initReq), initReq))

	uri := "file://" + filepath.Join(tmpDir, "test_dot.xru")
	content := `from "services" use cdn
cdn.
`
	didOpenPayload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri":        uri,
				"languageId": "xru",
				"version":    1,
				"text":       content,
			},
		},
	}
	openData, _ := json.Marshal(didOpenPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(openData), string(openData)))

	// Completion right after `cdn.` (line 1, col 4)
	compPayload := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 1, "character": 4},
		},
	}
	compData, _ := json.Marshal(compPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(compData), string(compData)))

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()

	// Should contain cdn's exports: deploy and cacheDir
	if !strings.Contains(outStr, `"label":"deploy"`) {
		t.Errorf("Expected 'deploy' in cdn. completions, got: %s", outStr)
	}
	if !strings.Contains(outStr, `"label":"cacheDir"`) {
		t.Errorf("Expected 'cacheDir' in cdn. completions, got: %s", outStr)
	}

	// Must NOT contain global snippets, surgical methods or other modules
	if strings.Contains(outStr, `"label":"append"`) {
		t.Errorf("Completions for cdn. should NOT contain 'append'")
	}
	if strings.Contains(outStr, `"label":"auth"`) {
		t.Errorf("Completions for cdn. should NOT contain 'auth'")
	}
	if strings.Contains(outStr, `"label":"const"`) {
		t.Errorf("Completions for cdn. should NOT contain 'const'")
	}
}

func formatLSPMessage(method string, params any) string {
	payloadMap := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	}
	data, _ := json.Marshal(payloadMap)
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(data), string(data))
}

func TestLSPHoverDocComments(t *testing.T) {
	testCode := `
// Helper to clean targets
// Returns true on success
func clean(target: string) -> bool {
	return true
}

let plainVar = 123
`
	uri := "file:///test_hover.xru"
	didOpenReq := formatLSPMessage("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        uri,
			"languageId": "xru",
			"version":    1,
			"text":       testCode,
		},
	})

	hoverReqFunc := formatLSPMessage("textDocument/hover", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 3, "character": 6}, // on "clean"
	})

	hoverReqVar := formatLSPMessage("textDocument/hover", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 7, "character": 5}, // on "plainVar"
	})

	testInput := bytes.NewBufferString(didOpenReq + hoverReqFunc + hoverReqVar)
	testOutput := &bytes.Buffer{}

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()

	// 1. Should contain comments above clean
	if !strings.Contains(outStr, "Helper to clean targets") {
		t.Errorf("Expected doc comment 'Helper to clean targets' in hover, got: %s", outStr)
	}

	// 2. Should NOT contain 'Declared at line'
	if strings.Contains(outStr, "Declared at line") {
		t.Errorf("Hover output should NOT contain 'Declared at line', got: %s", outStr)
	}
}

func TestLSPUnusedVariableDiagnostic(t *testing.T) {
	uri := "file:///test_unused_diag.xru"
	testCode := `const r = 42
let a = 1
std.println(a)
`
	didOpenReq := formatLSPMessage("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        uri,
			"languageId": "xru",
			"version":    1,
			"text":       testCode,
		},
	})

	testInput := bytes.NewBufferString(didOpenReq)
	testOutput := &bytes.Buffer{}

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()

	// Should contain publishDiagnostics with unused variable message and tags: [1]
	if !strings.Contains(outStr, "Constant 'r' is declared but never used") {
		t.Errorf("Expected 'Constant \\'r\\' is declared but never used' in diagnostics, got: %s", outStr)
	}
	if !strings.Contains(outStr, `"tags":[1]`) {
		t.Errorf("Expected '\"tags\":[1]' in diagnostics, got: %s", outStr)
	}
}

func TestLocalSymbolsCompletion(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.xru")
	uri := "file://" + filePath

	testCode := `use std
use net as Net
from "services" use (auth, cdn)
from "services" use cdn::deploy

let a = 3
const isProd = true
let host = "https://api.xypriss.dev"

func clean(target: string) -> bool {
    const d = "package.json"
    
    return true
}
`

	didOpenReq := formatLSPMessage("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        uri,
			"languageId": "xru",
			"version":    1,
			"text":       testCode,
		},
	})

	// Complétion à l'intérieur de la fonction `clean` (ligne 11, col 4)
	compReq := formatLSPMessage("textDocument/completion", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 11, "character": 4},
	})

	testInput := bytes.NewBufferString(didOpenReq + compReq)
	testOutput := &bytes.Buffer{}

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()

	// Doit proposer les variables locales
	expectedSymbols := []string{"a", "isProd", "host", "clean", "target", "d", "Net", "auth", "cdn", "deploy"}
	for _, sym := range expectedSymbols {
		expectedLabel := fmt.Sprintf(`"label":"%s"`, sym)
		if !strings.Contains(outStr, expectedLabel) {
			t.Errorf("Expected local symbol '%s' in completions, got: %s", sym, outStr)
		}
	}

	// Doit avoir isIncomplete: false
	if !strings.Contains(outStr, `"isIncomplete":false`) {
		t.Errorf("Expected '\"isIncomplete\":false' in completions, got: %s", outStr)
	}
}

func TestLSPTaskAndUnitMemberCompletion(t *testing.T) {
	uri := "file:///test_task_comp.xru"
	testCode := `unit TemplateManager {
    prop baseDir: string = "./"
    func clean(target: string) -> bool {
        return true
    }
}

let mgr = @TemplateManager()
let downloadTask = with Net.fetch("https://example.com")
mgr.
downloadTask.
`
	didOpenReq := formatLSPMessage("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        uri,
			"languageId": "xru",
			"version":    1,
			"text":       testCode,
		},
	})

	// 1. Completion on mgr. (line 9, col 4)
	compMgrReq := formatLSPMessage("textDocument/completion", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 9, "character": 4},
	})

	// 2. Completion on downloadTask. (line 10, col 13)
	compTaskReq := formatLSPMessage("textDocument/completion", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 10, "character": 13},
	})

	testInput := bytes.NewBufferString(didOpenReq + compMgrReq + compTaskReq)
	testOutput := &bytes.Buffer{}

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()
	if !strings.Contains(outStr, `"label":"clean"`) {
		t.Errorf("Expected 'clean' method on mgr completion, got: %s", outStr)
	}
	if !strings.Contains(outStr, `"label":"wait"`) {
		t.Errorf("Expected 'wait' method on downloadTask completion, got: %s", outStr)
	}
	if !strings.Contains(outStr, `"label":"isDone"`) {
		t.Errorf("Expected 'isDone' method on downloadTask completion, got: %s", outStr)
	}
}

func TestInterpolationAndCallCompletion(t *testing.T) {
	uri := "file:///test_interp.xru"
	testCode := `use std
let result = "ok"
let downloadTask = with Net.fetch("https://example.com") catch err or "fallback"
for item, idx in 1..5 {
    std.println("Test: {")
}
std.println("Result: $f{re
std.println(re
`
	didOpenReq := formatLSPMessage("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        uri,
			"languageId": "xru",
			"version":    1,
			"text":       testCode,
		},
	})

	// 1. Completion inside interpolation "$f{re" on line 6, col 27
	compInterpReq := formatLSPMessage("textDocument/completion", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 6, "character": 27},
	})

	// 2. Completion inside call argument "std.println(re" on line 7, col 14
	compArgReq := formatLSPMessage("textDocument/completion", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 7, "character": 14},
	})

	testInput := bytes.NewBufferString(didOpenReq + compInterpReq + compArgReq)
	testOutput := &bytes.Buffer{}

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()

	// Both completions should suggest 'result' and 'err' and 'item'
	if !strings.Contains(outStr, `"label":"result"`) {
		t.Errorf("Expected 'result' in completions, got: %s", outStr)
	}
	if !strings.Contains(outStr, `"label":"err"`) {
		t.Errorf("Expected 'err' in completions, got: %s", outStr)
	}
	if !strings.Contains(outStr, `"label":"idx"`) {
		t.Errorf("Expected 'idx' in completions, got: %s", outStr)
	}
}

func TestFileStatementLSPCompletionAndHover(t *testing.T) {
	uri := "file:///test_file_lsp.xru"
	code := `use std

file "package.json", (content, error) -> {
    const a = 87
    content.
    error.
    std.println("Name: $f{content.")
}
`
	didOpenReq := formatLSPMessage("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        uri,
			"languageId": "xru",
			"version":    1,
			"text":       code,
		},
	})

	// 1. Completion on line 4 (inside file block): should suggest 'content', 'error', 'a'
	compLocalReq := formatLSPMessage("textDocument/completion", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 4, "character": 4},
	})

	// 2. Member completion on "content." on line 4, col 12
	compContentReq := formatLSPMessage("textDocument/completion", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 4, "character": 12},
	})

	// 3. Member completion on "content." inside string interpolation on line 6, col 35
	compInterpMemberReq := formatLSPMessage("textDocument/completion", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 6, "character": 35},
	})

	// 4. Hover on "content" on line 2, col 24
	hoverContentReq := formatLSPMessage("textDocument/hover", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 2, "character": 24},
	})

	testInput := bytes.NewBufferString(didOpenReq + compLocalReq + compContentReq + compInterpMemberReq + hoverContentReq)
	testOutput := &bytes.Buffer{}

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()

	if !strings.Contains(outStr, `"label":"content"`) {
		t.Errorf("Expected 'content' in local completions, got: %s", outStr)
	}
	if !strings.Contains(outStr, `"label":"error"`) {
		t.Errorf("Expected 'error' in local completions, got: %s", outStr)
	}
	if !strings.Contains(outStr, `"label":"getJson"`) {
		t.Errorf("Expected 'getJson' in content. completions, got: %s", outStr)
	}
	if !strings.Contains(outStr, `"label":"replaceFuzzy"`) {
		t.Errorf("Expected 'replaceFuzzy' in content. completions, got: %s", outStr)
	}
	if !strings.Contains(outStr, "Document") {
		t.Errorf("Expected 'Document' in hover, got: %s", outStr)
	}
}

func TestLSPUnusedFromUseImportDiagnostic(t *testing.T) {
	uri := "file:///home/idevo/Documents/projects/XTemplates/test_from_unused.xru"
	code := `from "dll" use XpackD::downloadRelease
from "dll" use gArgs::name
let myName = name
myName
`
	didOpenReq := formatLSPMessage("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        uri,
			"languageId": "xru",
			"version":    1,
			"text":       code,
		},
	})

	testInput := bytes.NewBufferString(didOpenReq)
	testOutput := &bytes.Buffer{}

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()
	if !strings.Contains(outStr, `"tags":[1]`) {
		t.Errorf("Expected DiagnosticTagUnnecessary (tags:[1]) in publishDiagnostics, got: %s", outStr)
	}
	if !strings.Contains(outStr, "downloadRelease") {
		t.Errorf("Expected diagnostic message mentioning 'downloadRelease', got: %s", outStr)
	}
}

func TestLSPGoToDefinitionPriority(t *testing.T) {
	uri := "file:///home/idevo/Documents/projects/XTemplates/test_def_priority.xru"
	code := `mod TDP

use std
from "dll" use gArgs::(author, version)

file "package.json", (doc, err) -> {
    let localAuthor = "local"
    doc.setJson("author", author)
    doc.setJson("local", localAuthor)
    if err != nil {
        unknownSymbol
    }
}
`
	didOpenReq := formatLSPMessage("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        uri,
			"languageId": "xru",
			"version":    1,
			"text":       code,
		},
	})

	formatReqWithID := func(id int, method string, params any) string {
		payloadMap := map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"method":  method,
			"params":  params,
		}
		data, _ := json.Marshal(payloadMap)
		return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(data), string(data))
	}

	// 1. Definition of imported member `author` on line 7 col 27 (doc.setJson("author", author))
	defAuthorReq := formatReqWithID(1, "textDocument/definition", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 7, "character": 27},
	})

	// 2. Definition of local variable `localAuthor` on line 8 col 27 (doc.setJson("local", localAuthor))
	defLocalReq := formatReqWithID(2, "textDocument/definition", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 8, "character": 27},
	})

	// 3. Definition of lambda parameter `err` on line 9 col 8 (if err != nil)
	defErrReq := formatReqWithID(3, "textDocument/definition", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 9, "character": 8},
	})

	// 4. Definition of unknown / unimported symbol `unknownSymbol` on line 10 col 9
	defUnknownReq := formatReqWithID(4, "textDocument/definition", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 10, "character": 9},
	})

	testInput := bytes.NewBufferString(didOpenReq + defAuthorReq + defLocalReq + defErrReq + defUnknownReq)
	testOutput := &bytes.Buffer{}

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()

	// Verify author points to args.xru
	if !strings.Contains(outStr, "args.xru") {
		t.Errorf("Expected author definition to point to args.xru, got: %s", outStr)
	}

	// Verify localAuthor points to test_def_priority.xru line 6
	if !strings.Contains(outStr, "test_def_priority.xru") {
		t.Errorf("Expected localAuthor definition to point to test_def_priority.xru, got: %s", outStr)
	}

	// Verify unknownSymbol returns null
	responses := strings.Split(outStr, "Content-Length:")
	var unknownResp string
	for _, r := range responses {
		if strings.Contains(r, `"id":4`) {
			unknownResp = r
			break
		}
	}
	if !strings.Contains(unknownResp, `"result":null`) {
		t.Errorf("Expected unknownSymbol definition result to be null, got: %s", unknownResp)
	}
}

func TestLSPTypeBasedCompletion(t *testing.T) {
	testInput := new(bytes.Buffer)
	testOutput := new(bytes.Buffer)

	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(initReq), initReq))

	code := `interface TServerConfig @map {
    host string
    port? number
    start func() -> void
}

let cfg: TServerConfig = @map{}
cfg.

let options: @map{ silent?: bool, retries: number } = @map{}
options.

let literal = { name: "xypriss", version: "2.0.0" }
literal.
`
	uri := "file:///test_types.xru"
	didOpenPayload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri":        uri,
				"languageId": "xru",
				"version":    1,
				"text":       code,
			},
		},
	}
	openData, _ := json.Marshal(didOpenPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(openData), string(openData)))

	// 1. Completion on `cfg.` (line 7, char 4)
	comp1 := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 7, "character": 4},
		},
	}
	c1Data, _ := json.Marshal(comp1)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(c1Data), string(c1Data)))

	// 2. Completion on `options.` (line 10, char 8)
	comp2 := map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 10, "character": 8},
		},
	}
	c2Data, _ := json.Marshal(comp2)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(c2Data), string(c2Data)))

	// 3. Completion on `literal.` (line 13, char 8)
	comp3 := map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 13, "character": 8},
		},
	}
	c3Data, _ := json.Marshal(comp3)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(c3Data), string(c3Data)))

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()

	// Verify interface fields in response 2
	if !strings.Contains(outStr, `"label":"host"`) || !strings.Contains(outStr, `"label":"port"`) || !strings.Contains(outStr, `"label":"start"`) {
		t.Errorf("Expected host, port, start for cfg., got: %s", outStr)
	}

	// Verify inline map fields in response 3
	if !strings.Contains(outStr, `"label":"silent"`) || !strings.Contains(outStr, `"label":"retries"`) {
		t.Errorf("Expected silent and retries for options., got: %s", outStr)
	}

	// Verify map literal fields in response 4
	if !strings.Contains(outStr, `"label":"name"`) || !strings.Contains(outStr, `"label":"version"`) {
		t.Errorf("Expected name and version for literal., got: %s", outStr)
	}
}

func TestLSPModuleImportedAsMapCompletion(t *testing.T) {
	testInput := new(bytes.Buffer)
	testOutput := new(bytes.Buffer)

	tmpDir := setupServicesWorkspace(t)

	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(initReq), initReq))

	uri := "file://" + filepath.Join(tmpDir, "test_as_map.xru")
	code := `from "services" use auth::@map as myAuth
myAuth.
`
	didOpenPayload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri":        uri,
				"languageId": "xru",
				"version":    1,
				"text":       code,
			},
		},
	}
	openData, _ := json.Marshal(didOpenPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(openData), string(openData)))

	// Completion right after `myAuth.` (line 1, col 7)
	compPayload := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 1, "character": 7},
		},
	}
	compData, _ := json.Marshal(compPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(compData), string(compData)))

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()

	if !strings.Contains(outStr, `"label":"login"`) {
		t.Errorf("Expected login completion for myAuth., got: %s", outStr)
	}
}

func TestInterpolationAndMemberCompletionAndDefinition(t *testing.T) {
	testInput := new(bytes.Buffer)
	testOutput := new(bytes.Buffer)

	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(initReq), initReq))

	uri := "file:///test_syntax.xru"
	code := `use std

type TExample = @map{
    a: string,
    b: number,
    c: func(id: string?) -> void or number,
}

func retryOperation(url: string, opts: @map{ silent?: "bool", retries: number }) -> void {
    std.println("Retrying $f{opts.} times...")
}

let validExample: TExample = @map{}
validExample.c()

func testIndexed() -> TExample[""] {
}
`
	didOpenPayload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri":        uri,
				"languageId": "xru",
				"version":    1,
				"text":       code,
			},
		},
	}
	openData, _ := json.Marshal(didOpenPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(openData), string(openData)))

	// 1. Completion inside $f{opts.} at line 9, col 34
	compPayload := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 9, "character": 34},
		},
	}
	compData, _ := json.Marshal(compPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(compData), string(compData)))

	// 2. Completion inside indexed interface TExample[" at line 15, col 31
	compIdxPayload := map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 15, "character": 31},
		},
	}
	compIdxData, _ := json.Marshal(compIdxPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(compIdxData), string(compIdxData)))

	// 3. Definition on validExample.c at line 13, col 13
	defPayload := map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "textDocument/definition",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 13, "character": 13},
		},
	}
	defData, _ := json.Marshal(defPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(defData), string(defData)))

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()

	// 1. Must offer "silent" and "retries"
	if !strings.Contains(outStr, `"label":"silent"`) || !strings.Contains(outStr, `"label":"retries"`) {
		t.Errorf("Expected opts properties 'silent' and 'retries' in interpolation completions, got: %s", outStr)
	}

	// 2. Must offer indexed properties "a", "b", "c"
	if !strings.Contains(outStr, `"label":"a"`) || !strings.Contains(outStr, `"label":"c"`) {
		t.Errorf("Expected indexed properties 'a' and 'c' in completions, got: %s", outStr)
	}

	// 3. Definition must point to line of `c:` in TExample (line 5, 0-indexed)
	if !strings.Contains(outStr, `"line":5`) {
		t.Errorf("Expected definition of validExample.c to point to line 5 in TExample, got: %s", outStr)
	}
}

func TestMapLiteralFieldCompletionWithDeduplication(t *testing.T) {
	testInput := new(bytes.Buffer)
	testOutput := new(bytes.Buffer)

	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(initReq), initReq))

	uri := "file:///test_map_dedup.xru"
	code := `type TSample = @map{
    a: string,
    b: number,
    c: bool,
}

let ex: TSample = @map{
    c: true,
    
}
`
	didOpenPayload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri":        uri,
				"languageId": "xru",
				"version":    1,
				"text":       code,
			},
		},
	}
	openData, _ := json.Marshal(didOpenPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(openData), string(openData)))

	// Completion on line 8, col 4 (inside @map{ after `c: true,`):
	// Should suggest `a` and `b`, but NOT `c`!
	compPayload := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 8, "character": 4},
		},
	}
	compData, _ := json.Marshal(compPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(compData), string(compData)))

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()

	// Must contain `a` and `b`
	if !strings.Contains(outStr, `"label":"a"`) || !strings.Contains(outStr, `"label":"b"`) {
		t.Errorf("Expected 'a' and 'b' in map literal completions, got: %s", outStr)
	}

	// Must NOT contain `c` because `c` is already declared in this map literal!
	if strings.Contains(outStr, `"label":"c"`) {
		t.Errorf("Map literal completions should NOT contain 'c' since it is already declared, got: %s", outStr)
	}
}

func TestPropertyHoverAndDefinitionAndTypeActions(t *testing.T) {
	testInput := new(bytes.Buffer)
	testOutput := new(bytes.Buffer)

	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(initReq), initReq))

	uri := "file:///test_hover_def.xru"
	code := `interface TExample {
    /**
     * Voilà un bon a bien propre
     */
    a string
    c func(id: string?) -> void or number // Signature de fonction
}

const data: TExample = {
    a: "hello"
}

const test = "Howdy"
let count = 42
let cur = date.now()

func name(params: TExample) -> void {
    std.println("Test: $f{params.a}")
}

data.a
test.
count.
cur.
`
	didOpenPayload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri":        uri,
				"languageId": "xru",
				"version":    1,
				"text":       code,
			},
		},
	}
	openData, _ := json.Marshal(didOpenPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(openData), string(openData)))

	// 1. Hover on property `a` inside map literal `a: "hello"` (line 9, col 4)
	hover1 := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "textDocument/hover",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 9, "character": 4},
		},
	}
	h1Data, _ := json.Marshal(hover1)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(h1Data), string(h1Data)))

	// 2. Definition on `a` inside map literal `a: "hello"` (line 9, col 4)
	def1 := map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "textDocument/definition",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 9, "character": 4},
		},
	}
	d1Data, _ := json.Marshal(def1)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(d1Data), string(d1Data)))

	// 3. Hover on `data.a` (line 20, col 6)
	hover2 := map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "textDocument/hover",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 20, "character": 6},
		},
	}
	h2Data, _ := json.Marshal(hover2)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(h2Data), string(h2Data)))

	// 4. Definition on `data.a` (line 20, col 6)
	def2 := map[string]any{
		"jsonrpc": "2.0",
		"id":      5,
		"method":  "textDocument/definition",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 20, "character": 6},
		},
	}
	d2Data, _ := json.Marshal(def2)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(d2Data), string(d2Data)))

	// 5. Completion on `test.` (line 21, col 5) -> string methods
	comp1 := map[string]any{
		"jsonrpc": "2.0",
		"id":      6,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 21, "character": 5},
		},
	}
	c1Data, _ := json.Marshal(comp1)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(c1Data), string(c1Data)))

	// 6. Completion on `cur.` (line 23, col 4) -> date fields (iso, timestamp, year, etc.)
	comp2 := map[string]any{
		"jsonrpc": "2.0",
		"id":      7,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 23, "character": 4},
		},
	}
	c2Data, _ := json.Marshal(comp2)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(c2Data), string(c2Data)))

	// 7. Definition on `$f{params.a}` (line 17, col 33)
	def3 := map[string]any{
		"jsonrpc": "2.0",
		"id":      8,
		"method":  "textDocument/definition",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 17, "character": 33},
		},
	}
	d3Data, _ := json.Marshal(def3)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(d3Data), string(d3Data)))

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()

	// 1. Hover on `a` in map literal must show (property) TExample.a: string and doc comment
	if !strings.Contains(outStr, `(property) TExample.a: string`) || !strings.Contains(outStr, `Voilà un bon a bien propre`) {
		t.Errorf("Expected TS-like property hover with doc on map literal field, got: %s", outStr)
	}

	// 2. Definition of `a` must point to line 4 (0-indexed line where `a: string` is declared)
	if !strings.Contains(outStr, `"id":3,"result":{"uri":"file:///test_hover_def.xru","range":{"start":{"line":4`) {
		t.Errorf("Expected definition of a in map literal to point to line 4, got: %s", outStr)
	}

	// 3. Definition of `data.a` must point to line 4
	if !strings.Contains(outStr, `"id":5,"result":{"uri":"file:///test_hover_def.xru","range":{"start":{"line":4`) {
		t.Errorf("Expected definition of data.a to point to line 4, got: %s", outStr)
	}

	// 4. Definition of `$f{params.a}` must point to line 4
	if !strings.Contains(outStr, `"id":8,"result":{"uri":"file:///test_hover_def.xru","range":{"start":{"line":4`) {
		t.Errorf("Expected definition of params.a in string interpolation to point to line 4, got: %s", outStr)
	}

	// 5. Completion on `test.` must contain `trim` and `len`
	if !strings.Contains(outStr, `"label":"trim"`) || !strings.Contains(outStr, `"label":"toupper"`) {
		t.Errorf("Expected string methods for test., got: %s", outStr)
	}

	// 6. Completion on `cur.` (date.now()) must contain `iso`, `timestamp`, `year`, `month`
	if !strings.Contains(outStr, `"label":"iso"`) || !strings.Contains(outStr, `"label":"timestamp"`) || !strings.Contains(outStr, `"label":"weekday"`) {
		t.Errorf("Expected date fields for cur., got: %s", outStr)
	}
}

func TestChainedAndInferredMemberCompletions(t *testing.T) {
	testInput := new(bytes.Buffer)
	testOutput := new(bytes.Buffer)

	// 1. Initialize
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(initReq), initReq))

	// 2. didOpen file with variables, trailing dot, calls, chained accesses
	content := `const test = "Howdy".
test.
date.now().
date.now().iso.
let d = date.now()
d.
let s = test.trim()
s.
func getMsg() -> string { return "hi" }
let m = getMsg()
m.
getMsg().
`
	didOpenPayload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri":        "file:///test_inference.xru",
				"languageId": "xru",
				"version":    1,
				"text":       content,
			},
		},
	}
	openData, _ := json.Marshal(didOpenPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(openData), string(openData)))

	// 3. Completion on `test.` (line 1, col 5)
	c1 := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": "file:///test_inference.xru"},
			"position":     map[string]any{"line": 1, "character": 5},
		},
	}
	c1Data, _ := json.Marshal(c1)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(c1Data), string(c1Data)))

	// 4. Completion on `date.now().` (line 2, col 11)
	c2 := map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": "file:///test_inference.xru"},
			"position":     map[string]any{"line": 2, "character": 11},
		},
	}
	c2Data, _ := json.Marshal(c2)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(c2Data), string(c2Data)))

	// 5. Completion on `date.now().iso.` (line 3, col 15)
	c3 := map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": "file:///test_inference.xru"},
			"position":     map[string]any{"line": 3, "character": 15},
		},
	}
	c3Data, _ := json.Marshal(c3)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(c3Data), string(c3Data)))

	// 6. Completion on `d.` (line 5, col 2)
	c4 := map[string]any{
		"jsonrpc": "2.0",
		"id":      5,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": "file:///test_inference.xru"},
			"position":     map[string]any{"line": 5, "character": 2},
		},
	}
	c4Data, _ := json.Marshal(c4)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(c4Data), string(c4Data)))

	// 7. Completion on `s.` (line 7, col 2)
	c5 := map[string]any{
		"jsonrpc": "2.0",
		"id":      6,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": "file:///test_inference.xru"},
			"position":     map[string]any{"line": 7, "character": 2},
		},
	}
	c5Data, _ := json.Marshal(c5)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(c5Data), string(c5Data)))

	// 8. Completion on `m.` (line 10, col 2)
	c6 := map[string]any{
		"jsonrpc": "2.0",
		"id":      7,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": "file:///test_inference.xru"},
			"position":     map[string]any{"line": 10, "character": 2},
		},
	}
	c6Data, _ := json.Marshal(c6)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(c6Data), string(c6Data)))

	// 9. Completion on `getMsg().` (line 11, col 9)
	c7 := map[string]any{
		"jsonrpc": "2.0",
		"id":      8,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": "file:///test_inference.xru"},
			"position":     map[string]any{"line": 11, "character": 9},
		},
	}
	c7Data, _ := json.Marshal(c7)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(c7Data), string(c7Data)))

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()

	// Parse responses by id
	getResp := func(id int) string {
		needle := fmt.Sprintf(`"id":%d,`, id)
		idx := strings.Index(outStr, needle)
		if idx == -1 {
			needle = fmt.Sprintf(`"id":%d}`, id)
			idx = strings.Index(outStr, needle)
		}
		if idx == -1 {
			return ""
		}
		start := strings.LastIndex(outStr[:idx], "{")
		if start == -1 {
			return ""
		}
		depth := 0
		for i := start; i < len(outStr); i++ {
			if outStr[i] == '{' {
				depth++
			} else if outStr[i] == '}' {
				depth--
				if depth == 0 {
					return outStr[start : i+1]
				}
			}
		}
		return ""
	}

	// 1. Response for `test.` (id: 2): must contain string methods, must NOT contain downloadTask
	r1 := getResp(2)
	if !strings.Contains(r1, `"label":"trim"`) || !strings.Contains(r1, `"label":"toupper"`) {
		t.Errorf("Expected string methods for test., got: %s", r1)
	}
	if strings.Contains(r1, `"label":"downloadTask"`) {
		t.Errorf("Did not expect downloadTask for test., got: %s", r1)
	}

	// 2. Response for `date.now().` (id: 3): must contain date fields and map methods, must NOT contain global vars
	r2 := getResp(3)
	if !strings.Contains(r2, `"label":"iso"`) || !strings.Contains(r2, `"label":"timestamp"`) || !strings.Contains(r2, `"label":"weekday"`) || !strings.Contains(r2, `"label":"year"`) {
		t.Errorf("Expected date fields for date.now()., got: %s", r2)
	}
	if strings.Contains(r2, `"label":"getMsg"`) || strings.Contains(r2, `"label":"downloadTask"`) {
		t.Errorf("Did not expect global symbols for date.now()., got: %s", r2)
	}

	// 3. Response for `date.now().iso.` (id: 4): must contain string methods
	r3 := getResp(4)
	if !strings.Contains(r3, `"label":"trim"`) || !strings.Contains(r3, `"label":"startswith"`) {
		t.Errorf("Expected string methods for date.now().iso., got: %s", r3)
	}

	// 4. Response for `d.` (id: 5): must contain date fields
	r4 := getResp(5)
	if !strings.Contains(r4, `"label":"iso"`) || !strings.Contains(r4, `"label":"month"`) {
		t.Errorf("Expected date fields for d., got: %s", r4)
	}

	// 5. Response for `s.` (id: 6): must contain string methods
	r5 := getResp(6)
	if !strings.Contains(r5, `"label":"trim"`) || !strings.Contains(r5, `"label":"len"`) {
		t.Errorf("Expected string methods for s., got: %s", r5)
	}

	// 6. Response for `m.` (id: 7): must contain string methods
	r6 := getResp(7)
	if !strings.Contains(r6, `"label":"trim"`) || !strings.Contains(r6, `"label":"slice"`) {
		t.Errorf("Expected string methods for m., got: %s", r6)
	}

	// 7. Response for `getMsg().` (id: 8): must contain string methods
	r7 := getResp(8)
	if !strings.Contains(r7, `"label":"trim"`) || !strings.Contains(r7, `"label":"toupper"`) {
		t.Errorf("Expected string methods for getMsg()., got: %s", r7)
	}
}

func TestCompletionInsideCallAndAssignment(t *testing.T) {
	testInput := new(bytes.Buffer)
	testOutput := new(bytes.Buffer)

	// 1. Initialize
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(initReq), initReq))

	// 2. didOpen file matching user's exact draft
	content := `use std
use date

const dObj = date.now()
const d1 = date.
std.println(dObj.)
std.println(date.)
`
	didOpenPayload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri":        "file:///test_call_assign.xru",
				"languageId": "xru",
				"version":    1,
				"text":       content,
			},
		},
	}
	openData, _ := json.Marshal(didOpenPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(openData), string(openData)))

	// 3. Completion on `const d1 = date.` (line 4, col 16)
	c1 := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": "file:///test_call_assign.xru"},
			"position":     map[string]any{"line": 4, "character": 16},
		},
	}
	c1Data, _ := json.Marshal(c1)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(c1Data), string(c1Data)))

	// 4. Completion on `std.println(dObj.)` (line 5, col 17)
	c2 := map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": "file:///test_call_assign.xru"},
			"position":     map[string]any{"line": 5, "character": 17},
		},
	}
	c2Data, _ := json.Marshal(c2)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(c2Data), string(c2Data)))

	// 5. Completion on `std.println(date.)` (line 6, col 17)
	c3 := map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": "file:///test_call_assign.xru"},
			"position":     map[string]any{"line": 6, "character": 17},
		},
	}
	c3Data, _ := json.Marshal(c3)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(c3Data), string(c3Data)))

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()

	getResp := func(id int) string {
		needle := fmt.Sprintf(`"id":%d,`, id)
		idx := strings.Index(outStr, needle)
		if idx == -1 {
			needle = fmt.Sprintf(`"id":%d}`, id)
			idx = strings.Index(outStr, needle)
		}
		if idx == -1 {
			return ""
		}
		start := strings.LastIndex(outStr[:idx], "{")
		if start == -1 {
			return ""
		}
		depth := 0
		for i := start; i < len(outStr); i++ {
			if outStr[i] == '{' {
				depth++
			} else if outStr[i] == '}' {
				depth--
				if depth == 0 {
					return outStr[start : i+1]
				}
			}
		}
		return ""
	}

	// 1. In assignment: `const d1 = date.` must contain date methods (`now`, `format`, `parse`, etc.)
	r1 := getResp(2)
	if !strings.Contains(r1, `"label":"now"`) || !strings.Contains(r1, `"label":"format"`) {
		t.Errorf("Expected date methods on 'const d1 = date.', got: %s", r1)
	}

	// 2. In call argument: `std.println(dObj.)` must contain date map fields (`iso`, `timestamp`, `weekday`, etc.)
	r2 := getResp(3)
	if !strings.Contains(r2, `"label":"iso"`) || !strings.Contains(r2, `"label":"timestamp"`) || !strings.Contains(r2, `"label":"weekday"`) {
		t.Errorf("Expected date map fields on 'std.println(dObj.)', got: %s", r2)
	}

	// 3. In call argument: `std.println(date.)` must contain date methods (`now`, `format`, `parse`, etc.)
	r3 := getResp(4)
	if !strings.Contains(r3, `"label":"now"`) || !strings.Contains(r3, `"label":"format"`) {
		t.Errorf("Expected date methods on 'std.println(date.)', got: %s", r3)
	}
}

func TestLSPStdNavigation(t *testing.T) {
	testInput := new(bytes.Buffer)
	testOutput := new(bytes.Buffer)

	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(initReq), initReq))

	code := `from path use { clean }
let name: string = "  world  "
let t = name.trim()
let info: DateInfo = {}
let p = path::clean("foo/bar")
`
	testDir := t.TempDir()
	testFilePath := filepath.Join(testDir, "main.xru")
	testURI := "file://" + filepath.ToSlash(testFilePath)

	openPayload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri":        testURI,
				"languageId": "xru",
				"version":    1,
				"text":       code,
			},
		},
	}
	openData, _ := json.Marshal(openPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(openData), string(openData)))

	// 1. Definition on 'clean' in 'from path use { clean }' (line 0, char 18)
	defReq1 := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "textDocument/definition",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": testURI},
			"position":     map[string]any{"line": 0, "character": 18},
		},
	}
	d1, _ := json.Marshal(defReq1)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(d1), string(d1)))

	// 2. Definition on 'string' in 'let name: string' (line 1, char 12)
	defReq2 := map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "textDocument/definition",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": testURI},
			"position":     map[string]any{"line": 1, "character": 12},
		},
	}
	d2, _ := json.Marshal(defReq2)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(d2), string(d2)))

	// 3. Definition on 'trim' in 'name.trim()' (line 2, char 14)
	defReq3 := map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "textDocument/definition",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": testURI},
			"position":     map[string]any{"line": 2, "character": 14},
		},
	}
	d3, _ := json.Marshal(defReq3)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(d3), string(d3)))

	// 4. Definition on 'DateInfo' in 'let info: DateInfo' (line 3, char 13)
	defReq4 := map[string]any{
		"jsonrpc": "2.0",
		"id":      5,
		"method":  "textDocument/definition",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": testURI},
			"position":     map[string]any{"line": 3, "character": 13},
		},
	}
	d4, _ := json.Marshal(defReq4)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(d4), string(d4)))

	server := lsp.NewServer(testInput, testOutput)
	server.Run()

	outStr := testOutput.String()

	getResp := func(id int) string {
		needle := fmt.Sprintf(`"id":%d,`, id)
		idx := strings.Index(outStr, needle)
		if idx == -1 {
			needle = fmt.Sprintf(`"id": %d,`, id)
			idx = strings.Index(outStr, needle)
		}
		if idx == -1 {
			return ""
		}
		for s := idx; s >= 0; s-- {
			if outStr[s] == '{' {
				end := strings.Index(outStr[s:], "\r\n")
				if end != -1 {
					return outStr[s : s+end]
				}
				return outStr[s:]
			}
		}
		return ""
	}

	rClean := getResp(2)
	if !strings.Contains(rClean, "path.xru") {
		t.Errorf("Expected definition of 'clean' to point to path.xru, got: %s", rClean)
	}

	rString := getResp(3)
	if !strings.Contains(rString, "builtin.xru") {
		t.Errorf("Expected definition of 'string' to point to builtin.xru, got: %s", rString)
	}

	rTrim := getResp(4)
	if !strings.Contains(rTrim, "builtin.xru") {
		t.Errorf("Expected definition of 'trim' to point to builtin.xru, got: %s", rTrim)
	}

	rDateInfo := getResp(5)
	if !strings.Contains(rDateInfo, "date.xru") {
		t.Errorf("Expected definition of 'DateInfo' to point to date.xru, got: %s", rDateInfo)
	}
}

func TestCompletionMultilineAndChaining(t *testing.T) {
	// 1. Multiline map literal completion after }.
	uri1 := "file:///test_multiline_map.xru"
	codeMultilineMap := "func test() {\n    let m = @map {\n        foo: \"bar\"\n    }.\n}\n"
	didOpen1 := formatLSPMessage("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": uri1, "languageId": "xru", "version": 1, "text": codeMultilineMap},
	})
	comp1 := formatLSPMessage("textDocument/completion", map[string]any{
		"textDocument": map[string]any{"uri": uri1},
		"position":     map[string]any{"line": 3, "character": 6},
	})
	testInput1 := bytes.NewBufferString(didOpen1 + comp1)
	testOutput1 := &bytes.Buffer{}
	srv1 := lsp.NewServer(testInput1, testOutput1)
	_ = srv1.Run()
	out1 := testOutput1.String()
	if !strings.Contains(out1, `"label":"clear"`) || !strings.Contains(out1, `"label":"foo"`) {
		t.Errorf("Expected multiline map literal completion to contain 'clear' and 'foo', got: %s", out1)
	}

	// 2. Chained method completion on string.split("/"). (returns list methods)
	uri2 := "file:///test_chaining.xru"
	codeChained := "func test(s: string) {\n    let parts = s.split(\"/\").\n}\n"
	didOpen2 := formatLSPMessage("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": uri2, "languageId": "xru", "version": 1, "text": codeChained},
	})
	comp2 := formatLSPMessage("textDocument/completion", map[string]any{
		"textDocument": map[string]any{"uri": uri2},
		"position":     map[string]any{"line": 1, "character": 29},
	})
	testInput2 := bytes.NewBufferString(didOpen2 + comp2)
	testOutput2 := &bytes.Buffer{}
	srv2 := lsp.NewServer(testInput2, testOutput2)
	_ = srv2.Run()
	out2 := testOutput2.String()
	if !strings.Contains(out2, `"label":"get"`) || !strings.Contains(out2, `"label":"join"`) {
		t.Errorf("Expected chained list completion to contain 'get' and 'join', got: %s", out2)
	}

	// 3. Double colon unit self completion on &date::
	uri3 := "file:///test_unit_scope.xru"
	codeSelfScope := "unit date {\n    func now() -> string { return \"\" }\n    func today() {\n        let x = &date::\n    }\n}\n"
	didOpen3 := formatLSPMessage("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": uri3, "languageId": "xru", "version": 1, "text": codeSelfScope},
	})
	comp3 := formatLSPMessage("textDocument/completion", map[string]any{
		"textDocument": map[string]any{"uri": uri3},
		"position":     map[string]any{"line": 3, "character": 23},
	})
	testInput3 := bytes.NewBufferString(didOpen3 + comp3)
	testOutput3 := &bytes.Buffer{}
	srv3 := lsp.NewServer(testInput3, testOutput3)
	_ = srv3.Run()
	out3 := testOutput3.String()
	if !strings.Contains(out3, `"label":"now"`) {
		t.Errorf("Expected &date:: completion to contain 'now', got: %s", out3)
	}
}

func TestDynamicStdDocsLoading(t *testing.T) {
	// Verify that documentation is dynamically resolved from std/*.xru definitions
	docToday := lsp.GetDocForWord("date.today", "today")
	if docToday == nil {
		t.Fatalf("Expected documentation for date.today, got nil")
	}
	if !strings.Contains(docToday.Doc, "today's date") {
		t.Errorf("Expected doc comment from std/date.xru, got: %s", docToday.Doc)
	}

	docPrintln := lsp.GetDocForWord("std.println", "println")
	if docPrintln == nil {
		t.Fatalf("Expected documentation for std.println, got nil")
	}
	if !strings.Contains(docPrintln.Doc, "standard output with rich ANSI") {
		t.Errorf("Expected doc comment from std/std.xru, got: %s", docPrintln.Doc)
	}

	docTrim := lsp.GetDocForWord("str.trim", "trim")
	if docTrim == nil {
		t.Fatalf("Expected documentation for string.trim, got nil")
	}
	if !strings.Contains(docTrim.Doc, "whitespace removed") {
		t.Errorf("Expected doc comment from std/builtin.xru, got: %s", docTrim.Doc)
	}

	docGetLine := lsp.GetDocForWord("doc.getLine", "getLine")
	if docGetLine == nil {
		t.Fatalf("Expected documentation for doc.getLine, got nil")
	}
	if !strings.Contains(docGetLine.Doc, "1-based index") {
		t.Errorf("Expected doc comment from std/builtin.xru, got: %s", docGetLine.Doc)
	}
}

func TestTypeAutocompletionInLSP(t *testing.T) {
	testInput := new(bytes.Buffer)
	testOutput := new(bytes.Buffer)

	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(initReq), initReq))

	uri := "file:///test_types.xru"
	code := `interface ROpt @map {
	onMismatch "fatal" or "warn" or "auto-update"
	mode "log" or "silent"
}

interface SysNative {
	func require(constraint: string, options: ) -> bool
}
`
	didOpenPayload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri":        uri,
				"languageId": "xru",
				"version":    1,
				"text":       code,
			},
		},
	}
	openData, _ := json.Marshal(didOpenPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(openData), string(openData)))

	// Request completion right after "options: " at line 6, character 42
	compPayload := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 6, "character": 42},
		},
	}
	compData, _ := json.Marshal(compPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(compData), string(compData)))

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()

	// Must propose ROpt (declared interface)
	if !strings.Contains(outStr, `"label":"ROpt"`) {
		t.Errorf("Expected 'ROpt' in type autocompletions, got: %s", outStr)
	}
	// Must propose string, bool, number
	if !strings.Contains(outStr, `"label":"string"`) || !strings.Contains(outStr, `"label":"bool"`) {
		t.Errorf("Expected 'string' and 'bool' in type autocompletions, got: %s", outStr)
	}
	// Must propose list[T]
	if !strings.Contains(outStr, `"label":"list[T]"`) {
		t.Errorf("Expected 'list[T]' in type autocompletions, got: %s", outStr)
	}
}

func TestMapLiteralKeyAndValueDX(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "draft.xru")
	content := `use std
use sys as Sys

Sys.require("<= 2.0.0", @map{
	mode: "log"
})
`
	_ = os.WriteFile(filePath, []byte(content), 0644)
	uri := "file://" + filePath

	testInput := new(bytes.Buffer)
	testOutput := new(bytes.Buffer)

	// 1. Initialize
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(initReq), initReq))

	// 2. didOpen
	didOpenPayload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri":        uri,
				"languageId": "xru",
				"version":    1,
				"text":       content,
			},
		},
	}
	openData, _ := json.Marshal(didOpenPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(openData), string(openData)))

	// 3. Hover on 'mode' at line 4, character 2
	hoverPayload := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "textDocument/hover",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 4, "character": 2},
		},
	}
	hoverData, _ := json.Marshal(hoverPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(hoverData), string(hoverData)))

	// 4. Definition on 'mode' at line 4, character 2
	defPayload := map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "textDocument/definition",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 4, "character": 2},
		},
	}
	defData, _ := json.Marshal(defPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(defData), string(defData)))

	// 5. Completion after 'mode: ' at line 4, character 7
	compValPayload := map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 4, "character": 7},
		},
	}
	compValData, _ := json.Marshal(compValPayload)
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(compValData), string(compValData)))

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()

	// Hover response must contain ROpt.mode and MUST NOT contain gArgs::mode
	if !strings.Contains(outStr, "ROpt.mode") {
		t.Errorf("Expected 'ROpt.mode' in hover, got output: %s", outStr)
	}
	if strings.Contains(outStr, "gArgs::mode") {
		t.Errorf("Map property key 'mode' erroneously resolved to external workspace symbol 'gArgs::mode'!")
	}

	// Definition response must jump to sys.xru
	if !strings.Contains(outStr, "sys.xru") {
		t.Errorf("Expected definition on 'mode' to jump to sys.xru, got: %s", outStr)
	}

	// Value completion response must propose "log" and "silent"
	if !strings.Contains(outStr, `"label":"\"log\""`) || !strings.Contains(outStr, `"label":"\"silent\""`) {
		t.Errorf("Expected value autocompletions '\"log\"' and '\"silent\"', got: %s", outStr)
	}
}

func TestUnionAssignmentCompletion(t *testing.T) {
	uri := "file:///test_union_comp.xru"
	code := `interface S {
    mode "dev" or "prod"
}

arg env: S["mode"] = "
`
	didOpenReq := formatLSPMessage("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        uri,
			"languageId": "xru",
			"version":    1,
			"text":       code,
		},
	})

	// Completion after '=' on line 4, col 23
	compReq := formatLSPMessage("textDocument/completion", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 4, "character": 23},
	})

	testInput := bytes.NewBufferString(didOpenReq + compReq)
	testOutput := &bytes.Buffer{}

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()
	if !strings.Contains(outStr, `"label":"\"dev\""`) {
		t.Errorf("Expected '\"dev\"' in union completion, got: %s", outStr)
	}
	if !strings.Contains(outStr, `"label":"\"prod\""`) {
		t.Errorf("Expected '\"prod\"' in union completion, got: %s", outStr)
	}
	if strings.Contains(outStr, `"label":"mode"`) {
		t.Errorf("Should NOT propose property 'mode' after '=' assignment, got: %s", outStr)
	}
}

func TestLocalInterfaceHoverDoesNotFallbackToBuiltin(t *testing.T) {
	uri := "file:///test_local_doc_hover.xru"
	code := `interface Document {
    test string
}

file "tests.html" (e, d: Document) -> {
    d.replace("a", "b")
}
`
	didOpenReq := formatLSPMessage("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        uri,
			"languageId": "xru",
			"version":    1,
			"text":       code,
		},
	})

	// Hover on "replace" in d.replace on line 5, col 6
	hoverReq := formatLSPMessage("textDocument/hover", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 5, "character": 6},
	})

	testInput := bytes.NewBufferString(didOpenReq + hoverReq)
	testOutput := &bytes.Buffer{}

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()
	// Must NOT contain "(method) Document.replace" or "Replaces all occurrences"
	if strings.Contains(outStr, "Replaces all occurrences") {
		t.Errorf("Expected NO builtin replace doc on local Document, but got: %s", outStr)
	}
}

func TestNamespaceModuleDefinitionAndTypeChecking(t *testing.T) {
	tempDir := t.TempDir()

	modFilePath := filepath.Join(tempDir, "_mod.xru")
	modContent := `map {
    "rt" -> "./"
}
`
	_ = os.WriteFile(modFilePath, []byte(modContent), 0644)

	cArgPath := filepath.Join(tempDir, "cArg.xru")
	cArgContent := `mod cArg

global const dArgVal: string = "default"
global func run(x: number) -> bool {
    return true
}
`
	_ = os.WriteFile(cArgPath, []byte(cArgContent), 0644)

	mainPath := filepath.Join(tempDir, "main.xru")
	mainContent := `use std
from "rt" use (cArg, mainb)
from "rt" use cArg::(dArgVal, run)

const a = cArg::dArgVal
const b: number = cArg::dArgVal
`
	_ = os.WriteFile(mainPath, []byte(mainContent), 0644)
	mainURI := "file://" + mainPath

	didOpenReq := formatLSPMessage("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        mainURI,
			"languageId": "xru",
			"version":    1,
			"text":       mainContent,
		},
	})

	formatReqWithID := func(id int, method string, params any) string {
		payloadMap := map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"method":  method,
			"params":  params,
		}
		data, _ := json.Marshal(payloadMap)
		return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(data), string(data))
	}

	// 1. Definition on "cArg" in 'from "rt" use (cArg, mainb)' (line 1, col 16)
	defModReq := formatReqWithID(1, "textDocument/definition", map[string]any{
		"textDocument": map[string]any{"uri": mainURI},
		"position":     map[string]any{"line": 1, "character": 16},
	})

	// 2. Definition on "dArgVal" in 'from "rt" use cArg::(dArgVal, run)' (line 2, col 23)
	defMemberReq := formatReqWithID(2, "textDocument/definition", map[string]any{
		"textDocument": map[string]any{"uri": mainURI},
		"position":     map[string]any{"line": 2, "character": 23},
	})

	// 3. Definition on "cArg" in 'const a = cArg::dArgVal' (line 4, col 11)
	defQualModReq := formatReqWithID(3, "textDocument/definition", map[string]any{
		"textDocument": map[string]any{"uri": mainURI},
		"position":     map[string]any{"line": 4, "character": 11},
	})

	// 4. Hover on "dArgVal" in 'const a = cArg::dArgVal' (line 4, col 18)
	hoverReq := formatReqWithID(4, "textDocument/hover", map[string]any{
		"textDocument": map[string]any{"uri": mainURI},
		"position":     map[string]any{"line": 4, "character": 18},
	})

	testInput := bytes.NewBufferString(didOpenReq + defModReq + defMemberReq + defQualModReq + hoverReq)
	testOutput := &bytes.Buffer{}

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()

	// Diagnostics: must flag line 5 (const b: number = cArg::dArgVal) because dArgVal is string!
	if !strings.Contains(outStr, "ERR_TYPE_MISMATCH") {
		t.Errorf("Expected ERR_TYPE_MISMATCH for const b: number = cArg::dArgVal, got: %s", outStr)
	}

	// ID 1: Definition on cArg must point to cArg.xru
	if !strings.Contains(outStr, `"id":1`) || !strings.Contains(outStr, "cArg.xru") {
		t.Errorf("Expected definition of cArg to point to cArg.xru, got: %s", outStr)
	}

	// ID 2: Definition on dArgVal must point to cArg.xru
	if !strings.Contains(outStr, `"id":2`) || !strings.Contains(outStr, "cArg.xru") {
		t.Errorf("Expected definition of dArgVal to point to cArg.xru, got: %s", outStr)
	}

	// ID 3: Definition on cArg in cArg::dArgVal must point to cArg.xru
	if !strings.Contains(outStr, `"id":3`) || !strings.Contains(outStr, "cArg.xru") {
		t.Errorf("Expected definition on module part of cArg::dArgVal to point to cArg.xru, got: %s", outStr)
	}

	// ID 4: Hover on dArgVal must show constant/property and NOT '(method) cArg::dArgVal(): string'
	if strings.Contains(outStr, "(method) cArg::dArgVal()") {
		t.Errorf("Expected hover on dArgVal to NOT be a method, got: %s", outStr)
	}
	if !strings.Contains(outStr, "cArg::dArgVal: string") {
		t.Errorf("Expected hover to show 'cArg::dArgVal: string', got: %s", outStr)
	}
}

func TestStringRepeatCompletionAndHover(t *testing.T) {
	uri := "file:///test_repeat.xru"
	code := `let str = "hello"
str.repeat(3)
`
	didOpenReq := formatLSPMessage("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        uri,
			"languageId": "xru",
			"version":    1,
			"text":       code,
		},
	})

	// Completion after 'str.' on line 1, col 4
	compReq := formatLSPMessage("textDocument/completion", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 1, "character": 4},
	})

	// Hover on "repeat" on line 1, col 6
	hoverReq := formatLSPMessage("textDocument/hover", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 1, "character": 6},
	})

	testInput := bytes.NewBufferString(didOpenReq + compReq + hoverReq)
	testOutput := &bytes.Buffer{}

	srv := lsp.NewServer(testInput, testOutput)
	_ = srv.Run()

	outStr := testOutput.String()

	if !strings.Contains(outStr, `"label":"repeat"`) {
		t.Errorf("Expected 'repeat' in string completions, got: %s", outStr)
	}
	if !strings.Contains(outStr, "repeat(count: number)") {
		t.Errorf("Expected repeat signature in hover/completion, got: %s", outStr)
	}
}

