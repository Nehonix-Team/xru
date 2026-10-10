package dispatcher

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/checker"
	"github.com/Nehonix-Team/xru/pkg/lexer"
	"github.com/Nehonix-Team/xru/pkg/parser"
	"github.com/Nehonix-Team/xru/pkg/runtime"
)

type Dispatcher struct {
	rootDir   string
	evaluator *runtime.Evaluator
	parentEnv *runtime.Environment
}

func New(rootDir string, eval *runtime.Evaluator, env *runtime.Environment) *Dispatcher {
	return &Dispatcher{
		rootDir:   rootDir,
		evaluator: eval,
		parentEnv: env,
	}
}

// DispatchTree parcourt l'arborescence et exécute les descripteurs locaux correspondant au pattern
func (d *Dispatcher) DispatchTree(targetPath string, pattern string) error {
	if pattern == "" {
		pattern = ".rules.xru"
	}
	fullTarget := filepath.Join(d.rootDir, targetPath)
	if !filepath.IsAbs(targetPath) {
		fullTarget = filepath.Clean(fullTarget)
	}

	return filepath.WalkDir(fullTarget, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}

		// Détection d'un descripteur local correspondant au pattern
		name := entry.Name()
		matched, _ := filepath.Match(pattern, name)
		if matched || name == pattern {
			dir := filepath.Dir(path)
			if err := d.executeDescriptor(dir, path); err != nil {
				return err
			}
		}
		return nil
	})
}

// executeDescriptor exécute un descripteur dans un bac à sable hermétique (Sandbox)
func (d *Dispatcher) executeDescriptor(dir string, descriptorPath string) error {
	content, err := os.ReadFile(descriptorPath)
	if err != nil {
		return err
	}

	l := lexer.New(descriptorPath, string(content))
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return fmt.Errorf("descriptor syntax error in %s: %s", descriptorPath, p.Errors()[0])
	}

	c := checker.New()
	c.DefineUnit("Local")
	if d.parentEnv != nil {
		for varName := range d.parentEnv.All() {
			c.DefineVar(varName)
			c.DefineVar(strings.ToLower(varName))
			c.DefineVar(strings.ToUpper(varName))
		}
	}
	c.Check(prog)
	if len(c.Errors()) > 0 {
		return fmt.Errorf("descriptor validation error in %s: %s", descriptorPath, c.Errors()[0])
	}

	// Création du scope local hérité du scope parent
	localEnv := runtime.NewEnvironment(d.parentEnv)
	localEnv.SetCurrentDir(dir)
	if d.parentEnv != nil {
		for k, v := range d.parentEnv.All() {
			_ = localEnv.Set(strings.ToLower(k), v, false)
			_ = localEnv.Set(strings.ToUpper(k), v, false)
		}
	}

	// Injection des primitives locales confinées : Keep, Drop, UseVariant
	d.injectLocalPrimitives(dir, localEnv)

	// Évaluation
	if _, err := d.evaluator.Eval(prog, localEnv); err != nil {
		return fmt.Errorf("descriptor execution error in %s: %w", descriptorPath, err)
	}

	// Suppression du fichier descripteur lui-même après exécution pour laisser le projet propre
	_ = os.Remove(descriptorPath)
	return nil
}

