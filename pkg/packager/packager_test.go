package packager_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Nehonix-Team/xru/pkg/packager"
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
