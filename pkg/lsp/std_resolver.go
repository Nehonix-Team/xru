package lsp

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

var (
	stdDocsMu     sync.RWMutex
	lastLoadedDir string

	reFuncDecl  = regexp.MustCompile(`^\s*func\s+([A-Za-z0-9_]+)\s*\((.*?)\)(\s*->\s*([^{]+))?`)
	reIfaceDecl = regexp.MustCompile(`^\s*interface\s+([A-Za-z0-9_]+)`)
	reUnitDecl  = regexp.MustCompile(`^\s*unit\s+([A-Za-z0-9_]+)`)
)

func init() {
	// Attempt initial load from current working directory or standard locations
	LoadStdDocs("")
}

// FindStdDir locates the standard library directory across project and global paths.
func FindStdDir(curFileDir string) string {
	// 1. Explicit environment variable
	if envStd := os.Getenv("XRU_STD_PATH"); envStd != "" {
		if info, err := os.Stat(envStd); err == nil && info.IsDir() {
			return envStd
		}
	}

	// 2. Local project directories (walking up from curFileDir)
	if curFileDir != "" {
		dir := curFileDir
		for i := 0; i < 15; i++ {
			cand := filepath.Join(dir, "std")
			if info, err := os.Stat(cand); err == nil && info.IsDir() {
				return cand
			}
			cand2 := filepath.Join(dir, "mods", "xru", "std")
			if info, err := os.Stat(cand2); err == nil && info.IsDir() {
				return cand2
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// 3. Working directory walking up
	if wd, err := os.Getwd(); err == nil {
		dir := wd
		for i := 0; i < 15; i++ {
			cand := filepath.Join(dir, "std")
			if info, err := os.Stat(cand); err == nil && info.IsDir() {
				return cand
			}
			cand2 := filepath.Join(dir, "mods", "xru", "std")
			if info, err := os.Stat(cand2); err == nil && info.IsDir() {
				return cand2
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// 4. Relative to current binary executable (e.g. ~/.xfpm/bin/../lib/xru/std)
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		candXfpmLib := filepath.Join(execDir, "..", "lib", "xru", "std")
		if info, err := os.Stat(candXfpmLib); err == nil && info.IsDir() {
			return candXfpmLib
		}
		candStd := filepath.Join(execDir, "std")
		if info, err := os.Stat(candStd); err == nil && info.IsDir() {
			return candStd
		}
		candParentStd := filepath.Join(execDir, "..", "std")
		if info, err := os.Stat(candParentStd); err == nil && info.IsDir() {
			return candParentStd
		}
	}

	// 5. Global XFPM installations (Windows, macOS, Linux)
	if xfpmEnv := os.Getenv("XFPM_INSTALL"); xfpmEnv != "" {
		cand := filepath.Join(xfpmEnv, "lib", "xru", "std")
		if info, err := os.Stat(cand); err == nil && info.IsDir() {
			return cand
		}
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candHome := filepath.Join(home, ".xfpm", "lib", "xru", "std")
		if info, err := os.Stat(candHome); err == nil && info.IsDir() {
			return candHome
		}
	}
	// Windows specific paths
	if localApp := os.Getenv("LOCALAPPDATA"); localApp != "" {
		candWin := filepath.Join(localApp, "xfpm", "lib", "xru", "std")
		if info, err := os.Stat(candWin); err == nil && info.IsDir() {
			return candWin
		}
	}
	if appData := os.Getenv("APPDATA"); appData != "" {
		candWin := filepath.Join(appData, "xfpm", "lib", "xru", "std")
		if info, err := os.Stat(candWin); err == nil && info.IsDir() {
			return candWin
		}
	}
	if progData := os.Getenv("ProgramData"); progData != "" {
		candWin := filepath.Join(progData, "xfpm", "lib", "xru", "std")
		if info, err := os.Stat(candWin); err == nil && info.IsDir() {
			return candWin
		}
	}
	// Unix / macOS system locations
	for _, sysBase := range []string{"/usr/local/share/xfpm/lib/xru/std", "/opt/xfpm/lib/xru/std"} {
		if info, err := os.Stat(sysBase); err == nil && info.IsDir() {
			return sysBase
		}
	}

	return ""
}

// findStdFile locates a specific .xru standard library file.
func findStdFile(curFileDir, name string) string {
	if !strings.HasSuffix(name, ".xru") {
		name += ".xru"
	}
	stdDir := FindStdDir(curFileDir)
	if stdDir != "" {
		target := filepath.Join(stdDir, name)
		if info, err := os.Stat(target); err == nil && !info.IsDir() {
			return target
		}
	}
	return ""
}

// EnsureStdDocsLoaded guarantees standard documentation is populated for the given context.
func EnsureStdDocsLoaded(curFileDir string) {
	stdDir := FindStdDir(curFileDir)
	if stdDir == "" {
		return
	}
	stdDocsMu.RLock()
	alreadyLoaded := (lastLoadedDir == stdDir && len(BuiltinUnitsDocs) > 0)
	stdDocsMu.RUnlock()

	if !alreadyLoaded {
		LoadStdDocs(curFileDir)
	}
}

// LoadStdDocs parses all .xru files in the resolved std directory and populates documentation maps.
func LoadStdDocs(curFileDir string) {
	stdDir := FindStdDir(curFileDir)
	if stdDir == "" {
		return
	}

	entries, err := os.ReadDir(stdDir)
	if err != nil {
		return
	}

	stdDocsMu.Lock()
	defer stdDocsMu.Unlock()

	if BuiltinUnitsDocs == nil {
		BuiltinUnitsDocs = make(map[string]UnitDoc)
	}
	if StringMethodsDocs == nil {
		StringMethodsDocs = make(map[string]DocItem)
	}
	if ListMethodsDocs == nil {
		ListMethodsDocs = make(map[string]DocItem)
	}
	if NumberMethodsDocs == nil {
		NumberMethodsDocs = make(map[string]DocItem)
	}
	if MapMethodsDocs == nil {
		MapMethodsDocs = make(map[string]DocItem)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".xru") {
			continue
		}
		filePath := filepath.Join(stdDir, entry.Name())
		contentBytes, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		baseName := strings.TrimSuffix(entry.Name(), ".xru")
		parseStdFileContent(baseName, string(contentBytes))
	}

	lastLoadedDir = stdDir
}

func parseStdFileContent(fileBaseName, content string) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")

	var docComments []string
	pkgDoc := ""
	currentIface := ""
	currentUnit := fileBaseName

	// Map unit base name to standard display name
	unitDisplay := fileBaseName
	if strings.EqualFold(fileBaseName, "net") {
		unitDisplay = "Net"
	} else if strings.EqualFold(fileBaseName, "fs") {
		unitDisplay = "Fs"
	} else if strings.EqualFold(fileBaseName, "crypto") {
		unitDisplay = "Crypto"
	}

	unitKey := strings.ToLower(fileBaseName)
	existingUnit, hasUnit := BuiltinUnitsDocs[unitKey]
	if !hasUnit {
		existingUnit = UnitDoc{
			Detail:  "unit " + unitDisplay,
			Doc:     "",
			Methods: make(map[string]DocItem),
		}
	} else if existingUnit.Methods == nil {
		existingUnit.Methods = make(map[string]DocItem)
	}

	for _, rawLine := range lines {
		trimmed := strings.TrimSpace(rawLine)

		if strings.HasPrefix(trimmed, "// Package ") {
			parts := strings.SplitN(trimmed, "// Package ", 2)
			if len(parts) == 2 {
				pkgDoc = strings.TrimSpace(parts[1])
			}
			continue
		}

		if strings.HasPrefix(trimmed, "///") {
			comment := strings.TrimPrefix(trimmed, "///")
			comment = strings.TrimPrefix(comment, " ")
			docComments = append(docComments, comment)
			continue
		}

		if strings.HasPrefix(trimmed, "//") {
			// Regular single line comment, do not accumulate in docComments
			continue
		}

		if m := reIfaceDecl.FindStringSubmatch(trimmed); m != nil {
			currentIface = m[1]
			docComments = nil
			continue
		}

		if m := reUnitDecl.FindStringSubmatch(trimmed); m != nil {
			currentUnit = m[1]
			unitKey = strings.ToLower(currentUnit)
			unitDisplay = currentUnit
			if len(docComments) > 0 {
				existingUnit.Doc = strings.Join(docComments, "\n")
			}
			docComments = nil
			continue
		}

		if m := reFuncDecl.FindStringSubmatch(trimmed); m != nil {
			funcName := m[1]
			params := strings.TrimSpace(m[2])
			retType := "void"
			if len(m) > 4 && strings.TrimSpace(m[4]) != "" {
				retType = strings.TrimSpace(m[4])
			}
			doc := strings.Join(docComments, "\n")
			docComments = nil

			funcLower := strings.ToLower(funcName)

			// 1. Primitive interfaces in builtin.xru
			switch currentIface {
			case "String":
				item := DocItem{
					Detail: fmt.Sprintf("(string).%s(%s) -> %s", funcName, params, retType),
					Doc:    doc,
				}
				StringMethodsDocs[funcLower] = item
				if strings.Contains(funcName, "_") {
					StringMethodsDocs[strings.ReplaceAll(funcLower, "_", "")] = item
				}
				continue

			case "List":
				item := DocItem{
					Detail: fmt.Sprintf("(list).%s(%s) -> %s", funcName, params, retType),
					Doc:    doc,
				}
				ListMethodsDocs[funcLower] = item
				continue

			case "Number":
				item := DocItem{
					Detail: fmt.Sprintf("(number).%s(%s) -> %s", funcName, params, retType),
					Doc:    doc,
				}
				NumberMethodsDocs[funcLower] = item
				if strings.Contains(funcName, "_") {
					NumberMethodsDocs[strings.ReplaceAll(funcLower, "_", "")] = item
				}
				continue

			case "Map":
				item := DocItem{
					Detail: fmt.Sprintf("(map).%s(%s) -> %s", funcName, params, retType),
					Doc:    doc,
				}
				MapMethodsDocs[funcLower] = item
				continue

			case "Document":
				docUnit, ok := BuiltinUnitsDocs["document"]
				if !ok {
					docUnit = UnitDoc{
						Detail:  "unit Document",
						Doc:     "Multi-view surgical file manipulation unit providing raw, linewise, JSON, and fuzzy editing capabilities.",
						Methods: make(map[string]DocItem),
					}
				}
				docUnit.Methods[funcLower] = DocItem{
					Detail: fmt.Sprintf("doc.%s(%s) -> %s", funcName, params, retType),
					Doc:    doc,
				}
				docUnit.Methods[funcName] = docUnit.Methods[funcLower]
				BuiltinUnitsDocs["document"] = docUnit
				continue
			}

			// 2. Unit methods (defined in <Unit>Native or unit <name>)
			methodDetail := fmt.Sprintf("%s.%s(%s) -> %s", unitDisplay, funcName, params, retType)
			item := DocItem{
				Detail: methodDetail,
				Doc:    doc,
			}
			existingUnit.Methods[funcLower] = item
			existingUnit.Methods[funcName] = item
			continue
		}

		if trimmed == "}" {
			currentIface = ""
			docComments = nil
			continue
		}

		if trimmed != "" {
			docComments = nil
		}
	}

	if existingUnit.Doc == "" && pkgDoc != "" {
		existingUnit.Doc = pkgDoc
	}

	BuiltinUnitsDocs[unitKey] = existingUnit
	BuiltinUnitsDocs[strings.ToLower(fileBaseName)] = existingUnit
}
