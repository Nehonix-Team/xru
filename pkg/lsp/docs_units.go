package lsp

// BuiltinUnitsDocs contient la documentation complète des unités et de leurs méthodes,
// chargée dynamiquement et strictement depuis les définitions dans le dossier 'std/'.
var BuiltinUnitsDocs = make(map[string]UnitDoc)
