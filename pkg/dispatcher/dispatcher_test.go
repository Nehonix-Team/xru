package dispatcher_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Nehonix-Team/xru/pkg/dispatcher"
	"github.com/Nehonix-Team/xru/pkg/runtime"
)

func TestLocalDescriptorSandboxAndKeep(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "xru_dispatcher_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Création d'une structure sous-dossier : tempDir/src/routes/
	routesDir := filepath.Join(tempDir, "src", "routes")
	if err := os.MkdirAll(routesDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Création de fichiers : api.ts, auth.ts, admin.ts
	_ = os.WriteFile(filepath.Join(routesDir, "api.ts"), []byte("// api"), 0644)
	_ = os.WriteFile(filepath.Join(routesDir, "auth.ts"), []byte("// auth"), 0644)
	_ = os.WriteFile(filepath.Join(routesDir, "admin.ts"), []byte("// admin"), 0644)

	// Création du descripteur local .rules.xru :
	// Ne garder que api.ts et supprimer le reste
	rulesCode := `
#IF {PLUGIN} == "false"
    Local.Keep(["api.ts"])
#END
`
	rulesPath := filepath.Join(routesDir, ".rules.xru")
	if err := os.WriteFile(rulesPath, []byte(rulesCode), 0644); err != nil {
		t.Fatal(err)
	}

	eval := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)
	env.Set("PLUGIN", runtime.StringValue{Val: "false"}, false)

	disp := dispatcher.New(tempDir, eval, env)
	if err := disp.DispatchTree("src", ".rules.xru"); err != nil {
		t.Fatalf("DispatchTree failed: %v", err)
	}

	// Vérification : api.ts doit exister
	if _, err := os.Stat(filepath.Join(routesDir, "api.ts")); os.IsNotExist(err) {
		t.Errorf("Expected api.ts to be kept")
	}

	// Vérification : auth.ts et admin.ts doivent avoir été supprimés
	if _, err := os.Stat(filepath.Join(routesDir, "auth.ts")); !os.IsNotExist(err) {
		t.Errorf("Expected auth.ts to be dropped")
	}
	if _, err := os.Stat(filepath.Join(routesDir, "admin.ts")); !os.IsNotExist(err) {
		t.Errorf("Expected admin.ts to be dropped")
	}

	// Le descripteur local .rules.xru doit avoir été nettoyé
	if _, err := os.Stat(rulesPath); !os.IsNotExist(err) {
		t.Errorf("Expected .rules.xru to be cleaned up after execution")
	}
}
