/***************************************************************************
 * XFPM — Structured Text Patcher (STP)
 *
 * This module applies XRU transformations to text files by identifying 
 * structure (braces, keys) without parsing the entire file as JSON.
 * It preserves existing comments, indentation, and XyPriss-specific syntax.
 ***************************************************************************** */

package engine

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ApplyPatch applies a structured patch to a string content.
func ApplyPatch(content string, op PatchOp, patch Value) string {
	switch op {
	case PatchMerge:
		if obj, ok := patch.(Object); ok {
			return mergeStructured(content, obj)
		}
	case PatchRM:
		return removeStructured(content, patch)
	case PatchAppend:
		if obj, ok := patch.(Object); ok {
			return appendStructured(content, obj)
		}
	case PatchRegex:
		if obj, ok := patch.(Object); ok {
			return applyRegex(content, obj)
		}
	}
	return content
}

func removeStructured(content string, patch Value) string {
	switch p := patch.(type) {
	case Object:
		for k, v := range p {
			// Look for key in content
			keyPatternQuoted := fmt.Sprintf("%q:", k)
			keyPatternBare := k + ":"
			
			start := strings.Index(content, keyPatternQuoted)
			if start == -1 {
				start = strings.Index(content, keyPatternBare)
			}

			if start != -1 {
				if nestedPatch, ok := v.(Object); ok {
					// Recurse into object
					openOff := strings.Index(content[start:], "{")
					if openOff != -1 {
						absOpen := start + openOff
						closeAbs := matchingBrace(content, absOpen)
						if closeAbs != -1 {
							inner := content[absOpen : closeAbs+1]
							newInner := removeStructured(inner, nestedPatch)
							content = content[:absOpen] + newInner + content[closeAbs+1:]
						}
					}
				} else {
					// Remove the key and its value (potentially a whole block)
					lineStart := strings.LastIndex(content[:start], "\n")
					if lineStart == -1 { lineStart = 0 } else { lineStart++ }
					
					// Check if it's a block in the target file
					sepPos := strings.Index(content[start:], ":")
					if sepPos != -1 {
						afterSep := content[start+sepPos+1:]
						p := 0
						for p < len(afterSep) && (afterSep[p] == ' ' || afterSep[p] == '\t' || afterSep[p] == '\r') {
							p++
						}
						if p < len(afterSep) && afterSep[p] == '{' {
							// It's a block! Find matching brace
							absOpen := start + sepPos + 1 + p
							closeAbs := matchingBrace(content, absOpen)
							if closeAbs != -1 {
								// Find the end of that line (to include potential comma)
								lineEnd := strings.Index(content[closeAbs:], "\n")
								if lineEnd == -1 { lineEnd = len(content[closeAbs:]) }
								absEnd := closeAbs + lineEnd
								content = content[:lineStart] + content[absEnd+1:]
								continue
							}
						}
					}

					// Standard single-line removal
					lineEnd := strings.Index(content[start:], "\n")
					if lineEnd == -1 { lineEnd = len(content) - start }
					absEnd := start + lineEnd
					content = content[:lineStart] + content[absEnd+1:]
				}
			}
		}
	case Array:
		for _, v := range p {
			if s, ok := v.(Literal); ok {
				content = removeStructured(content, Object{string(s): Literal("")})
			}
		}
	}
	return content
}

func mergeStructured(content string, patch Object) string {
	for k, v := range patch {
		// Look for key in content
		keyPatternQuoted := fmt.Sprintf("%q:", k)
		keyPatternBare := k + ":"
		
		start := strings.Index(content, keyPatternQuoted)
		if start == -1 {
			start = strings.Index(content, keyPatternBare)
		}

		if start != -1 {
			switch val := v.(type) {
			case Object:
				// Find opening brace
				openOff := strings.Index(content[start:], "{")
				if openOff != -1 {
					absOpen := start + openOff
					closeAbs := matchingBrace(content, absOpen)
					if closeAbs != -1 {
						inner := content[absOpen : closeAbs+1]
						newInner := mergeStructured(inner, val)
						content = content[:absOpen] + newInner + content[closeAbs+1:]
					}
				}
			default:
				// Update existing literal value
				// Find the end of the line or the next comma
				lineEnd := strings.Index(content[start:], "\n")
				if lineEnd == -1 { lineEnd = len(content) - start }
				
				absEnd := start + lineEnd
				// Simple replacement of the value part
				// This is a bit naive but works for simple configs
				sepPos := strings.Index(content[start:absEnd], ":")
				if sepPos != -1 {
					prefix := content[:start+sepPos+1]
					suffix := content[absEnd:]
					// Keep trailing comma if present
					comma := ""
					if strings.Contains(content[start+sepPos:absEnd], ",") {
						comma = ","
					}
					content = prefix + " " + serialiseValue(v, "") + comma + suffix
				}
			}
		} else {
			// Key missing, inject
			content = injectAtEnd(content, k, v)
		}
	}
	return content
}

