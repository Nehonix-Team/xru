package engine

// RuleFile is the top-level container for a .xru file.
type RuleFile struct {
	Rules []Rule
}

// RuleType defines the category of a transformation rule.
type RuleType string

const (
	RuleTypeBegin  RuleType = "BEGIN"  // Transform an existing file
	RuleTypeCreate RuleType = "CREATE" // Create a new file
	RuleTypeSelect RuleType = "SELECT" // Define a target sandbox/directory
	RuleTypeBreak  RuleType = "BREAK"  // Exit the program
	RuleTypeLog    RuleType = "LOG"    // Display a message
	RuleTypeIf      RuleType = "IF"      // Conditional block
	RuleTypeElseIf  RuleType = "ELSEIF"  // Conditional alternative
	RuleTypeElse    RuleType = "ELSE"    // Fallback block
	RuleTypeInclude RuleType = "INCLUDE" // Include another rule file
	RuleTypeExec    RuleType = "EXEC"    // Execute a shell command
	RuleTypeGlobal  RuleType = "GLOBAL"  // Apply to all matching files
	RuleTypeVar     RuleType = "VAR"     // Variable declaration
	RuleTypeUse     RuleType = "USE"     // Load a module
	RuleTypeModule  RuleType = "MODULE"  // Call a module method
	RuleTypeArg     RuleType = "ARG"     // Read terminal argument
	RuleTypeFor     RuleType = "FOR"     // Loop over a list
)

// PatchOp defines the type of patch operation.
type PatchOp string

const (
	PatchMerge  PatchOp = "MERGE"  // Merge JSON objects or text blocks
	PatchSet    PatchOp = "SET"    // Set a specific key or value
	PatchRM     PatchOp = "REMOVE" // Remove a key or block
	PatchPush   PatchOp = "PUSH"   // Append to a JSON array
	PatchRPK    PatchOp = "RPK"    // Replace key
	PatchRPV    PatchOp = "RPV"    // Replace value
	PatchAppend PatchOp = "APPEND" // Append text
	PatchRegex  PatchOp = "REGEX"  // Regex replacement
)

// Value represents a generic value in the rules (string, object, array).
type Value interface {
	IsValue()
}

type Literal string
func (Literal) IsValue() {}

type Object map[string]Value
func (Object) IsValue() {}

type Array []Value
func (Array) IsValue() {}

// Rule represents a single transformation directive.
type Rule struct {
	Type     RuleType
	Target   string
	As       string // Captured variable name
	Content  string
	Actions  []Action
	SubRules []Rule
	Raw      bool
	Line     int
}

// Action is the interface for all rule actions.
type Action interface {
	IsAction()
}

// PatchAction represents a modification to structured text.
type PatchAction struct {
	Op    PatchOp
	Path  string
	Value Value
	Line  int
}

func (PatchAction) IsAction() {}

// InjectAction represents a code injection at a specific marker.
type InjectAction struct {
	Lang string
	Key  string
	Code string
	Raw  bool
	Line int
}

func (InjectAction) IsAction() {}

// VarAction represents a variable declaration inside a rule.
type VarAction struct {
	Name  string
	Value string
	Line  int
}

func (VarAction) IsAction() {}

// ModuleAction represents a call to an external module (e.g. Utils.LOG).
type ModuleAction struct {
	Module string
	Method string
	Target string
	As     string
	Line   int
}

func (ModuleAction) IsAction() {}
