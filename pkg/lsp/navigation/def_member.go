package navigation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/modules"
)

// resolveMemberDefinition resolves "obj.member" dot-access expressions.
func (n *Navigator) resolveMemberDefinition(lines []string, fileImports []FileImport, modCfg *modules.ModConfig, fullExpr, word, uri, fileDir string) *Location {
	dotIdx := strings.LastIndex(fullExpr, ".")
	if dotIdx == -1 {
		return nil
	}
	rawObj := strings.TrimSpace(fullExpr[:dotIdx])
	memberName := strings.TrimSpace(fullExpr[dotIdx+1:])
	rawObj = strings.TrimPrefix(rawObj, "$f{")
	rawObj = strings.TrimPrefix(rawObj, "${")
	rawObj = strings.TrimPrefix(rawObj, "{")
	objName := rawObj

	if objName == "" || memberName == "" || word == objName {
		return nil
	}

	if n.FindVarInfo != nil {
		if varInfo := n.FindVarInfo(lines, objName); varInfo != nil {
			targetIface := varInfo.InterfaceName
			if targetIface == "" {
				targetIface = varInfo.UnitType
			}
			if targetIface != "" {
				if loc := n.FindMemberInInterface(lines, targetIface, memberName, uri); loc != nil {
					return loc
				}
			}
			// Builtin primitive methods: string, list, number, map
			if varInfo.IsString || strings.EqualFold(targetIface, "string") || objName == "string" {
				if stdBuiltin := n.findStdFile(fileDir, "builtin"); stdBuiltin != "" {
					if lIdx, charPos, found := FindDeclarationInFile(stdBuiltin, memberName); found {
						return &Location{
							URI: pathToURI(stdBuiltin),
							Range: Range{
								Start: Position{Line: lIdx, Character: charPos},
								End:   Position{Line: lIdx, Character: charPos + len(memberName)},
							},
						}
					}
				}
			}
			if varInfo.IsList || strings.EqualFold(targetIface, "list") || objName == "list" {
				if stdBuiltin := n.findStdFile(fileDir, "builtin"); stdBuiltin != "" {
					if lIdx, charPos, found := FindDeclarationInFile(stdBuiltin, memberName); found {
						return &Location{
							URI: pathToURI(stdBuiltin),
							Range: Range{
								Start: Position{Line: lIdx, Character: charPos},
								End:   Position{Line: lIdx, Character: charPos + len(memberName)},
							},
						}
					}
				}
			}
			if varInfo.IsNumber || strings.EqualFold(targetIface, "number") || objName == "number" {
				if stdBuiltin := n.findStdFile(fileDir, "builtin"); stdBuiltin != "" {
					if lIdx, charPos, found := FindDeclarationInFile(stdBuiltin, memberName); found {
						return &Location{
							URI: pathToURI(stdBuiltin),
							Range: Range{
								Start: Position{Line: lIdx, Character: charPos},
								End:   Position{Line: lIdx, Character: charPos + len(memberName)},
							},
						}
					}
				}
			}
			if varInfo.IsMap || strings.EqualFold(targetIface, "map") || objName == "map" {
				if stdBuiltin := n.findStdFile(fileDir, "builtin"); stdBuiltin != "" {
					if lIdx, charPos, found := FindDeclarationInFile(stdBuiltin, memberName); found {
						return &Location{
							URI: pathToURI(stdBuiltin),
							Range: Range{
								Start: Position{Line: lIdx, Character: charPos},
								End:   Position{Line: lIdx, Character: charPos + len(memberName)},
							},
						}
					}
				}
			}
			// Builtin Document methods in std/builtin.xru
			if targetIface == "Document" {
				if stdBuiltin := n.findStdFile(fileDir, "builtin"); stdBuiltin != "" {
					if lIdx, charPos, found := FindDeclarationInFile(stdBuiltin, memberName); found {
						return &Location{
							URI: pathToURI(stdBuiltin),
							Range: Range{
								Start: Position{Line: lIdx, Character: charPos},
								End:   Position{Line: lIdx, Character: charPos + len(memberName)},
							},
						}
					}
				}
			}
			// Inline map in function or variable
			reFieldDecl := regexp.MustCompile(fmt.Sprintf(`\b(%s)\s*\??\s*(?::|\s+)`, regexp.QuoteMeta(memberName)))
			for idx, l := range lines {
				if strings.Contains(l, "@map") && strings.Contains(l, memberName) {
					if matchLoc := reFieldDecl.FindStringSubmatchIndex(l); matchLoc != nil && len(matchLoc) >= 4 {
						return &Location{
							URI: uri,
							Range: Range{
								Start: Position{Line: idx, Character: matchLoc[2]},
								End:   Position{Line: idx, Character: matchLoc[3]},
							},
						}
					}
				}
			}
		}
	}

	// Check module or imported alias (e.g. Fs.exists, std.println, Date.now)
	targetMod := objName
	targetSource := ""
	for _, imp := range fileImports {
		if imp.Alias != "" && imp.Alias == objName {
			targetMod = imp.Module
			targetSource = imp.Source
			break
		} else if imp.Module == objName {
			targetMod = imp.Module
			targetSource = imp.Source
			break
		}
	}

	// Check external module in _mod.xru
	if targetSource != "" {
		modFile := n.ResolveModuleFilePath(modCfg, targetSource, targetMod, fileDir)
		if modFile != "" {
			if lIdx, charPos, found := FindDeclarationInFile(modFile, memberName); found {
				return &Location{
					URI: pathToURI(modFile),
					Range: Range{
						Start: Position{Line: lIdx, Character: charPos},
						End:   Position{Line: lIdx, Character: charPos + len(memberName)},
					},
				}
			}
		}
	}

	// Check standard library module (fs, std, path, crypto, date, math, sys, cmd, json, net)
	lowerMod := strings.ToLower(targetMod)
	if stdFile := n.findStdFile(fileDir, lowerMod); stdFile != "" {
		if lIdx, charPos, found := FindDeclarationInFile(stdFile, memberName); found {
			return &Location{
				URI: pathToURI(stdFile),
				Range: Range{
					Start: Position{Line: lIdx, Character: charPos},
					End:   Position{Line: lIdx, Character: charPos + len(memberName)},
				},
			}
		}
	}

	// Check builtin methods in std/builtin.xru
	if stdBuiltin := n.findStdFile(fileDir, "builtin"); stdBuiltin != "" {
		if lIdx, charPos, found := FindDeclarationInFile(stdBuiltin, memberName); found {
			return &Location{
				URI: pathToURI(stdBuiltin),
				Range: Range{
					Start: Position{Line: lIdx, Character: charPos},
					End:   Position{Line: lIdx, Character: charPos + len(memberName)},
				},
			}
		}
	}

	return nil
}
