/***************************************************************************
 * XFPM — XRU AST Node Definitions
 *
 * This defines the XRU language structure.
 ***************************************************************************** */

package xru

// RuleType identifies what a rule does to its target file.
type RuleType string

const (
	RuleTypeBegin  RuleType = "BEGIN"  // Transform an existing file
	RuleTypeCreate RuleType = "CREATE" // Create a new file
	RuleTypeGlobal RuleType = "GLOBAL" // Apply to all matching files
)

// PatchOp is the structured mutation operation.
type PatchOp string

const (
	PatchRM    PatchOp = "RM"    // Remove keys
	PatchRPK   PatchOp = "RPK"   // Rename key
	PatchRPV   PatchOp = "RPV"   // Replace value
	PatchMerge PatchOp = "MERGE"  // Structured Deep-merge
	PatchAppend PatchOp = "APPEND" // Append to array
	PatchRegex  PatchOp = "REGEX"  // Regex replacement
)

// Value represents a structured data piece in the XRU language.
type Value interface {
	IsValue()
}

type Object map[string]Value
type Array []Value
type Literal string

func (Object) IsValue()  {}
func (Array) IsValue()   {}
func (Literal) IsValue() {}

// RuleFile is the top-level result of parsing an .xru file.
type RuleFile struct {
	Rules []Rule
}

// Rule represents a single transformation directive.
type Rule struct {
	Type    RuleType
	Target  string
	Content string
	Actions []Action
}

// Action is the interface for all rule actions.
type Action interface {
	IsAction()
}

// InjectAction replaces a `// xfpm: {{KEY}}` marker.
type InjectAction struct {
	Lang string // Target language/extension (e.g., "TS", "GO")
	Key  string
	Code string
}

func (InjectAction) IsAction() {}

// PatchAction applies a structured mutation.
type PatchAction struct {
	Op    PatchOp
	Value Value
}

func (PatchAction) IsAction() {}
