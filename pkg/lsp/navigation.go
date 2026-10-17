package lsp

import (
	"github.com/Nehonix-Team/xru/pkg/lsp/navigation"
)

// Type aliases for seamless compatibility with the navigation package
type FileImport = navigation.FileImport
type PropertyInfo = navigation.PropertyInfo

// getNavigator returns the initialized Navigator instance for Server.
func (s *Server) getNavigator() *navigation.Navigator {
	if s.navigator != nil {
		return s.navigator
	}
	s.navigator = navigation.NewNavigator(
		func(uri string) (string, bool) {
			s.mu.Lock()
			defer s.mu.Unlock()
			t, ok := s.docs[uri]
			return t, ok
		},
		func() map[string]string {
			s.mu.Lock()
			defer s.mu.Unlock()
			docsCopy := make(map[string]string, len(s.docs))
			for u, t := range s.docs {
				docsCopy[u] = t
			}
			return docsCopy
		},
		func(lines []string, mapStartLine int, currentFilePath string) string {
			return resolveExpectedMapType(lines, mapStartLine, currentFilePath)
		},
		func(lines []string, varName string) *navigation.VarInfo {
			v := findVariableTargetInfo(lines, varName)
			if v == nil {
				return nil
			}
			return &navigation.VarInfo{
				InterfaceName: v.interfaceName,
				UnitType:      v.unitType,
				IsString:      v.isString,
				IsList:        v.isList,
				IsNumber:      v.isNumber,
				IsMap:         v.isMap,
			}
		},
		FindStdDir,
	)
	return s.navigator
}

// handleDefinition dispatches definition lookup to the navigation package.
func (s *Server) handleDefinition(uri string, pos Position) *Location {
	return s.getNavigator().HandleDefinition(uri, pos)
}

// handleReferences dispatches references lookup to the navigation package.
func (s *Server) handleReferences(uri string, pos Position) []Location {
	return s.getNavigator().HandleReferences(uri, pos)
}

// handleDocumentHighlight dispatches document highlight lookup to the navigation package.
func (s *Server) handleDocumentHighlight(uri string, pos Position) []DocumentHighlight {
	return s.getNavigator().HandleDocumentHighlight(uri, pos)
}

// Backward-compatible package-level helpers used across completion and symbols.
func isWordChar(b byte) bool {
	return navigation.IsWordChar(b)
}

func getWordAt(text string, line, char int) string {
	return navigation.GetWordAt(text, line, char)
}

func getFullExpressionAt(text string, line, char int) (string, string) {
	return navigation.GetFullExpressionAt(text, line, char)
}

func parseFileImports(text string) []FileImport {
	return navigation.ParseFileImports(text)
}

func isCursorOnMapKey(lines []string, pos Position) (bool, string, int) {
	return navigation.IsCursorOnMapKey(lines, pos)
}

func findPropertyDeclaration(lines []string, targetType, propName string, uri string) *PropertyInfo {
	dummyNav := navigation.NewNavigator(nil, nil, nil, nil, FindStdDir)
	return dummyNav.FindPropertyDeclaration(lines, targetType, propName, uri)
}
