package ast

import (
	"fmt"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/lexer"
)

// Node représente tout élément de l'AST
type Node interface {
	Pos() lexer.Position
	String() string
}

// Statement représente une instruction
type Statement interface {
	Node
	statementNode()
}

// Expression représente une expression évaluable
type Expression interface {
	Node
	expressionNode()
}

// Program est le nœud racine d'un fichier .xru
type Program struct {
	Statements []Statement
}

func (p *Program) Pos() lexer.Position {
	if len(p.Statements) > 0 {
		return p.Statements[0].Pos()
	}
	return lexer.Position{}
}

func (p *Program) String() string {
	var sb strings.Builder
	for _, s := range p.Statements {
		sb.WriteString(s.String() + "\n")
	}
	return sb.String()
}

// -----------------------------------------------------------------------------
// Déclarations de tête (#USE, #ARG, let, const)
// -----------------------------------------------------------------------------

// UseStatement : #USE module as Alias
type UseStatement struct {
	Position lexer.Position
	Module   string
	Alias    string
	CatchVar string
}

func (s *UseStatement) statementNode()      {}
func (s *UseStatement) Pos() lexer.Position { return s.Position }
func (s *UseStatement) String() string {
	if s.Alias != "" {
		return fmt.Sprintf("#USE %s as %s", s.Module, s.Alias)
	}
	return fmt.Sprintf("#USE %s", s.Module)
}

// ArgStatement : [global/export] arg NAME [as ALIAS]: type = default  /  #ARG NAME: type or default
type ArgStatement struct {
	Position     lexer.Position
	TypePos      lexer.Position
	Name         string // Variable name & long flag (e.g. port -> --port)
	Short        string // Short CLI flag alias (e.g. p -> -p)
	Alias        string // Optional mangled internal variable name in bundled code
	Type         string // "string", "number", "bool", "duration", etc.
	DefaultValue Expression
	IsGlobal     bool
}

func (s *ArgStatement) VarName() string {
	if s.Alias != "" {
		return s.Alias
	}
	return s.Name
}

func (s *ArgStatement) statementNode()      {}
func (s *ArgStatement) Pos() lexer.Position { return s.Position }
func (s *ArgStatement) String() string {
	prefix := ""
	if s.IsGlobal {
		prefix = "global "
	}
	namePart := s.Name
	if s.Short != "" && s.Alias != "" {
		namePart = fmt.Sprintf("%s, %s as %s", s.Name, s.Short, s.Alias)
	} else if s.Short != "" {
		namePart = fmt.Sprintf("%s, %s", s.Name, s.Short)
	} else if s.Alias != "" {
		namePart = fmt.Sprintf("%s as %s", s.Name, s.Alias)
	}
	if s.DefaultValue != nil {
		return fmt.Sprintf("%sarg %s: %s = %s", prefix, namePart, s.Type, s.DefaultValue.String())
	}
	return fmt.Sprintf("%sarg %s: %s", prefix, namePart, s.Type)
}

// MapStatement : map { alias -> path }
type MapStatement struct {
	Position lexer.Position
	Mappings map[string]string
}

func (s *MapStatement) statementNode()      {}
func (s *MapStatement) Pos() lexer.Position { return s.Position }
func (s *MapStatement) String() string {
	return fmt.Sprintf("map (%d mappings)", len(s.Mappings))
}

// ModStatement : mod name
type ModStatement struct {
	Position lexer.Position
	Name     string
}

func (s *ModStatement) statementNode()      {}
func (s *ModStatement) Pos() lexer.Position { return s.Position }
func (s *ModStatement) String() string      { return fmt.Sprintf("mod %s", s.Name) }

// FromUseStatement : from "source" use ...
type FromUseStatement struct {
	Position lexer.Position
	Source   string   // e.g. "services"
	Module   string   // e.g. "auth"
	Member   string   // e.g. "b" (when from "services" use auth::b)
	Members  []string // e.g. ["name", "xems"] (when from "services" use auth::@map{name, xems})
	Alias    string   // e.g. "a" (when from "services" use auth as a or auth::@map as a)
	Modules  []string // e.g. ["auth", "cdn", "db"] (when from "services" use (auth, cdn, db))
	AsMap    bool     // true when module::@map or module::@map{ ... }
	CatchVar        string                  // e.g. "e" (when from "services" use auth::b catch e)
	MemberPositions map[string]lexer.Position // Position de chaque membre ou module importé
}

