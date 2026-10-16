package lsp

// StringMethodsDocs contient la documentation des méthodes natives sur les chaînes
var StringMethodsDocs = map[string]DocItem{
	"len": {
		Detail: "(string).len() -> number",
		Doc:    "Returns the length (number of characters) of the string.",
	},
	"trim": {
		Detail: "(string).trim() -> string",
		Doc:    "Returns the string with leading and trailing whitespace removed.",
	},
	"trimleft": {
		Detail: "(string).trimLeft() -> string",
		Doc:    "Returns the string with leading whitespace removed.",
	},
	"trimright": {
		Detail: "(string).trimRight() -> string",
		Doc:    "Returns the string with trailing whitespace removed.",
	},
	"toupper": {
		Detail: "(string).toUpper() -> string",
		Doc:    "Returns the string with all characters converted to uppercase.",
	},
	"tolower": {
		Detail: "(string).toLower() -> string",
		Doc:    "Returns the string with all characters converted to lowercase.",
	},
	"contains": {
		Detail: "(string).contains(substr: string) -> bool",
		Doc:    "Reports whether `substr` is within the string.",
	},
	"startswith": {
		Detail: "(string).startsWith(prefix: string) -> bool",
		Doc:    "Tests whether the string begins with `prefix`.",
	},
	"endswith": {
		Detail: "(string).endsWith(suffix: string) -> bool",
		Doc:    "Tests whether the string ends with `suffix`.",
	},
	"split": {
		Detail: "(string).split(separator: string) -> list[string]",
		Doc:    "Slices the string into all substrings separated by `separator` and returns a list of them.",
	},
	"replace": {
		Detail: "(string).replace(old: string, new: string) -> string",
		Doc:    "Returns a copy of the string with all occurrences of `old` replaced by `new`.",
	},
	"slice": {
		Detail: "(string).slice(start: number, end?: number) -> string",
		Doc:    "Returns a substring starting from index `start` up to optional index `end` (or the end of the string).",
	},
	"tostring": {
		Detail: "(string).toString() -> string",
		Doc:    "Returns the string itself.",
	},
	"to_string": {
		Detail: "(string).to_string() -> string",
		Doc:    "Returns the string itself.",
	},
}

// ListMethodsDocs contient la documentation des méthodes natives sur les listes
var ListMethodsDocs = map[string]DocItem{
	"len": {
		Detail: "(list).len() -> number",
		Doc:    "Returns the number of elements in the list.",
	},
	"join": {
		Detail: "(list).join(separator: string) -> string",
		Doc:    "Concatenates the elements of the list to create a single string, using `separator` between elements.",
	},
	"contains": {
		Detail: "(list).contains(item: any) -> bool",
		Doc:    "Reports whether `item` is present in the list.",
	},
	"push": {
		Detail: "(list).push(item: any) -> void",
		Doc:    "Appends a new element to the end of the list in place.",
	},
	"pop": {
		Detail: "(list).pop() -> any",
		Doc:    "Removes and returns the last element of the list.",
	},
	"reverse": {
		Detail: "(list).reverse() -> list",
		Doc:    "Returns a new list with elements in reverse order.",
	},
	"slice": {
		Detail: "(list).slice(start: number, end?: number) -> list",
		Doc:    "Returns a new list containing a portion of the list from index `start` up to optional index `end`.",
	},
	"get": {
		Detail: "(list).get(index: number) -> any",
		Doc:    "Returns the element at the specified index (supports negative index).",
	},
	"at": {
		Detail: "(list).at(index: number) -> any",
		Doc:    "Returns the element at the specified index (supports negative index).",
	},
}

// NumberMethodsDocs contient la documentation des méthodes natives sur les nombres
var NumberMethodsDocs = map[string]DocItem{
	"abs": {
		Detail: "(number).abs() -> number",
		Doc:    "Returns the absolute value of the number.",
	},
	"floor": {
		Detail: "(number).floor() -> number",
		Doc:    "Returns the largest integer less than or equal to the number.",
	},
	"ceil": {
		Detail: "(number).ceil() -> number",
		Doc:    "Returns the smallest integer greater than or equal to the number.",
	},
	"round": {
		Detail: "(number).round() -> number",
		Doc:    "Returns the nearest integer, rounding half away from zero.",
	},
	"fixed": {
		Detail: "(number).fixed(digits: number) -> string",
		Doc:    "Formats the number using fixed-point notation with specified decimal digits.",
	},
	"int": {
		Detail: "(number).int() -> number",
		Doc:    "Truncates the number to an integer.",
	},
	"tostring": {
		Detail: "(number).toString() -> string",
		Doc:    "Converts the number to its string representation.",
	},
	"to_string": {
		Detail: "(number).to_string() -> string",
		Doc:    "Converts the number to its string representation.",
	},
}

// MapMethodsDocs contient la documentation des méthodes natives sur les maps
var MapMethodsDocs = map[string]DocItem{
	"keys": {
		Detail: "(map).keys() -> list[string]",
		Doc:    "Returns a list containing all the keys in the map.",
	},
	"values": {
		Detail: "(map).values() -> list[any]",
		Doc:    "Returns a list containing all the values in the map.",
	},
	"entries": {
		Detail: "(map).entries() -> list[any]",
		Doc:    "Returns a list of key-value pairs from the map.",
	},
	"has": {
		Detail: "(map).has(key: string) -> bool",
		Doc:    "Returns true if the map contains the specified key.",
	},
	"len": {
		Detail: "(map).len() -> number",
		Doc:    "Returns the number of entries in the map.",
	},
	"delete": {
		Detail: "(map).delete(key: string) -> void",
		Doc:    "Removes the entry with the specified key from the map.",
	},
	"clear": {
		Detail: "(map).clear() -> void",
		Doc:    "Removes all entries from the map.",
	},
}
