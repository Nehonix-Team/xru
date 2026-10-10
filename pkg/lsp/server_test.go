package lsp_test

import (
	"bytes"
	"encoding/json"
	"fmt"
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



