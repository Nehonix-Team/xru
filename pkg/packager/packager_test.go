package packager_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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

