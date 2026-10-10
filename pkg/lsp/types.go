package lsp

import "encoding/json"

// JSON-RPC 2.0 Basic types
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      any         `json:"id"`
	Result  any         `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type Notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// LSP Position & Range
type Position struct {
	Line      int `json:"line"`      // 0-indexed
	Character int `json:"character"` // 0-indexed
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// Diagnostic
type DiagnosticSeverity int

const (
	SeverityError       DiagnosticSeverity = 1
	SeverityWarning     DiagnosticSeverity = 2
	SeverityInformation DiagnosticSeverity = 3
	SeverityHint        DiagnosticSeverity = 4
)

type Diagnostic struct {
	Range    Range              `json:"range"`
	Severity DiagnosticSeverity `json:"severity"`
	Source   string             `json:"source"`
	Message  string             `json:"message"`
}

type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// Document Synchronization
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type VersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier
	Version int `json:"version"`
}

type TextDocumentContentChangeEvent struct {
	Range       *Range `json:"range,omitempty"`
	RangeLength int    `json:"rangeLength,omitempty"`
	Text        string `json:"text"`
}

type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// Hover
type HoverParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type MarkupContent struct {
	Kind  string `json:"kind"` // "markdown" or "plaintext"
	Value string `json:"value"`
}

type Hover struct {
	Contents MarkupContent `json:"contents"`
	Range    *Range        `json:"range,omitempty"`
}

// Definition
type DefinitionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// Document Symbols
type DocumentSymbolParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type SymbolKind int

const (
	SymbolKindFile        SymbolKind = 1
	SymbolKindModule      SymbolKind = 2
	SymbolKindNamespace   SymbolKind = 3
	SymbolKindPackage     SymbolKind = 4
	SymbolKindClass       SymbolKind = 5
	SymbolKindMethod      SymbolKind = 6
	SymbolKindProperty    SymbolKind = 7
	SymbolKindField       SymbolKind = 8
	SymbolKindConstructor SymbolKind = 9
	SymbolKindEnum        SymbolKind = 10
	SymbolKindInterface   SymbolKind = 11
	SymbolKindFunction    SymbolKind = 12
	SymbolKindVariable    SymbolKind = 13
	SymbolKindConstant    SymbolKind = 14
)

type DocumentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Kind           SymbolKind       `json:"kind"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

// Initialize
type InitializeParams struct {
	ProcessID int    `json:"processId"`
	RootURI   string `json:"rootUri,omitempty"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   *ServerInfo        `json:"serverInfo,omitempty"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type TextDocumentSyncOptions struct {
	OpenClose bool `json:"openClose"`
	Change    int  `json:"change"` // 1 = Full, 2 = Incremental
}

// Completion
type CompletionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type CompletionItemKind int

const (
	CompletionItemKindText        CompletionItemKind = 1
	CompletionItemKindMethod      CompletionItemKind = 2
	CompletionItemKindFunction    CompletionItemKind = 3
	CompletionItemKindConstructor CompletionItemKind = 4
	CompletionItemKindField       CompletionItemKind = 5
	CompletionItemKindVariable    CompletionItemKind = 6
	CompletionItemKindClass       CompletionItemKind = 7
	CompletionItemKindInterface   CompletionItemKind = 8
	CompletionItemKindModule      CompletionItemKind = 9
	CompletionItemKindProperty    CompletionItemKind = 10
	CompletionItemKindUnit        CompletionItemKind = 11
	CompletionItemKindValue       CompletionItemKind = 12
	CompletionItemKindEnum        CompletionItemKind = 13
	CompletionItemKindKeyword     CompletionItemKind = 14
	CompletionItemKindSnippet     CompletionItemKind = 15
	CompletionItemKindFolder      CompletionItemKind = 19
	CompletionItemKindEvent       CompletionItemKind = 23
)

const (
	InsertTextFormatPlainText int = 1
	InsertTextFormatSnippet   int = 2
)

type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

type CompletionItem struct {
	Label               string             `json:"label"`
	Kind                CompletionItemKind `json:"kind,omitempty"`
	Detail              string             `json:"detail,omitempty"`
	Documentation       any                `json:"documentation,omitempty"` // MarkupContent or string
	InsertText          string             `json:"insertText,omitempty"`
	FilterText          string             `json:"filterText,omitempty"`
	InsertTextFormat    int                `json:"insertTextFormat,omitempty"` // 1 = PlainText, 2 = Snippet
	AdditionalTextEdits []TextEdit         `json:"additionalTextEdits,omitempty"`
}


type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}

type CompletionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

// Document Highlight
type DocumentHighlightParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type DocumentHighlightKind int

const (
	HighlightKindText  DocumentHighlightKind = 1
	HighlightKindRead  DocumentHighlightKind = 2
	HighlightKindWrite DocumentHighlightKind = 3
)

type DocumentHighlight struct {
	Range Range                 `json:"range"`
	Kind  DocumentHighlightKind `json:"kind,omitempty"`
}

// References
type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

type ReferenceParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	Context      ReferenceContext       `json:"context"`
}

type ServerCapabilities struct {
	TextDocumentSync          TextDocumentSyncOptions `json:"textDocumentSync"`
	HoverProvider             bool                    `json:"hoverProvider"`
	DefinitionProvider        bool                    `json:"definitionProvider"`
	DocumentSymbolProvider    bool                    `json:"documentSymbolProvider"`
	CompletionProvider        *CompletionOptions      `json:"completionProvider,omitempty"`
	DocumentHighlightProvider bool                    `json:"documentHighlightProvider"`
	ReferencesProvider        bool                    `json:"referencesProvider"`
	CodeActionProvider        bool                    `json:"codeActionProvider"`
}

// Code Actions (QuickFix / Auto-Import)
type CodeActionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	Context      CodeActionContext      `json:"context"`
}

type CodeActionContext struct {
	Diagnostics []Diagnostic `json:"diagnostics"`
}

type CodeAction struct {
	Title       string         `json:"title"`
	Kind        string         `json:"kind,omitempty"`
	Diagnostics []Diagnostic   `json:"diagnostics,omitempty"`
	IsPreferred bool           `json:"isPreferred,omitempty"`
	Edit        *WorkspaceEdit `json:"edit,omitempty"`
}

type WorkspaceEdit struct {
	Changes map[string][]TextEdit `json:"changes,omitempty"`
}



