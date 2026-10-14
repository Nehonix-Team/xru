package modules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/lexer"
	"github.com/Nehonix-Team/xru/pkg/parser"
)

const ModConfigFileName = "_mod.xru"

// IsModConfigFile vérifie si un nom de fichier est le fichier de configuration natif '_mod.xru'.
func IsModConfigFile(path string) bool {
	return filepath.Base(path) == ModConfigFileName
}

// ExportedMember représente un symbole global exporté par un module local.
type ExportedMember struct {
	Name     string
	Kind     string // "var", "const", "func"
	Type     string
	Params   []*ast.Parameter
	Position lexer.Position
}

// ModuleDetails contient les informations d'un module local analysé.
type ModuleDetails struct {
	Name      string
	Namespace string
	FilePath  string
	Exports   map[string]*ExportedMember
}

// ModConfig représente le mapping déclaré dans '_mod.xru'.
type ModConfig struct {
	FilePath string
	Dir      string
	Mappings map[string]string // ex: "services" -> "./src/services/*"
}

// FindModConfigFile recherche le fichier '_mod.xru' en remontant l'arborescence à partir de startDir.
func FindModConfigFile(startDir string) string {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return ""
	}

	for {
		candidate := filepath.Join(dir, ModConfigFileName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}

		parent := filepath.Dir(dir)
		if parent == dir || parent == "" {
			break
		}
		dir = parent
	}
	return ""
}

// LoadModConfig parse et valide '_mod.xru'. Il vérifie rigoureusement que SEULES les déclarations
// 'map { ... }' y sont présentes.
func LoadModConfig(filePath string) (*ModConfig, []string) {
	var errs []string

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, []string{fmt.Sprintf("cannot read '%s': %v", filePath, err)}
	}

	l := lexer.New(filePath, string(content))
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return nil, p.Errors()
	}

	dir := filepath.Dir(filePath)
	modCfg := &ModConfig{
		FilePath: filePath,
		Dir:      dir,
		Mappings: make(map[string]string),
	}

	for _, stmt := range prog.Statements {
		mapStmt, ok := stmt.(*ast.MapStatement)
		if !ok {
			pos := stmt.Pos()
			errs = append(errs, fmt.Sprintf("%s:%d:%d: [ERR_INVALID_MOD_DECLARATION] '%s' is strictly reserved for module mapping definitions ('map { ... }'). Executable code, statements, variables, and imports are not allowed.",
				filePath, pos.Line, pos.Column, ModConfigFileName))
			continue
		}

		for k, v := range mapStmt.Mappings {
			modCfg.Mappings[k] = v
		}
	}

	return modCfg, errs
}

// GetNamespaceModules analyse et extrait les modules .xru du dossier mappé par ce namespace.
func (cfg *ModConfig) GetNamespaceModules(namespace string) (map[string]*ModuleDetails, error) {
	pattern, ok := cfg.Mappings[namespace]
	if !ok {
		return nil, fmt.Errorf("module namespace '%s' is not mapped in %s", namespace, ModConfigFileName)
	}

	// Nettoyage du pattern : ex: "./src/services/*" -> "./src/services"
	cleanPath := strings.TrimSuffix(pattern, "/*")
	cleanPath = strings.TrimSuffix(cleanPath, "*")
	cleanPath = strings.TrimSuffix(cleanPath, "/")

	var targetDir string
	if filepath.IsAbs(cleanPath) {
		targetDir = cleanPath
	} else {
		targetDir = filepath.Join(cfg.Dir, cleanPath)
	}

	info, err := os.Stat(targetDir)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("mapped path '%s' (for namespace '%s') does not exist or is not a directory", cleanPath, namespace)
	}

	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read mapped directory '%s': %v", targetDir, err)
	}

	modules := make(map[string]*ModuleDetails)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".xru") {
			continue
		}
		// Ignorer le fichier _mod.xru lui-même s'il est dans ce dossier
		if entry.Name() == ModConfigFileName {
			continue
		}

		fullPath := filepath.Join(targetDir, entry.Name())
		modDetails, err := parseModuleFile(fullPath, namespace)
		if err != nil {
			continue
		}
		modules[modDetails.Name] = modDetails
		fileBase := strings.TrimSuffix(entry.Name(), ".xru")
		if fileBase != modDetails.Name {
			modules[fileBase] = modDetails
		}
	}

	return modules, nil
}

// ResolveModule résout un module spécifique dans un namespace mappé.
func (cfg *ModConfig) ResolveModule(namespace, modName string) (*ModuleDetails, error) {
	mods, err := cfg.GetNamespaceModules(namespace)
	if err != nil {
		return nil, err
	}

	modDetails, ok := mods[modName]
	if !ok {
		return nil, fmt.Errorf("module '%s' not found in namespace '%s'", modName, namespace)
	}
	return modDetails, nil
}

