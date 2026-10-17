package packager_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nehonix-Team/xru/pkg/checker"
	"github.com/Nehonix-Team/xru/pkg/lexer"
	"github.com/Nehonix-Team/xru/pkg/packager"
	"github.com/Nehonix-Team/xru/pkg/parser"
	"github.com/Nehonix-Team/xru/pkg/runtime"
)

func TestPackAndUnpack(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "xru_pack_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Création d'une arborescence source : tempDir/src/server.ts
	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(srcDir, "server.ts"), []byte("// hello xru"), 0644)
	_ = os.WriteFile(filepath.Join(srcDir, "ignore_me.tmp"), []byte("// ignored"), 0644)

	// Fichier .xruignore
	_ = os.WriteFile(filepath.Join(tempDir, ".xruignore"), []byte("*.tmp\n"), 0644)

	packPath := filepath.Join(tempDir, "template.xpack")
	pkg := packager.New(tempDir, ".xruignore")

	// 1. Pack
	if err := pkg.Pack(packPath, "src"); err != nil {
		t.Fatalf("Pack failed: %v", err)
	}

	// 2. Unpack dans un autre dossier
	destDir := filepath.Join(tempDir, "extracted")
	manifest, err := pkg.Unpack(packPath, destDir)
	if err != nil {
		t.Fatalf("Unpack failed: %v", err)
	}

	if manifest == nil || manifest.Version != "2.0.0" {
		t.Errorf("Invalid manifest unpacked: %v", manifest)
	}

	// Vérifier l'extraction de server.ts
	extractedServer := filepath.Join(destDir, "src", "server.ts")
	if _, err := os.Stat(extractedServer); os.IsNotExist(err) {
		t.Errorf("Expected %s to exist", extractedServer)
	}

	// Vérifier l'exclusion de ignore_me.tmp
	extractedIgnored := filepath.Join(destDir, "src", "ignore_me.tmp")
	if _, err := os.Stat(extractedIgnored); !os.IsNotExist(err) {
		t.Errorf("Expected %s to be ignored and absent", extractedIgnored)
	}
}

func TestPackBundleAndMinify(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "xru_bundle_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create root main.xru
	mainCode := `
arg mode: string = "default"
arg cdn: bool = true

global func prune(kind: string, mode: string, cdn: bool) {
    if mode == "minimal" || !cdn {
        file "item.txt", (doc, err) -> {
            doc.replace("HELLO", "WORLD")
        }
    }
}

dispatch "./src"
`
	_ = os.WriteFile(filepath.Join(tempDir, "main.xru"), []byte(mainCode), 0644)

	// Create src/controller/.rules.xru and src/controller/item.txt
	controllerDir := filepath.Join(tempDir, "src", "controller")
	_ = os.MkdirAll(controllerDir, 0755)
	controllerRule := `
from "root" use main::prune
prune("controller", mode, cdn)
`
	_ = os.WriteFile(filepath.Join(controllerDir, ".rules.xru"), []byte(controllerRule), 0644)
	_ = os.WriteFile(filepath.Join(controllerDir, "item.txt"), []byte("HELLO Controller"), 0644)

	// Pack
	packPath := filepath.Join(tempDir, "bundle.xpack")
	pkg := packager.New(tempDir, "")
	if err := pkg.Pack(packPath, "src"); err != nil {
		t.Fatalf("Pack failed: %v", err)
	}

	// Unpack
	destDir := filepath.Join(tempDir, "dest")
	_, err = pkg.Unpack(packPath, destDir)
	if err != nil {
		t.Fatalf("Unpack failed: %v", err)
	}

	// 1. Verify NO .rules.xru was extracted into destination tree
	extractedRule := filepath.Join(destDir, "src", "controller", ".rules.xru")
	if _, err := os.Stat(extractedRule); !os.IsNotExist(err) {
		t.Errorf("Expected .rules.xru to NOT exist in unpacked tree")
	}

	// 2. Verify bundled & minified main.xru exists at root of destination
	bundledMain := filepath.Join(destDir, "main.xru")
	contentBytes, err := os.ReadFile(bundledMain)
	if err != nil {
		t.Fatalf("Expected bundled main.xru at root of unpacked dir: %v", err)
	}
	content := string(contentBytes)
	t.Logf("Bundled & Minified content:\n%s", content)

	// 3. Verify minification: prune should be mangled to a short identifier (e.g. _a)
	// and in "src/controller" block should be present!
	if !strings.Contains(content, "in \"src/controller\"") {
		t.Errorf("Expected bundled main.xru to contain 'in \"src/controller\"', got:\n%s", content)
	}
	if strings.Contains(content, "global func prune") {
		t.Errorf("Expected function 'prune' to be mangled, got:\n%s", content)
	}

	// 4. Verify evaluation of bundled main.xru
	destItem := filepath.Join(destDir, "src", "controller", "item.txt")
	l := lexer.New(bundledMain, content)
	pr := parser.New(l)
	prog := pr.ParseProgram()
	if len(pr.Errors()) > 0 {
		t.Fatalf("Bundled code parse error: %v", pr.Errors())
	}

	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)
	env.SetCurrentDir(destDir)
	_ = env.Set("mode", runtime.StringValue{Val: "minimal"}, false)
	_ = env.Set("cdn", runtime.BoolValue{Val: true}, false)

	if _, err := ev.Eval(prog, env); err != nil {
		t.Fatalf("Eval error on bundled code: %v", err)
	}

	modData, err := os.ReadFile(destItem)
	if err != nil {
		t.Fatalf("Failed to read dest item: %v", err)
	}
	if !strings.Contains(string(modData), "WORLD Controller") {
		t.Errorf("Expected 'WORLD Controller', got: %s", string(modData))
	}
}

