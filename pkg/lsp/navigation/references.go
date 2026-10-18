package navigation

import (
	"fmt"
	"regexp"
	"strings"
)

// HandleReferences finds all occurrences of the word at pos across all documents.
func (n *Navigator) HandleReferences(uri string, pos Position) []Location {
	if n.GetDoc == nil {
		return []Location{}
	}
	text, ok := n.GetDoc(uri)
	if !ok {
		return []Location{}
	}

	word := GetWordAt(text, pos.Line, pos.Character)
	if word == "" {
		return []Location{}
	}

	var locs []Location
	re := regexp.MustCompile(fmt.Sprintf(`\b(%s)\b`, regexp.QuoteMeta(word)))

	var docsCopy map[string]string
	if n.GetAllDocs != nil {
		docsCopy = n.GetAllDocs()
	} else {
		docsCopy = map[string]string{uri: text}
	}

	for docURI, docText := range docsCopy {
		lines := strings.Split(docText, "\n")
		for idx, line := range lines {
			matches := re.FindAllStringIndex(line, -1)
			for _, m := range matches {
				locs = append(locs, Location{
					URI: docURI,
					Range: Range{
						Start: Position{Line: idx, Character: m[0]},
						End:   Position{Line: idx, Character: m[1]},
					},
				})
			}
		}
	}
	return locs
}

// HandleDocumentHighlight finds all highlights of the word at pos in the current document.
func (n *Navigator) HandleDocumentHighlight(uri string, pos Position) []DocumentHighlight {
	if n.GetDoc == nil {
		return []DocumentHighlight{}
	}
	text, ok := n.GetDoc(uri)
	if !ok {
		return []DocumentHighlight{}
	}

	word := GetWordAt(text, pos.Line, pos.Character)
	if word == "" {
		return []DocumentHighlight{}
	}

	var highlights []DocumentHighlight
	lines := strings.Split(text, "\n")
	re := regexp.MustCompile(fmt.Sprintf(`\b(%s)\b`, regexp.QuoteMeta(word)))

	for idx, line := range lines {
		matches := re.FindAllStringIndex(line, -1)
		for _, m := range matches {
			highlights = append(highlights, DocumentHighlight{
				Range: Range{
					Start: Position{Line: idx, Character: m[0]},
					End:   Position{Line: idx, Character: m[1]},
				},
				Kind: HighlightKindRead,
			})
		}
	}
	return highlights
}
