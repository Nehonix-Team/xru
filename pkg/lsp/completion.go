package lsp

import "strings"

func (s *Server) handleCompletion(uri string, pos Position) *CompletionList {
	text, ok := s.docs[uri]
	if !ok {
		return &CompletionList{Items: []CompletionItem{}}
	}

	lines := strings.Split(text, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) {
		return &CompletionList{Items: []CompletionItem{}}
	}

	line := lines[pos.Line]
	col := pos.Character
	if col > len(line) {
		col = len(line)
	}
	prefixLine := line[:col]
	trimmedPrefix := strings.TrimSpace(prefixLine)

	// 0. Handle `from` statement autocompletion (modules and namespaces from _mod.xru)
	if strings.HasPrefix(trimmedPrefix, "from") {
		fromItems := s.handleFromCompletion(uri, prefixLine, trimmedPrefix)
		if len(fromItems) > 0 {
			return &CompletionList{Items: fromItems}
		}
	}

	// 0.1 Handle double colon `::` member completion (e.g. auth::login)
	if colonIdx := strings.LastIndex(prefixLine, "::"); colonIdx != -1 {
		memberItems := s.handleScopeMemberCompletion(uri, prefixLine[:colonIdx])
		return &CompletionList{Items: memberItems}
	}

	// 0.2 Handle `use ` statement for builtin units
	if strings.HasPrefix(trimmedPrefix, "use ") || trimmedPrefix == "use" {
		useItems := s.handleUseBuiltinsCompletion()
		return &CompletionList{Items: useItems}
	}

	// 1. Context Check: Inside a `with Net.fetch(...) {` block
	if isInsideWithNet(lines, pos.Line) {
		withItems := s.handleWithNetCompletion(trimmedPrefix)
		return &CompletionList{Items: withItems}
	}

	// 1.5 Context Check: Inside a string literal
	quoteCount := 0
	for i := 0; i < len(prefixLine); i++ {
		if prefixLine[i] == '"' && (i == 0 || prefixLine[i-1] != '\\') {
			quoteCount++
		}
	}
	if quoteCount%2 != 0 {
		// Dans une chaîne de caractères ouverte
		lastOpenBrace := strings.LastIndex(prefixLine, "$f{")
		lastCloseBrace := strings.LastIndex(prefixLine, "}")
		if lastOpenBrace != -1 && lastOpenBrace > lastCloseBrace {
			// Le curseur est à l'intérieur d'un bloc d'interpolation `$f{...}` (ex: `"$f{re|"`)
			// Proposer tous les symboles locaux pertinents (variables, arguments, paramètres, &prop)
			localItems := collectLocalSymbols(lines, pos.Line)
			return &CompletionList{IsIncomplete: false, Items: localItems}
		}
		// Texte normal dans une chaîne (pas d'autocomplétion intempestive de snippets ou mots-clés)
		return &CompletionList{IsIncomplete: false, Items: []CompletionItem{}}
	}

	// 2. Member completion: e.g. "Utils." or "Net." or "Crypto." or "Fs." or "cdn."
	if memberList, handled := s.handleMemberCompletion(uri, lines, prefixLine); handled {
		return memberList
	}

	// 2.5 Local Symbols from current document (variables, constants, functions, params, local imports)
	var items []CompletionItem
	localItems := collectLocalSymbols(lines, pos.Line)
	items = append(items, localItems...)

	// Déterminer la position d'insertion pour l'import (en haut du fichier)
	importInsertLine := 0
	for idx, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "use ") || strings.HasPrefix(trimmed, "from ") || strings.HasPrefix(trimmed, "mod ") {
			importInsertLine = idx + 1
		}
	}

	// 3. Global completions: snippets, built-in units, surgical methods, keywords, workspace auto-imports
	globalItems := s.collectGlobalCompletions(uri, lines, importInsertLine)
	items = append(items, globalItems...)

	return &CompletionList{IsIncomplete: false, Items: items}
}