func TestBuildSingleFileModule(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "xru_singlefile_build_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// 1. Create _mod.xru
	modConf := `map {
    "rt" -> "./rt"
}`
	_ = os.WriteFile(filepath.Join(tempDir, "_mod.xru"), []byte(modConf), 0644)

	// 2. Create rt/cArg.xru
	_ = os.MkdirAll(filepath.Join(tempDir, "rt"), 0755)
	cArgContent := `use std
const dArgVal: string = "hello"
func run(x: number) -> string {
    return dArgVal + " " + x
}`
	_ = os.WriteFile(filepath.Join(tempDir, "rt", "cArg.xru"), []byte(cArgContent), 0644)

	// 3. Create main.xru that uses cArg as a qualified module
	mainContent := `use std
from "rt" use (cArg)

let res = cArg::run(42)
file "output.txt", (doc, err) -> {
    doc.replace("INITIAL", res)
}`
	_ = os.WriteFile(filepath.Join(tempDir, "main.xru"), []byte(mainContent), 0644)

	// 4. Create other.xru that uses direct member import
	otherContent := `use std
from "rt" use cArg::(dArgVal, run)

let res2 = run(99)
file "output2.txt", (doc, err) -> {
    doc.replace("INITIAL", res2)
}`
	_ = os.WriteFile(filepath.Join(tempDir, "other.xru"), []byte(otherContent), 0644)

	pkg := packager.New(tempDir, "")

	// Bundle main.xru
	bundledMain, err := pkg.BundleAndMinifyWithEntry("main.xru")
	if err != nil {
		t.Fatalf("BundleAndMinifyWithEntry(main.xru) failed: %v", err)
	}
	t.Logf("Bundled main:\n%s", bundledMain)

	// Ensure no 'from "rt" use' remains
	if strings.Contains(bundledMain, "from \"rt\"") {
		t.Errorf("Expected no 'from \"rt\"' in bundled output, got: %s", bundledMain)
	}

	// Execute bundledMain in an isolated directory WITHOUT _mod.xru or rt/
	isolatedDir, err := os.MkdirTemp("", "xru_isolated_eval_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(isolatedDir)

	_ = os.WriteFile(filepath.Join(isolatedDir, "output.txt"), []byte("INITIAL"), 0644)
	_ = os.WriteFile(filepath.Join(isolatedDir, "output2.txt"), []byte("INITIAL"), 0644)

	l := lexer.New("bundle.xru", bundledMain)
	pr := parser.New(l)
	prog := pr.ParseProgram()
	if len(pr.Errors()) > 0 {
		t.Fatalf("Parse error on bundledMain: %v", pr.Errors())
	}

	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)
	env.SetCurrentDir(isolatedDir)

	if _, err := ev.Eval(prog, env); err != nil {
		t.Fatalf("Eval error on bundled code: %v", err)
	}

	outBytes, err := os.ReadFile(filepath.Join(isolatedDir, "output.txt"))
	if err != nil {
		t.Fatalf("Failed to read output.txt: %v", err)
	}
	if string(outBytes) != "hello 42" {
		t.Errorf("Expected 'hello 42', got: %s", string(outBytes))
	}

	// Bundle other.xru
	bundledOther, err := pkg.BundleAndMinifyWithEntry("other.xru")
	if err != nil {
		t.Fatalf("BundleAndMinifyWithEntry(other.xru) failed: %v", err)
	}
	t.Logf("Bundled other:\n%s", bundledOther)

	l2 := lexer.New("bundle2.xru", bundledOther)
	pr2 := parser.New(l2)
	prog2 := pr2.ParseProgram()
	if len(pr2.Errors()) > 0 {
		t.Fatalf("Parse error on bundledOther: %v", pr2.Errors())
	}

	ev2 := runtime.NewEvaluator()
	env2 := runtime.NewEnvironment(nil)
	env2.SetCurrentDir(isolatedDir)

	if _, err := ev2.Eval(prog2, env2); err != nil {
		t.Fatalf("Eval error on bundled other code: %v", err)
	}

	out2Bytes, err := os.ReadFile(filepath.Join(isolatedDir, "output2.txt"))
	if err != nil {
		t.Fatalf("Failed to read output2.txt: %v", err)
	}
	if string(out2Bytes) != "hello 99" {
		t.Errorf("Expected 'hello 99', got: %s", string(out2Bytes))
	}
}