func (s *FromUseStatement) statementNode()      {}
func (s *FromUseStatement) Pos() lexer.Position { return s.Position }
func (s *FromUseStatement) String() string {
	if len(s.Modules) > 0 {
		return fmt.Sprintf("from %q use (%s)", s.Source, strings.Join(s.Modules, ", "))
	}
	if s.AsMap {
		if len(s.Members) > 0 {
			return fmt.Sprintf("from %q use %s::@map{%s}", s.Source, s.Module, strings.Join(s.Members, ", "))
		}
		if s.Alias != "" {
			return fmt.Sprintf("from %q use %s::@map as %s", s.Source, s.Module, s.Alias)
		}
		return fmt.Sprintf("from %q use %s::@map", s.Source, s.Module)
	}
	if len(s.Members) > 0 {
		return fmt.Sprintf("from %q use %s::(%s)", s.Source, s.Module, strings.Join(s.Members, ", "))
	}
	if s.Member != "" {
		return fmt.Sprintf("from %q use %s::%s", s.Source, s.Module, s.Member)
	}
	if s.Alias != "" {
		return fmt.Sprintf("from %q use %s as %s", s.Source, s.Module, s.Alias)
	}
	return fmt.Sprintf("from %q use %s", s.Source, s.Module)
}

// InterfaceStatement : interface Name @map { field: type, ... }
type InterfaceStatement struct {
	Position lexer.Position
	Name     string
	IsMap    bool
	Fields   []*InterfaceField
}

type InterfaceField struct {
	Position   lexer.Position
	Name       string
	Type       string
	Optional   bool
	Nested     *InterfaceStatement
	Params     []*Parameter
	ReturnType string
}

func (s *InterfaceStatement) statementNode()      {}
func (s *InterfaceStatement) Pos() lexer.Position { return s.Position }
func (s *InterfaceStatement) String() string {
	if s.IsMap {
		return fmt.Sprintf("interface %s @map { ... }", s.Name)
	}
	return fmt.Sprintf("interface %s { ... }", s.Name)
}

// VarStatement : let/const name: type = expr
type VarStatement struct {
	Position lexer.Position
	NamePos  lexer.Position
	TypePos  lexer.Position
	IsConst  bool
	IsGlobal bool
	Name     string
	Type     string
	Value    Expression
}

func (s *VarStatement) statementNode()      {}
func (s *VarStatement) Pos() lexer.Position { return s.Position }
func (s *VarStatement) String() string {
	keyword := "let"
	if s.IsConst {
		keyword = "const"
	}
	if s.Type != "" {
		return fmt.Sprintf("%s %s: %s = %s", keyword, s.Name, s.Type, s.Value.String())
	}
	return fmt.Sprintf("%s %s = %s", keyword, s.Name, s.Value.String())
}

// ReassignStatement : name =: expr (mutation of let variables)
type ReassignStatement struct {
	Position lexer.Position
	Name     string
	Value    Expression
}

func (s *ReassignStatement) statementNode()      {}
func (s *ReassignStatement) Pos() lexer.Position { return s.Position }
func (s *ReassignStatement) String() string {
	return fmt.Sprintf("%s =: %s", s.Name, s.Value.String())
}

// -----------------------------------------------------------------------------
// Déclarations #UNIT, #DECLARE, #DIRECTIVE, #FUNC, #RETURN
// -----------------------------------------------------------------------------

// UnitStatement : #UNIT Name ... #END
type UnitStatement struct {
	Position   lexer.Position
	Name       string
	Includes   string
	Properties []*DeclareStatement
	Directives []*DirectiveDefStatement
	Methods    []*FuncStatement
}

func (s *UnitStatement) statementNode()      {}
func (s *UnitStatement) Pos() lexer.Position { return s.Position }
func (s *UnitStatement) String() string {
	return fmt.Sprintf("#UNIT %s (%d props, %d directives, %d methods)", s.Name, len(s.Properties), len(s.Directives), len(s.Methods))
}

// DeclareStatement : #DECLARE prop: type = expr (with optional public/private modifier)
type DeclareStatement struct {
	Position  lexer.Position
	Name      string
	Type      string
	Value     Expression
	IsPrivate bool
}