// parseModuleFile lit et parse un fichier .xru pour extraire son identité de module (mod <name>)
// et l'ensemble de ses symboles 'global' exportés.
func parseModuleFile(filePath, namespace string) (*ModuleDetails, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	l := lexer.New(filePath, string(content))
	p := parser.New(l)
	prog := p.ParseProgram()
	if prog == nil {
		return nil, fmt.Errorf("failed to parse %s", filePath)
	}

	// Nom par défaut : le nom du fichier sans extension
	baseName := strings.TrimSuffix(filepath.Base(filePath), ".xru")
	modName := baseName

	exports := make(map[string]*ExportedMember)

	for _, stmt := range prog.Statements {
		if stmt == nil {
			continue
		}
		switch s := stmt.(type) {
		case *ast.ModStatement:
			if s != nil && s.Name != "" {
				modName = s.Name
			}
		case *ast.VarStatement:
			if s != nil && s.IsGlobal {
				kind := "var"
				if s.IsConst {
					kind = "const"
				}
				memberType := s.Type
				if memberType == "" && s.Value != nil {
					memberType = inferBasicType(s.Value)
				}
				exports[s.Name] = &ExportedMember{
					Name:     s.Name,
					Kind:     kind,
					Type:     memberType,
					Position: s.Pos(),
				}
			}
		case *ast.FuncStatement:
			if s != nil && s.IsGlobal {
				exports[s.Name] = &ExportedMember{
					Name:     s.Name,
					Kind:     "func",
					Type:     s.ReturnType,
					Params:   s.Parameters,
					Position: s.Pos(),
				}
			}
		case *ast.ArgStatement:
			if s != nil && s.IsGlobal {
				memberType := s.Type
				if memberType == "" && s.DefaultValue != nil {
					memberType = inferBasicType(s.DefaultValue)
				}
				exports[s.Name] = &ExportedMember{
					Name:     s.Name,
					Kind:     "arg",
					Type:     memberType,
					Position: s.Pos(),
				}
			}
		}
	}

	return &ModuleDetails{
		Name:      modName,
		Namespace: namespace,
		FilePath:  filePath,
		Exports:   exports,
	}, nil
}

// AutoImportCandidate représente un symbole importable avec sa commande d'import.
type AutoImportCandidate struct {
	Name            string // ex: "login" ou "auth"
	Namespace       string // ex: "services"
	ModuleName      string // ex: "auth"
	Kind            string // "func", "var", "const", "module"
	Type            string // ex: "void" ou "string"
	ImportStatement string // ex: 'from "services" use auth::login'
	FilePath        string // Chemin absolu du fichier source
}

// GetAllExportedCandidates liste tous les candidats d'auto-importation de tous les modules mappés dans _mod.xru.
func (cfg *ModConfig) GetAllExportedCandidates() []AutoImportCandidate {
	var candidates []AutoImportCandidate

	for ns := range cfg.Mappings {
		mods, err := cfg.GetNamespaceModules(ns)
		if err != nil {
			continue
		}

		for _, m := range mods {
			// Candidat 1 : Le module entier (from "services" use auth)
			candidates = append(candidates, AutoImportCandidate{
				Name:            m.Name,
				Namespace:       ns,
				ModuleName:      m.Name,
				Kind:            "module",
				Type:            "module",
				ImportStatement: fmt.Sprintf("from \"%s\" use %s", ns, m.Name),
				FilePath:        m.FilePath,
			})

			// Candidat 2 : Tous ses membres exportés (from "services" use auth::login)
			for _, exp := range m.Exports {
				candidates = append(candidates, AutoImportCandidate{
					Name:            exp.Name,
					Namespace:       ns,
					ModuleName:      m.Name,
					Kind:            exp.Kind,
					Type:            exp.Type,
					ImportStatement: fmt.Sprintf("from \"%s\" use %s::%s", ns, m.Name, exp.Name),
					FilePath:        m.FilePath,
				})
			}
		}
	}

	return candidates
}

func inferBasicType(expr ast.Expression) string {
	if expr == nil {
		return "any"
	}
	switch e := expr.(type) {
	case *ast.StringLiteral:
		return "string"
	case *ast.NumberLiteral:
		return "number"
	case *ast.BoolLiteral:
		return "bool"
	case *ast.DurationLiteral:
		return "duration"
	case *ast.WithExpression:
		return "task"
	case *ast.InstantiationExpression:
		return e.UnitName
	case *ast.ListLiteral:
		if len(e.Elements) > 0 {
			return "list[" + inferBasicType(e.Elements[0]) + "]"
		}
		return "list"
	case *ast.MapLiteral:
		return "map"
	case *ast.LambdaExpression:
		return "function"
	default:
		return "any"
	}
}
