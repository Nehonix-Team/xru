package modules_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nehonix-Team/xru/pkg/modules"
)

func TestModConfigValidation(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Valid _mod.xru
	validMod := filepath.Join(tmpDir, "_mod.xru")
	validContent := `
map {
    services -> "./src/services/*"
}
`
	if err := os.WriteFile(validMod, []byte(validContent), 0644); err != nil {
		t.Fatalf("Failed to write valid _mod.xru: %v", err)
	}

	cfg, errs := modules.LoadModConfig(validMod)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 errors on valid _mod.xru, got: %v", errs)
	}
	if cfg.Mappings["services"] != "./src/services/*" {
		t.Errorf("Expected mapping 'services' -> './src/services/*', got '%s'", cfg.Mappings["services"])
	}

	// 2. Invalid _mod.xru containing variables or scripts
	invalidMod := filepath.Join(tmpDir, "invalid_mod.xru")
	invalidContent := `
map {
    services -> "./src/services/*"
}
let a = 123
use std
func test() {}
`
	if err := os.WriteFile(invalidMod, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write invalid _mod.xru: %v", err)
	}

	_, errsInvalid := modules.LoadModConfig(invalidMod)
	if len(errsInvalid) == 0 {
		t.Fatalf("Expected errors on invalid _mod.xru containing executable code, got 0")
	}

	foundErr := false
	for _, e := range errsInvalid {
		if strings.Contains(e, "ERR_INVALID_MOD_DECLARATION") {
			foundErr = true
			break
		}
	}
	if !foundErr {
		t.Errorf("Expected ERR_INVALID_MOD_DECLARATION, got: %v", errsInvalid)
	}
}

func TestModuleAndMemberResolution(t *testing.T) {
	tmpDir := t.TempDir()
	servicesDir := filepath.Join(tmpDir, "src", "services")
	if err := os.MkdirAll(servicesDir, 0755); err != nil {
		t.Fatalf("Failed to create services dir: %v", err)
	}

	// Create auth.xru
	authFile := filepath.Join(servicesDir, "auth.xru")
	authContent := `
mod auth

global let b: string = "secret"
global const API_KEY = "xyz"
global func login(user: string) {
    return true
}

let internalKey = "private"
`
	if err := os.WriteFile(authFile, []byte(authContent), 0644); err != nil {
		t.Fatalf("Failed to write auth.xru: %v", err)
	}

	// Create _mod.xru
	modFile := filepath.Join(tmpDir, "_mod.xru")
	modContent := `
map {
    services -> "./src/services/*"
}
`
	if err := os.WriteFile(modFile, []byte(modContent), 0644); err != nil {
		t.Fatalf("Failed to write _mod.xru: %v", err)
	}

	cfg, errs := modules.LoadModConfig(modFile)
	if len(errs) > 0 {
		t.Fatalf("Errors loading _mod.xru: %v", errs)
	}

	// Resolve auth module
	modDetails, err := cfg.ResolveModule("services", "auth")
	if err != nil {
		t.Fatalf("Failed to resolve auth module: %v", err)
	}
	if modDetails.Name != "auth" {
		t.Errorf("Expected module name 'auth', got '%s'", modDetails.Name)
	}

	// Verify exported globals
	if _, ok := modDetails.Exports["b"]; !ok {
		t.Errorf("Expected exported member 'b'")
	}
	if _, ok := modDetails.Exports["API_KEY"]; !ok {
		t.Errorf("Expected exported member 'API_KEY'")
	}
	if _, ok := modDetails.Exports["login"]; !ok {
		t.Errorf("Expected exported member 'login'")
	}

	// Verify non-exported private var is absent
	if _, ok := modDetails.Exports["internalKey"]; ok {
		t.Errorf("Private variable 'internalKey' should NOT be exported")
	}

	// Verify candidate generation for auto-import
	candidates := cfg.GetAllExportedCandidates()
	if len(candidates) < 4 {
		t.Errorf("Expected at least 4 auto-import candidates, got %d", len(candidates))
	}

	foundLogin := false
	for _, c := range candidates {
		if c.Name == "login" && c.ImportStatement == "from \"services\" use auth::login" {
			foundLogin = true
			break
		}
	}
	if !foundLogin {
		t.Errorf("Expected auto-import candidate for 'login'")
	}
}

func TestExportedArgs(t *testing.T) {
	tempDir := t.TempDir()
	modFile := filepath.Join(tempDir, "_mod.xru")
	modContent := `map {
    root -> "./"
}`
	if err := os.WriteFile(modFile, []byte(modContent), 0644); err != nil {
		t.Fatalf("Failed to write _mod.xru: %v", err)
	}

	mainFile := filepath.Join(tempDir, "main.xru")
	mainContent := `mod main
global arg name: string = "app-name"
export arg port: number = 8080
arg internalFlag: bool = false
`
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main.xru: %v", err)
	}

	cfg, errs := modules.LoadModConfig(modFile)
	if len(errs) > 0 {
		t.Fatalf("Errors loading _mod.xru: %v", errs)
	}

	modDetails, err := cfg.ResolveModule("root", "main")
	if err != nil {
		t.Fatalf("Failed to resolve main module: %v", err)
	}

	if exp, ok := modDetails.Exports["name"]; !ok {
		t.Errorf("Expected 'name' to be exported")
	} else if exp.Kind != "arg" || exp.Type != "string" {
		t.Errorf("Expected 'name' to have kind 'arg' and type 'string', got: %v", exp)
	}

	if exp, ok := modDetails.Exports["port"]; !ok {
		t.Errorf("Expected 'port' to be exported")
	} else if exp.Kind != "arg" || exp.Type != "number" {
		t.Errorf("Expected 'port' to have kind 'arg' and type 'number', got: %v", exp)
	}

	if _, ok := modDetails.Exports["internalFlag"]; ok {
		t.Errorf("Non-global arg 'internalFlag' should not be exported")
	}
}