func (s *DeclareStatement) statementNode()      {}
func (s *DeclareStatement) Pos() lexer.Position { return s.Position }
func (s *DeclareStatement) String() string {
	vis := ""
	if s.IsPrivate {
		vis = "private "
	}
	if s.Value != nil {
		return fmt.Sprintf("declare %s%s: %s = %s", vis, s.Name, s.Type, s.Value.String())
	}
	return fmt.Sprintf("declare %s%s: %s", vis, s.Name, s.Type)
}

// DirectiveDefStatement : #DIRECTIVE Name(params...) ... #END
type DirectiveDefStatement struct {
	Position   lexer.Position
	Name       string
	Parameters []*Parameter
	Body       []Statement
}

func (s *DirectiveDefStatement) statementNode()      {}
func (s *DirectiveDefStatement) Pos() lexer.Position { return s.Position }
func (s *DirectiveDefStatement) String() string {
	return fmt.Sprintf("directive %s(%d params)", s.Name, len(s.Parameters))
}

// FuncStatement : #FUNC Name(params...) -> ReturnType ... #END
type FuncStatement struct {
	Position   lexer.Position
	Name       string
	Parameters []*Parameter
	ReturnType string
	Body       []Statement
	IsGlobal   bool
	NativeHook string
}

func (s *FuncStatement) statementNode()      {}
func (s *FuncStatement) Pos() lexer.Position { return s.Position }
func (s *FuncStatement) String() string {
	return fmt.Sprintf("#FUNC %s -> %s", s.Name, s.ReturnType)
}

type Parameter struct {
	Name         string
	Type         string
	Optional     bool
	IsVariadic   bool
	DefaultValue Expression
}

// ReturnStatement : #RETURN expr
type ReturnStatement struct {
	Position lexer.Position
	Value    Expression
}

func (s *ReturnStatement) statementNode()      {}
func (s *ReturnStatement) Pos() lexer.Position { return s.Position }
func (s *ReturnStatement) String() string {
	if s.Value != nil {
		return fmt.Sprintf("#RETURN %s", s.Value.String())
	}
	return "#RETURN"
}

// -----------------------------------------------------------------------------
// Blocs contextuels (#WITH), Custom Directives (#SET, #HEADER, etc.)
// -----------------------------------------------------------------------------

// WithStatement : #WITH TargetExpr ... #END
type WithStatement struct {
	Position lexer.Position
	Target   Expression
	Body     []Statement
}

func (s *WithStatement) statementNode()      {}
func (s *WithStatement) Pos() lexer.Position { return s.Position }
func (s *WithStatement) String() string {
	var body []string
	for _, st := range s.Body {
		if st != nil {
			body = append(body, st.String())
		}
	}
	if s.Target != nil {
		return fmt.Sprintf("with %s {\n%s\n}", s.Target.String(), strings.Join(body, "\n"))
	}
	return fmt.Sprintf("with {\n%s\n}", strings.Join(body, "\n"))
}

// WithExpression : with TargetExpr (lance TargetExpr en tâche asynchrone concurrente)
type WithExpression struct {
	Position lexer.Position
	Target   Expression
}

func (e *WithExpression) expressionNode()      {}
func (e *WithExpression) Pos() lexer.Position { return e.Position }
func (e *WithExpression) String() string {
	if e.Target != nil {
		return fmt.Sprintf("with %s", e.Target.String())
	}
	return "with"
}

// ForStatement : for item [, index] in iterable { ... }
type ForStatement struct {
	Position lexer.Position
	Item     string
	Index    string // optionnel
	Iterable Expression
	Body     []Statement
}

func (s *ForStatement) statementNode()      {}
func (s *ForStatement) Pos() lexer.Position { return s.Position }
func (s *ForStatement) String() string {
	var body []string
	for _, st := range s.Body {
		if st != nil {
			body = append(body, st.String())
		}
	}
	if s.Index != "" {
		return fmt.Sprintf("for %s, %s in %s {\n%s\n}", s.Item, s.Index, s.Iterable.String(), strings.Join(body, "\n"))
	}
	return fmt.Sprintf("for %s in %s {\n%s\n}", s.Item, s.Iterable.String(), strings.Join(body, "\n"))
}

// WhileStatement : while condition { ... }
type WhileStatement struct {
	Position  lexer.Position
	Condition Expression
	Body      []Statement
}

func (s *WhileStatement) statementNode()      {}
func (s *WhileStatement) Pos() lexer.Position { return s.Position }
func (s *WhileStatement) String() string {
	var body []string
	for _, st := range s.Body {
		if st != nil {
			body = append(body, st.String())
		}
	}
	return fmt.Sprintf("while %s {\n%s\n}", s.Condition.String(), strings.Join(body, "\n"))
}

