package lsp_test

import (
	"bytes"
	"encoding/json"
	"fmt"
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

	// 1. Initialize
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(initReq), initReq))

	// 2. didOpen for test2.xru
	content := `use std

`
	didOpenPayload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri":        "file:///home/idevo/Documents/projects/XTemplates/test_auto.xru",
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
				"uri": "file:///home/idevo/Documents/projects/XTemplates/test_auto.xru",
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

func TestLSPFromCompletion(t *testing.T) {
	testInput := new(bytes.Buffer)
	testOutput := new(bytes.Buffer)

	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(initReq), initReq))

	content := `from "
from "services" use 
from "services" use auth::
`
	uri := "file:///home/idevo/Documents/projects/XTemplates/test_from.xru"
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

	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(initReq), initReq))

	uri := "file:///home/idevo/Documents/projects/XTemplates/test_nav.xru"
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

	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	testInput.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(initReq), initReq))

	uri := "file:///home/idevo/Documents/projects/XTemplates/test_dot.xru"
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

