package navigation

// Position represents a 0-indexed line and character in a document.
type Position struct {
	Line      int `json:"line"`      // 0-indexed
	Character int `json:"character"` // 0-indexed
}

// Range represents a start and end Position in a document.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location represents a location inside a resource.
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// DocumentHighlightKind represents the kind of highlight.
type DocumentHighlightKind int

const (
	HighlightKindText  DocumentHighlightKind = 1
	HighlightKindRead  DocumentHighlightKind = 2
	HighlightKindWrite DocumentHighlightKind = 3
)

// DocumentHighlight represents a highlighted range in a document.
type DocumentHighlight struct {
	Range Range                 `json:"range"`
	Kind  DocumentHighlightKind `json:"kind,omitempty"`
}

// FileImport stores parsed import statements from a source file.
type FileImport struct {
	Source  string   // Namespace e.g. "dll"
	Module  string   // Module e.g. "gArgs"
	Member  string   // Member e.g. "author"
	Members []string // Grouped members e.g. ["author", "version", ...]
	Alias   string   // Alias e.g. "Fs"
	Line    int      // 0-indexed line where import statement is written
}

// PropertyInfo describes a property or method declaration in a type/interface.
type PropertyInfo struct {
	TypeName   string
	PropName   string
	PropType   string
	IsOptional bool
	IsMethod   bool
	Doc        string
	Line       int
	CharStart  int
	CharEnd    int
	URI        string
}

// VarInfo contains resolved type information for a variable.
type VarInfo struct {
	InterfaceName string
	UnitType      string
	IsString      bool
	IsList        bool
	IsNumber      bool
	IsMap         bool
}

// Navigator provides definition, highlight, and reference navigation operations.
type Navigator struct {
	GetDoc                 func(uri string) (string, bool)
	GetAllDocs             func() map[string]string
	ResolveExpectedMapType func(lines []string, mapStartLine int, currentFilePath string) string
	FindVarInfo            func(lines []string, varName string) *VarInfo
	FindStdDir             func(curFileDir string) string
}

// NewNavigator creates a new Navigator with the required resolution delegates.
func NewNavigator(
	getDoc func(uri string) (string, bool),
	getAllDocs func() map[string]string,
	resolveExpectedMapType func(lines []string, mapStartLine int, currentFilePath string) string,
	findVarInfo func(lines []string, varName string) *VarInfo,
	findStdDir func(curFileDir string) string,
) *Navigator {
	return &Navigator{
		GetDoc:                 getDoc,
		GetAllDocs:             getAllDocs,
		ResolveExpectedMapType: resolveExpectedMapType,
		FindVarInfo:            findVarInfo,
		FindStdDir:             findStdDir,
	}
}