// RangeExpression : start..end
type RangeExpression struct {
	Position lexer.Position
	Start    Expression
	End      Expression
}

func (e *RangeExpression) expressionNode()      {}
func (e *RangeExpression) Pos() lexer.Position { return e.Position }
func (e *RangeExpression) String() string {
	return fmt.Sprintf("%s..%s", e.Start.String(), e.End.String())
}

// CustomDirectiveInvocation : #DIR_NAME args... (ou #DIR_NAME key -> val)
type CustomDirectiveInvocation struct {
	Position  lexer.Position
	Directive string // ex: "#SET", "#HEADER", "#OPTION"
	Key       Expression
	Value     Expression
}

func (s *CustomDirectiveInvocation) statementNode()      {}
func (s *CustomDirectiveInvocation) Pos() lexer.Position { return s.Position }
func (s *CustomDirectiveInvocation) String() string {
	if s.Value != nil {
		return fmt.Sprintf("%s %s -> %s", s.Directive, s.Key.String(), s.Value.String())
	}
	return fmt.Sprintf("%s %s", s.Directive, s.Key.String())
}

// -----------------------------------------------------------------------------
// Structures de Contrôle (#IF, #MATCH), Dispatch & Files
// -----------------------------------------------------------------------------

type IfStatement struct {
	Position    lexer.Position
	Condition   Expression
	Consequence []Statement
	ElifBlocks  []*ElifStatement
	Alternative []Statement
}

type ElifStatement struct {
	Condition Expression
	Body      []Statement
}

// BlockStatement : { stmt1; stmt2; ... }
type BlockStatement struct {
	Position   lexer.Position
	Statements []Statement
}

func (s *BlockStatement) statementNode()      {}
func (s *BlockStatement) expressionNode()     {}
func (s *BlockStatement) Pos() lexer.Position { return s.Position }
func (s *BlockStatement) String() string {
	var sb strings.Builder
	sb.WriteString("{\n")
	for _, st := range s.Statements {
		sb.WriteString(st.String() + "\n")
	}
	sb.WriteString("}")
	return sb.String()
}

func (s *IfStatement) statementNode()      {}
func (s *IfStatement) Pos() lexer.Position { return s.Position }
func (s *IfStatement) String() string      { return "#IF ... #END" }

type MatchStatement struct {
	Position     lexer.Position
	Target       Expression
	Cases        []*MatchCase
	DefaultBlock []Statement
}

type MatchCase struct {
	Pattern Expression
	Body    []Statement
}

func (s *MatchStatement) statementNode()      {}
func (s *MatchStatement) Pos() lexer.Position { return s.Position }
func (s *MatchStatement) String() string      { return "#MATCH ... #END" }

type DispatchTreeStatement struct {
	Position    lexer.Position
	PathExpr    Expression
	PatternExpr Expression // optional, defaults to ".rules.xru"
	Path        string     // evaluated path string
	Pattern     string     // evaluated pattern string
	PassArgs    map[string]Expression
}

func (s *DispatchTreeStatement) statementNode()      {}
func (s *DispatchTreeStatement) Pos() lexer.Position { return s.Position }
func (s *DispatchTreeStatement) String() string {
	if s.Pattern != "" {
		return fmt.Sprintf("dispatch %s, %s", s.Path, s.Pattern)
	}
	return fmt.Sprintf("dispatch %s", s.Path)
}

// InStatement : in <path> { ... }
type InStatement struct {
	Position lexer.Position
	Path     Expression
	Body     []Statement
}

func (s *InStatement) statementNode()      {}
func (s *InStatement) Pos() lexer.Position { return s.Position }
func (s *InStatement) String() string {
	var body []string
	for _, st := range s.Body {
		if st != nil {
			body = append(body, st.String())
		}
	}
	if s.Path != nil {
		return fmt.Sprintf("in %s {\n%s\n}", s.Path.String(), strings.Join(body, "\n"))
	}
	return fmt.Sprintf("in {\n%s\n}", strings.Join(body, "\n"))
}

type FileStatement struct {
	Position lexer.Position
	Path     string
	PathExpr Expression
	Handler  Expression
	Body     []Statement
}