func injectAtEnd(content, key string, value Value) string {
	// Find last }
	pos := strings.LastIndex(content, "}")
	if pos == -1 {
		return content
	}

	// Determine indentation
	indent := "  "
	lastNewline := strings.LastIndex(content[:pos], "\n")
	if lastNewline != -1 {
		line := content[lastNewline+1 : pos]
		spaces := 0
		for _, r := range line {
			if r == ' ' { spaces++ } else { break }
		}
		if spaces > 0 {
			indent = strings.Repeat(" ", spaces)
		}
	}

	serialised := serialiseValue(value, indent)
	prefix := strings.TrimRight(content[:pos], " \t\n\r")
	
	// Add comma to previous entry if needed
	if !strings.HasSuffix(prefix, "{") && !strings.HasSuffix(prefix, ",") && !strings.HasSuffix(prefix, "[") {
		prefix += ","
	}
	
	entry := fmt.Sprintf("\n%s%q: %s,", indent, key, serialised)

	return prefix + entry + "\n" + content[pos:]
}

func serialiseValue(v Value, indent string) string {
	switch val := v.(type) {
	case Object:
		if len(val) == 0 { return "{}" }
		res := "{\n"
		for k, v := range val {
			res += fmt.Sprintf("%s  %q: %s,\n", indent, k, serialiseValue(v, indent+"  "))
		}
		res += indent + "}"
		return res
	case Array:
		if len(val) == 0 { return "[]" }
		res := "[\n"
		for _, v := range val {
			res += fmt.Sprintf("%s  %s,\n", indent, serialiseValue(v, indent+"  "))
		}
		res += indent + "]"
		return res
	case Literal:
		s := string(val)
		// Always quote strings to be safe, unless it's a number or boolean
		if s == "true" || s == "false" || s == "null" {
			return s
		}
		// Try to see if it's a number
		var n float64
		if _, err := fmt.Sscanf(s, "%f", &n); err == nil && !strings.Contains(s, " ") {
			return s
		}
		// Otherwise quote it
		b, _ := json.Marshal(s)
		return string(b)
	}
	return ""
}

func appendStructured(content string, patch Object) string {
	for k, v := range patch {
		keyPatternQuoted := fmt.Sprintf("%q:", k)
		keyPatternBare := k + ":"
		
		start := strings.Index(content, keyPatternQuoted)
		if start == -1 {
			start = strings.Index(content, keyPatternBare)
		}

		if start != -1 {
			// Find opening bracket [
			openOff := strings.Index(content[start:], "[")
			if openOff != -1 {
				absOpen := start + openOff
				closeAbs := matchingBracket(content, absOpen)
				if closeAbs != -1 {
					// Inject before ]
					indent := "  "
					// Naive indentation check
					lastNewline := strings.LastIndex(content[:closeAbs], "\n")
					if lastNewline != -1 {
						line := content[lastNewline+1 : closeAbs]
						spaces := 0
						for _, r := range line {
							if r == ' ' { spaces++ } else { break }
						}
						indent = strings.Repeat(" ", spaces+2)
					}
					
					serialised := serialiseValue(v, indent)
					prefix := strings.TrimRight(content[:closeAbs], " \t\n\r")
					// Check if array is empty
					separator := ","
					if strings.TrimSpace(content[absOpen+1:closeAbs]) == "" {
						separator = ""
					}
					
					entry := fmt.Sprintf("%s\n%s%s,", separator, indent, serialised)
					content = prefix + entry + "\n" + strings.Repeat(" ", strings.Count(indent, " ")-2) + content[closeAbs:]
				}
			}
		}
	}
	return content
}

func applyRegex(content string, patch Object) string {
	for k, v := range patch {
		if val, ok := v.(Literal); ok {
			re, err := regexp.Compile(k)
			if err != nil {
				continue
			}
			content = re.ReplaceAllString(content, string(val))
		}
	}
	return content
}

func matchingBracket(s string, openIdx int) int {
	depth := 0
	inString := false
	var quote rune

	runes := []rune(s)
	for i := openIdx; i < len(runes); i++ {
		r := runes[i]
		if inString {
			if r == quote && runes[i-1] != '\\' {
				inString = false
			}
			continue
		}

		if r == '"' || r == '\'' {
			inString = true
			quote = r
			continue
		}

		if r == '[' {
			depth++
		} else if r == ']' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func matchingBrace(s string, openIdx int) int {
	depth := 0
	inString := false
	var quote rune

	runes := []rune(s)
	for i := openIdx; i < len(runes); i++ {
		r := runes[i]
		if inString {
			if r == quote && runes[i-1] != '\\' {
				inString = false
			}
			continue
		}

		if r == '"' || r == '\'' {
			inString = true
			quote = r
			continue
		}

		if r == '{' {
			depth++
		} else if r == '}' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
