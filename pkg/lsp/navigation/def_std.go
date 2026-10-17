package navigation

import (
	"os"
	"path/filepath"
	"strings"
)

// resolveStdDefinition resolves definitions for standard library primitives, modules, and global functions.
func (n *Navigator) resolveStdDefinition(word, fileDir string) *Location {
	// 1. Builtin primitive contracts & types in std/builtin.xru
	lowerWord := strings.ToLower(word)
	if lowerWord == "string" || lowerWord == "list" || lowerWord == "number" || lowerWord == "map" {
		targetIface := strings.ToUpper(lowerWord[:1]) + lowerWord[1:]
		if stdBuiltin := n.findStdFile(fileDir, "builtin"); stdBuiltin != "" {
			if lIdx, charPos, found := FindDeclarationInFile(stdBuiltin, targetIface); found {
				return &Location{
					URI: pathToURI(stdBuiltin),
					Range: Range{
						Start: Position{Line: lIdx, Character: charPos},
						End:   Position{Line: lIdx, Character: charPos + len(targetIface)},
					},
				}
			}
		}
	}
	if word == "DateInfo" {
		if stdDate := n.findStdFile(fileDir, "date"); stdDate != "" {
			if lIdx, charPos, found := FindDeclarationInFile(stdDate, "DateInfo"); found {
				return &Location{
					URI: pathToURI(stdDate),
					Range: Range{
						Start: Position{Line: lIdx, Character: charPos},
						End:   Position{Line: lIdx, Character: charPos + len(word)},
					},
				}
			}
		}
	}

	// 2. Standard library modules: std, path, fs, crypto, date, math, sys, cmd, json, net
	for _, m := range []string{"std", "path", "fs", "crypto", "date", "math", "sys", "cmd", "json", "net"} {
		if strings.EqualFold(word, m) {
			if stdFile := n.findStdFile(fileDir, strings.ToLower(word)); stdFile != "" {
				return &Location{
					URI: pathToURI(stdFile),
					Range: Range{
						Start: Position{Line: 0, Character: 0},
						End:   Position{Line: 0, Character: len(word)},
					},
				}
			}
		}
	}

	// 3. Standard library global functions and declarations (e.g. Drop, drop, Keep, keep, println)
	if stdBuiltin := n.findStdFile(fileDir, "builtin"); stdBuiltin != "" {
		if lIdx, charPos, found := FindDeclarationInFile(stdBuiltin, word); found {
			return &Location{
				URI: pathToURI(stdBuiltin),
				Range: Range{
					Start: Position{Line: lIdx, Character: charPos},
					End:   Position{Line: lIdx, Character: charPos + len(word)},
				},
			}
		}
	}
	if n.FindStdDir != nil {
		if stdDir := n.FindStdDir(fileDir); stdDir != "" {
			if stdEntries, err := os.ReadDir(stdDir); err == nil {
				for _, entry := range stdEntries {
					if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".xru") {
						fp := filepath.Join(stdDir, entry.Name())
						if lIdx, charPos, found := FindDeclarationInFile(fp, word); found {
							return &Location{
								URI: pathToURI(fp),
								Range: Range{
									Start: Position{Line: lIdx, Character: charPos},
									End:   Position{Line: lIdx, Character: charPos + len(word)},
								},
							}
						}
					}
				}
			}
		}
	}

	return nil
}

func (n *Navigator) findStdFile(curFileDir, name string) string {
	if !strings.HasSuffix(name, ".xru") {
		name += ".xru"
	}
	if n.FindStdDir == nil {
		return ""
	}
	stdDir := n.FindStdDir(curFileDir)
	if stdDir != "" {
		target := filepath.Join(stdDir, name)
		if info, err := os.Stat(target); err == nil && !info.IsDir() {
			return target
		}
	}
	return ""
}