func (s *FileStatement) statementNode()      {}
func (s *FileStatement) Pos() lexer.Position { return s.Position }
func (s *FileStatement) String() string {
	if s.PathExpr != nil {
		return fmt.Sprintf("file %s", s.PathExpr.String())
	}
	return fmt.Sprintf("file %s", s.Path)
}

// ExpressionStatement encapsule un appel de fonction ou méthode libre
type ExpressionStatement struct {
	Position   lexer.Position
	Expression Expression
}

func (s *ExpressionStatement) statementNode()      {}
func (s *ExpressionStatement) Pos() lexer.Position { return s.Position }
func (s *ExpressionStatement) String() string      { return s.Expression.String() }

// -----------------------------------------------------------------------------
// Expressions (Littéraux, Variables, Appels, Instanciations, Collections)
// -----------------------------------------------------------------------------

type Identifier struct {
	Position lexer.Position
	Value    string
}

func (e *Identifier) expressionNode()      {}
func (e *Identifier) Pos() lexer.Position { return e.Position }
func (e *Identifier) String() string      { return e.Value }

type PropAccess struct {
	Position lexer.Position
	Name     string // ex: "baseDir" pour &baseDir
}

func (e *PropAccess) expressionNode()      {}
func (e *PropAccess) Pos() lexer.Position { return e.Position }
func (e *PropAccess) String() string      { return "&" + e.Name }

type StringLiteral struct {
	Position lexer.Position
	Value    string
}

func (e *StringLiteral) expressionNode()      {}
func (e *StringLiteral) Pos() lexer.Position { return e.Position }
func (e *StringLiteral) String() string      { return fmt.Sprintf("\"%s\"", e.Value) }

type NumberLiteral struct {
	Position lexer.Position
	Value    string
}

func (e *NumberLiteral) expressionNode()      {}
func (e *NumberLiteral) Pos() lexer.Position { return e.Position }
func (e *NumberLiteral) String() string      { return e.Value }

type BoolLiteral struct {
	Position lexer.Position
	Value    bool
}

func (e *BoolLiteral) expressionNode()      {}
func (e *BoolLiteral) Pos() lexer.Position { return e.Position }
func (e *BoolLiteral) String() string      { return fmt.Sprintf("%t", e.Value) }

type DurationLiteral struct {
	Position lexer.Position
	Value    string
}

func (e *DurationLiteral) expressionNode()      {}
func (e *DurationLiteral) Pos() lexer.Position { return e.Position }
func (e *DurationLiteral) String() string      { return e.Value }

// InstantiationExpression : @UnitName(args...)
type InstantiationExpression struct {
	Position  lexer.Position
	UnitName  string
	Arguments []Expression
}

func (e *InstantiationExpression) expressionNode()      {}
func (e *InstantiationExpression) Pos() lexer.Position { return e.Position }
func (e *InstantiationExpression) String() string {
	return fmt.Sprintf("@%s()", e.UnitName)
}

// CallExpression : Target.Method(args...) ou Func(args...)
type CallExpression struct {
	Position  lexer.Position
	Caller    Expression // Identifier ou MemberExpression
	Arguments []Expression
}

func (e *CallExpression) expressionNode()      {}
func (e *CallExpression) Pos() lexer.Position { return e.Position }
func (e *CallExpression) String() string {
	var args []string
	for _, a := range e.Arguments {
		if a != nil {
			args = append(args, a.String())
		}
	}
	return fmt.Sprintf("%s(%s)", e.Caller.String(), strings.Join(args, ", "))
}

type MemberExpression struct {
	Position lexer.Position
	Object   Expression
	Property string
}

func (e *MemberExpression) expressionNode()      {}
func (e *MemberExpression) Pos() lexer.Position { return e.Position }
func (e *MemberExpression) String() string {
	return fmt.Sprintf("%s.%s", e.Object.String(), e.Property)
}

// ListLiteral : [elem1, elem2]
type ListLiteral struct {
	Position lexer.Position
	Elements []Expression
}

func (e *ListLiteral) expressionNode()      {}
func (e *ListLiteral) Pos() lexer.Position { return e.Position }
func (e *ListLiteral) String() string {
	var elems []string
	for _, el := range e.Elements {
		if el != nil {
			elems = append(elems, el.String())
		}
	}
	return fmt.Sprintf("[%s]", strings.Join(elems, ", "))
}

// MapLiteral : ["key" -> "value"]
type MapLiteral struct {
	Position lexer.Position
	Pairs    []*MapPair
}

