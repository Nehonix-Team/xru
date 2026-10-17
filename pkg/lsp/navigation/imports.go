package navigation

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/modules"
)

// ParseFileImports extracts all import statements from the file content.
func ParseFileImports(text string) []FileImport {
	var imports []FileImport
	lines := strings.Split(text, "\n")

	reFromGroup := regexp.MustCompile(`^\s*from\s+["']?([^"'\s]+)["']?\s+use\s+([a-zA-Z0-9_]+)::(?:@map\s*)?(?:\{|\()([^})]+)(?:\}|\))`)
	reFromMember := regexp.MustCompile(`^\s*from\s+["']?([^"'\s]+)["']?\s+use\s+([a-zA-Z0-9_]+)::([a-zA-Z0-9_]+)(?:\s+as\s+([a-zA-Z0-9_]+))?(?:\s+catch\s+([a-zA-Z0-9_]+))?`)
	reFromMods := regexp.MustCompile(`^\s*from\s+["']?([^"'\s]+)["']?\s+use\s+(?:\{|\()([^})]+)(?:\}|\))`)
	reFromAs := regexp.MustCompile(`^\s*from\s+["']?([^"'\s]+)["']?\s+use\s+([a-zA-Z0-9_]+)\s+as\s+([a-zA-Z0-9_]+)`)
	reFromMod := regexp.MustCompile(`^\s*from\s+["']?([^"'\s]+)["']?\s+use\s+["']?([a-zA-Z0-9_]+)["']?`)
	reUseAs := regexp.MustCompile(`^\s*use\s+([a-zA-Z0-9_]+)\s+as\s+([a-zA-Z0-9_]+)`)
	reUse := regexp.MustCompile(`^\s*use\s+([a-zA-Z0-9_]+)`)

	for idx, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "//") {
			continue
		}
		if m := reFromGroup.FindStringSubmatch(trim); len(m) == 4 {
			rawMembers := strings.Split(m[3], ",")
			var members []string
			for _, rm := range rawMembers {
				mem := strings.TrimSpace(rm)
				if mem != "" {
					members = append(members, mem)
				}
			}
			imports = append(imports, FileImport{
				Source:  m[1],
				Module:  m[2],
				Members: members,
				Line:    idx,
			})
		} else if m := reFromMember.FindStringSubmatch(trim); len(m) >= 4 {
			alias := ""
			if len(m) > 4 && m[4] != "" {
				alias = m[4]
			}
			imports = append(imports, FileImport{
				Source: m[1],
				Module: m[2],
				Member: m[3],
				Alias:  alias,
				Line:   idx,
			})
		} else if m := reFromMods.FindStringSubmatch(trim); len(m) == 3 {
			rawMods := strings.Split(m[2], ",")
			for _, rm := range rawMods {
				mod := strings.TrimSpace(rm)
				if mod != "" {
					imports = append(imports, FileImport{
						Source: m[1],
						Module: mod,
						Line:   idx,
					})
				}
			}
		} else if m := reFromAs.FindStringSubmatch(trim); len(m) == 4 {
			imports = append(imports, FileImport{
				Source: m[1],
				Module: m[2],
				Alias:  m[3],
				Line:   idx,
			})
		} else if m := reFromMod.FindStringSubmatch(trim); len(m) == 3 {
			imports = append(imports, FileImport{
				Source: m[1],
				Module: m[2],
				Line:   idx,
			})
		} else if m := reUseAs.FindStringSubmatch(trim); len(m) == 3 {
			imports = append(imports, FileImport{
				Module: m[1],
				Alias:  m[2],
				Line:   idx,
			})
		} else if m := reUse.FindStringSubmatch(trim); len(m) == 2 {
			imports = append(imports, FileImport{
				Module: m[1],
				Line:   idx,
			})
		}
	}
	return imports
}

// ResolveModuleFilePath locates the file path for a module across mappings, local files, and stdlib.
func (n *Navigator) ResolveModuleFilePath(modCfg *modules.ModConfig, sourceNs, modName, curFileDir string) string {
	if modCfg != nil {
		if sourceNs != "" {
			if modDetails, err := modCfg.ResolveModule(sourceNs, modName); err == nil && modDetails != nil {
				return modDetails.FilePath
			}
		} else {
			for ns := range modCfg.Mappings {
				if modDetails, err := modCfg.ResolveModule(ns, modName); err == nil && modDetails != nil {
					return modDetails.FilePath
				}
			}
		}
	}
	localPath := filepath.Join(curFileDir, modName+".xru")
	if info, err := os.Stat(localPath); err == nil && !info.IsDir() {
		return localPath
	}
	if stdPath := n.findStdFile(curFileDir, modName); stdPath != "" {
		return stdPath
	}
	return ""
}