func (d *Dispatcher) injectLocalPrimitives(dir string, env *runtime.Environment) {
	// 1. Primitive Keep(list) : Ne garde que la liste de fichiers spécifiée
	keepMethod := func(args []runtime.Value) (runtime.Value, error) {
		if len(args) == 0 {
			return runtime.NilValue{}, nil
		}
		keepList, ok := args[0].(runtime.ListValue)
		if !ok {
			return runtime.NilValue{}, fmt.Errorf("Keep requires a list of file names")
		}

		keepMap := make(map[string]bool)
		for _, el := range keepList.Elements {
			keepMap[el.String()] = true
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			return runtime.NilValue{}, err
		}

		for _, entry := range entries {
			name := entry.Name()
			if name == ".rules.xru" || name == "rules.xru" {
				continue
			}
			if !keepMap[name] {
				targetPath := filepath.Join(dir, name)
				if err := d.validateSandbox(dir, targetPath); err != nil {
					return runtime.NilValue{}, err
				}
				_ = os.RemoveAll(targetPath)
			}
		}
		return runtime.BoolValue{Val: true}, nil
	}

	// 2. Primitive Drop(list) : Supprime les fichiers ciblés (supporte wildcards)
	dropMethod := func(args []runtime.Value) (runtime.Value, error) {
		if len(args) == 0 {
			return runtime.NilValue{}, nil
		}
		dropList, ok := args[0].(runtime.ListValue)
		if !ok {
			return runtime.NilValue{}, fmt.Errorf("Drop requires a list of file names/patterns")
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			return runtime.NilValue{}, err
		}

		for _, item := range dropList.Elements {
			pattern := item.String()
			for _, entry := range entries {
				name := entry.Name()
				matched, _ := filepath.Match(pattern, name)
				if pattern == "*" || matched || pattern == name {
					targetPath := filepath.Join(dir, name)
					if err := d.validateSandbox(dir, targetPath); err != nil {
						return runtime.NilValue{}, err
					}
					_ = os.RemoveAll(targetPath)
				}
			}
		}
		return runtime.BoolValue{Val: true}, nil
	}

	// 3. Primitive UseVariant(target, variantSource)
	useVariantMethod := func(args []runtime.Value) (runtime.Value, error) {
		if len(args) < 2 {
			return runtime.NilValue{}, fmt.Errorf("UseVariant requires target and variant source")
		}
		target := args[0].String()
		variant := args[1].String()

		targetPath := filepath.Join(dir, target)
		variantPath := filepath.Join(dir, variant)

		if err := d.validateSandbox(dir, targetPath); err != nil {
			return runtime.NilValue{}, err
		}
		if err := d.validateSandbox(dir, variantPath); err != nil {
			return runtime.NilValue{}, err
		}

		if _, err := os.Stat(variantPath); os.IsNotExist(err) {
			return runtime.NilValue{}, fmt.Errorf("[ERR_TARGET_NOT_FOUND] Variant source '%s' does not exist in %s", variant, dir)
		}

		data, err := os.ReadFile(variantPath)
		if err != nil {
			return runtime.NilValue{}, err
		}
		if err := os.WriteFile(targetPath, data, 0644); err != nil {
			return runtime.NilValue{}, err
		}
		_ = os.Remove(variantPath)
		return runtime.BoolValue{Val: true}, nil
	}

	// Enregistrement dans l'environnement local sous forme de #UNIT "Scope" ou fonctions libres
	scopeUnit := &runtime.UnitInstance{
		UnitName:   "LocalScope",
		Properties: make(map[string]runtime.Value),
		Methods: map[string]runtime.NativeMethod{
			"Keep":       keepMethod,
			"Drop":       dropMethod,
			"UseVariant": useVariantMethod,
		},
		Directives: make(map[string]runtime.NativeDirective),
	}
	env.Set("Local", scopeUnit, true)
	// Raccourcis directs
	env.Set("Keep", runtime.FunctionValue{Name: "Keep", Fn: keepMethod}, true)
	env.Set("Drop", runtime.FunctionValue{Name: "Drop", Fn: dropMethod}, true)
	env.Set("UseVariant", runtime.FunctionValue{Name: "UseVariant", Fn: useVariantMethod}, true)
}

// validateSandbox garantit qu'aucune opération ne remonte au-delà du dossier local
func (d *Dispatcher) validateSandbox(allowedDir, targetPath string) error {
	rel, err := filepath.Rel(allowedDir, targetPath)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return fmt.Errorf("[ERR_SANDBOX_VIOLATION] Path '%s' escapes local sandbox '%s'", targetPath, allowedDir)
	}
	return nil
}