type MapPair struct {
	Key   Expression
	Value Expression
}

func (e *MapLiteral) expressionNode()      {}
func (e *MapLiteral) Pos() lexer.Position { return e.Position }
func (e *MapLiteral) String() string      { return fmt.Sprintf("[%d pairs]", len(e.Pairs)) }

// PrefixExpression : !expr ou -expr
type PrefixExpression struct {
	Position lexer.Position
	Operator string
	Right    Expression
}

func (e *PrefixExpression) expressionNode()      {}
func (e *PrefixExpression) Pos() lexer.Position { return e.Position }
func (e *PrefixExpression) String() string {
	return fmt.Sprintf("(%s%s)", e.Operator, e.Right.String())
}

// InfixExpression (opérateurs de comparaison, assignation, etc.)
type InfixExpression struct {
	Position lexer.Position
	Left     Expression
	Operator string
	Right    Expression
}

func (e *InfixExpression) expressionNode()      {}
func (e *InfixExpression) Pos() lexer.Position { return e.Position }
func (e *InfixExpression) String() string {
	return fmt.Sprintf("(%s %s %s)", e.Left.String(), e.Operator, e.Right.String())
}

// GlobalCallExpression : ::Func(args...) ou global::Func(args...)
type GlobalCallExpression struct {
	Position  lexer.Position
	Function  string
	Arguments []Expression
}

func (e *GlobalCallExpression) expressionNode()      {}
func (e *GlobalCallExpression) Pos() lexer.Position { return e.Position }
func (e *GlobalCallExpression) String() string {
	var args []string
	for _, a := range e.Arguments {
		args = append(args, a.String())
	}
	return fmt.Sprintf("::%s(%s)", e.Function, strings.Join(args, ", "))
}

// GlobalAccessExpression : ::Var ou global::Var
type GlobalAccessExpression struct {
	Position lexer.Position
	Name     string
}

func (e *GlobalAccessExpression) expressionNode()      {}
func (e *GlobalAccessExpression) Pos() lexer.Position { return e.Position }
func (e *GlobalAccessExpression) String() string      { return fmt.Sprintf("::%s", e.Name) }

// ScopeAccessExpression : Module::Member (ex: auth::b ou a::login) ou &unit::method
type ScopeAccessExpression struct {
	Position   lexer.Position
	Scope      string
	Member     string
	IsUnitSelf bool
}

func (e *ScopeAccessExpression) expressionNode()      {}
func (e *ScopeAccessExpression) Pos() lexer.Position { return e.Position }
func (e *ScopeAccessExpression) String() string {
	if e.IsUnitSelf {
		return fmt.Sprintf("&%s::%s", e.Scope, e.Member)
	}
	return fmt.Sprintf("%s::%s", e.Scope, e.Member)
}

// LambdaExpression : (params...) => Expr ou () => Expr
type LambdaExpression struct {
	Position   lexer.Position
	Parameters []*Parameter
	Body       Expression
}

func (e *LambdaExpression) expressionNode()      {}
func (e *LambdaExpression) Pos() lexer.Position { return e.Position }
func (e *LambdaExpression) String() string {
	var params []string
	for _, p := range e.Parameters {
		params = append(params, p.Name)
	}
	return fmt.Sprintf("(%s) => %s", strings.Join(params, ", "), e.Body.String())
}

// IfExpression : if Condition -> Consequence else Alternative
type IfExpression struct {
	Position    lexer.Position
	Condition   Expression
	Consequence Expression
	Alternative Expression
}

func (e *IfExpression) expressionNode()      {}
func (e *IfExpression) Pos() lexer.Position { return e.Position }
func (e *IfExpression) String() string {
	return fmt.Sprintf("if %s -> %s else %s", e.Condition.String(), e.Consequence.String(), e.Alternative.String())
}

// CatchExpression : Target catch [errVar] [or Fallback]
type CatchExpression struct {
	Position lexer.Position
	Target   Expression
	ErrVar   string
	Fallback Node
}

func (e *CatchExpression) expressionNode()      {}
func (e *CatchExpression) Pos() lexer.Position { return e.Position }
func (e *CatchExpression) String() string {
	res := e.Target.String() + " catch"
	if e.ErrVar != "" {
		res += " " + e.ErrVar
	}
	if e.Fallback != nil {
		res += fmt.Sprintf(" or %s", e.Fallback.String())
	}
	return res
}