func TestBuildMapSyntaxAndPropertyAccess(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "xru_map_build_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	mainContent := `use std

interface ServerConfig @map {
    host string
    port number
    mode string
}

arg env: string = "prod"
arg port: number = 8080

const config: ServerConfig = @map {
    host: "127.0.0.1",
    port: port,
    mode: env
}

let p = config.port
let h = config.host
`
	_ = os.WriteFile(filepath.Join(tempDir, "main.xru"), []byte(mainContent), 0644)

	pkg := packager.New(tempDir, "")
	bundled, err := pkg.BundleAndMinifyWithEntry("main.xru")
	if err != nil {
		t.Fatalf("BundleAndMinifyWithEntry failed: %v", err)
	}
	t.Logf("Bundled output:\n%s", bundled)

	// 1. Must use @map{...} syntax, not JS-like raw {...}
	if !strings.Contains(bundled, "@map{") {
		t.Errorf("Expected bundled output to contain '@map{', got: %s", bundled)
	}

	// 2. Must convert config.port to .get("port")
	if !strings.Contains(bundled, `.get("port")`) {
		t.Errorf("Expected bundled output to convert map property access to .get(\"port\"), got: %s", bundled)
	}
	if !strings.Contains(bundled, `.get("host")`) {
		t.Errorf("Expected bundled output to convert map property access to .get(\"host\"), got: %s", bundled)
	}

	// 3. Mangling must use alphanumeric characters (a-z, 1-9) without leading underscore
	if strings.Contains(bundled, "_a") || strings.Contains(bundled, "_b") {
		t.Errorf("Expected mangled names to be alphanumeric without underscore, got: %s", bundled)
	}

	// 4. Bundled code must parse and check with 0 errors
	l := lexer.New("bundled.xru", bundled)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Bundled code has parse errors: %v", p.Errors())
	}
	c := checker.New()
	c.Check(prog)
	if len(c.Errors()) > 0 {
		t.Fatalf("Bundled code has checker errors: %v", c.Errors())
	}
}


func TestBuildAbortsOnValidationError(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "xru_build_err_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Code with semantic validation error: calling println with 0 arguments or assigning void
	invalidCode := `use std

const cfg = @map{
    name: "Seth",
}

const d = cfg.values().push(@map{test: "ok"})

std.println()
`
	_ = os.WriteFile(filepath.Join(tempDir, "main.xru"), []byte(invalidCode), 0644)

	pkg := packager.New(tempDir, "")
	_, err = pkg.BundleAndMinifyWithEntry("main.xru")
	if err == nil {
		t.Fatalf("Expected build to abort with validation error, but it succeeded")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "validation error in main.xru") {
		t.Errorf("Expected 'validation error in main.xru', got: %s", errStr)
	}
	if !strings.Contains(errStr, "ERR_TYPE_MISMATCH") && !strings.Contains(errStr, "ERR_ARG_COUNT") {
		t.Errorf("Expected specific checker error in message, got: %s", errStr)
	}
}
