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

func TestCustomModNameAndFallback(t *testing.T) {
	tempDir := t.TempDir()
	modFile := filepath.Join(tempDir, "_mod.xru")
	modContent := `map {
    scripts -> "./scripts/*"
}`
	if err := os.WriteFile(modFile, []byte(modContent), 0644); err != nil {
		t.Fatalf("Failed to write _mod.xru: %v", err)
	}

	scriptsDir := filepath.Join(tempDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatalf("Failed to create scripts dir: %v", err)
	}

	// 1. File with explicit 'mod arch' and fileName 'archPrune.xru'
	archFile := filepath.Join(scriptsDir, "archPrune.xru")
	archContent := `mod arch
global func archPrune() {}
`
	if err := os.WriteFile(archFile, []byte(archContent), 0644); err != nil {
		t.Fatalf("Failed to write archPrune.xru: %v", err)
	}

	// 2. File with spaces in filename and NO 'mod' declaration
	utilFile := filepath.Join(scriptsDir, "my util.xru")
	utilContent := `global func helper() {}
`
	if err := os.WriteFile(utilFile, []byte(utilContent), 0644); err != nil {
		t.Fatalf("Failed to write my util.xru: %v", err)
	}

	// 3. File with invalid mod order (mod after import)
	badOrderFile := filepath.Join(scriptsDir, "badOrder.xru")
	badOrderContent := `use std
mod shouldFail
global func bad() {}
`
	if err := os.WriteFile(badOrderFile, []byte(badOrderContent), 0644); err != nil {
		t.Fatalf("Failed to write badOrder.xru: %v", err)
	}

	cfg, errs := modules.LoadModConfig(modFile)
	if len(errs) > 0 {
		t.Fatalf("Errors loading _mod.xru: %v", errs)
	}

	// Test 1: Should resolve as 'arch', but NOT as 'archPrune'
	modArch, err := cfg.ResolveModule("scripts", "arch")
	if err != nil {
		t.Fatalf("Expected 'arch' to resolve successfully, got: %v", err)
	}
	if modArch.Name != "arch" {
		t.Errorf("Expected module name 'arch', got '%s'", modArch.Name)
	}
	if !modArch.HasCustomName {
		t.Errorf("Expected HasCustomName to be true for 'arch'")
	}

	// Crucial: 'archPrune' must FAIL to resolve!
	_, errPrune := cfg.ResolveModule("scripts", "archPrune")
	if errPrune == nil {
		t.Errorf("Expected 'archPrune' to FAIL resolution because 'mod arch' renamed the module, but it succeeded!")
	}

	// Test 2: 'my util.xru' should resolve as 'myutil' (spaces stripped)
	modUtil, errUtil := cfg.ResolveModule("scripts", "myutil")
	if errUtil != nil {
		t.Fatalf("Expected 'myutil' to resolve for 'my util.xru', got: %v", errUtil)
	}
	if modUtil.Name != "myutil" {
		t.Errorf("Expected module name 'myutil', got '%s'", modUtil.Name)
	}

	// Test 3: 'badOrder.xru' had mod after use std -> mod shouldFail is ignored, defaults to 'badOrder'
	_, errShouldFail := cfg.ResolveModule("scripts", "shouldFail")
	if errShouldFail == nil {
		t.Errorf("Expected 'shouldFail' to NOT be registered because 'mod' was not at the top")
	}
	modBad, errBad := cfg.ResolveModule("scripts", "badOrder")
	if errBad != nil {
		t.Fatalf("Expected 'badOrder' to resolve as default name, got: %v", errBad)
	}
	if modBad.HasCustomName {
		t.Errorf("Expected HasCustomName to be false when mod order was invalid")
	}
}


