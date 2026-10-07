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
}

func (s *UseStatement) statementNode()      {}
func (s *UseStatement) Pos() lexer.Position { return s.Position }
func (s *UseStatement) String() string {
	if s.Alias != "" {
		return fmt.Sprintf("#USE %s as %s", s.Module, s.Alias)
	}
	return fmt.Sprintf("#USE %s", s.Module)
}

// ArgStatement : #ARG NAME: type or default
type ArgStatement struct {
	Position     lexer.Position
	Name         string
	Type         string // "string", "number", "bool", "duration", etc.
	DefaultValue Expression
}

func (s *ArgStatement) statementNode()      {}
func (s *ArgStatement) Pos() lexer.Position { return s.Position }
func (s *ArgStatement) String() string {
	if s.DefaultValue != nil {
		return fmt.Sprintf("#ARG %s: %s or %s", s.Name, s.Type, s.DefaultValue.String())
	}
	return fmt.Sprintf("#ARG %s: %s", s.Name, s.Type)
}

// VarStatement : let/const name: type = expr
type VarStatement struct {
	Position lexer.Position
	IsConst  bool
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

// -----------------------------------------------------------------------------
// Déclarations #UNIT, #DECLARE, #DIRECTIVE, #FUNC, #RETURN
// -----------------------------------------------------------------------------

// UnitStatement : #UNIT Name ... #END
type UnitStatement struct {
	Position   lexer.Position
	Name       string
	Properties []*DeclareStatement
	Directives []*DirectiveDefStatement
	Methods    []*FuncStatement
}

func (s *UnitStatement) statementNode()      {}
func (s *UnitStatement) Pos() lexer.Position { return s.Position }
func (s *UnitStatement) String() string {
	return fmt.Sprintf("#UNIT %s (%d props, %d directives, %d methods)", s.Name, len(s.Properties), len(s.Directives), len(s.Methods))
}

// DeclareStatement : #DECLARE prop: type = expr
type DeclareStatement struct {
	Position lexer.Position
	Name     string
	Type     string
	Value    Expression
}

func (s *DeclareStatement) statementNode()      {}
func (s *DeclareStatement) Pos() lexer.Position { return s.Position }
func (s *DeclareStatement) String() string {
	return fmt.Sprintf("#DECLARE %s: %s = %s", s.Name, s.Type, s.Value.String())
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
	return fmt.Sprintf("#DIRECTIVE %s(%d params)", s.Name, len(s.Parameters))
}

// FuncStatement : #FUNC Name(params...) -> ReturnType ... #END
type FuncStatement struct {
	Position   lexer.Position
	Name       string
	Parameters []*Parameter
	ReturnType string
	Body       []Statement
}

func (s *FuncStatement) statementNode()      {}
func (s *FuncStatement) Pos() lexer.Position { return s.Position }
func (s *FuncStatement) String() string {
	return fmt.Sprintf("#FUNC %s -> %s", s.Name, s.ReturnType)
}

type Parameter struct {
	Name string
	Type string
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
	return fmt.Sprintf("#WITH %s (%d stmts)", s.Target.String(), len(s.Body))
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
	Position lexer.Position
	Path     string
}

func (s *DispatchTreeStatement) statementNode()      {}
func (s *DispatchTreeStatement) Pos() lexer.Position { return s.Position }
func (s *DispatchTreeStatement) String() string      { return fmt.Sprintf("#DISPATCH_TREE %s", s.Path) }

type FileStatement struct {
	Position lexer.Position
	Path     string
	Body     []Statement
}

func (s *FileStatement) statementNode()      {}
func (s *FileStatement) Pos() lexer.Position { return s.Position }
func (s *FileStatement) String() string      { return fmt.Sprintf("#FILE %s", s.Path) }

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
	return fmt.Sprintf("%s(%d args)", e.Caller.String(), len(e.Arguments))
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
func (e *ListLiteral) String() string      { return fmt.Sprintf("[%d elements]", len(e.Elements)) }

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
