package parser

import (
	"github.com/alecthomas/participle/v2/lexer"
)

type Program struct {
	Pos lexer.Position

	Name                 *string                `"program" @Ident`
	Sensor               *string                `"sensor" @Ident`
	VariableDeclarations []*VariableDeclaration `"var" @@ ";" ( ( @@ ";" )* )?`
	Statements           []*Statement           `"begin" @@ ";" ( ( "|" @@ ";" )* )? "end"`
	InitialState         []*Assignment          `( "init" "state" ( @@  ";" )+ )?`
}

type VariableDeclaration struct {
	Pos lexer.Position

	Access *string        `@ ( "public" | "private" )?`
	Type   *string        `@ ( "int" | "string" | "bool" )`
	Name   *LocalVariable `@@`
}

type LocalVariable struct {
	Pos lexer.Position

	Identifier *string `@Ident`
	Source     *string `("." @Ident)?`
}

type CopyVariable struct {
	Pos lexer.Position

	Source     *LocalVariable `"copy" "." "[" @@ "]"`
	Identifier *string        `@Ident`
}

type Statement struct {
	Pos lexer.Position

	Guard      *Guard      `"(" @@ ")" "->"`
	Assignment *Assignment `@@`
}

type BinaryGuard struct {
	Pos lexer.Position

	Left     *Expression `@@`
	Operator *string     `@( ">" | "<" | "==" | "!=" | ">=" | "<=" )`
	Right    *Expression `@@`
}

type ForAllGuard struct {
	Pos lexer.Position

	LoopIdentifier  *string   `"forall" @Ident ":"`
	LoopIdentifier2 *string   `@Ident "in"`
	LoopOver        *Variable `@@ ":"`
	Guard           *Guard    `@@`
}

type ConjunctionGuard struct {
	Pos lexer.Position

	Left  *Guard `@@ "&&"`
	Right *Guard `@@`
}

type DisjunctionGuard struct {
	Pos lexer.Position

	Left  *Guard `@@ "||"`
	Right *Guard `@@`
}

type Guard struct {
	Pos lexer.Position

	Expression       *Expression       `@@`
	NegateGuard      *Guard            `| "!" @@`
	BinaryGuard      *BinaryGuard      `| @@`
	ForAllGuard      *ForAllGuard      `| @@`
	ConjunctionGuard *ConjunctionGuard `| @@`
	DisjunctionGuard *DisjunctionGuard `| @@`
	ParenthesisGuard *Guard            `| "(" @@ ")"`
}

type Assignment struct {
	Pos lexer.Position

	Variable   *string     `@Ident "="`
	Expression *Expression `@@`
}

type Operand struct {
	Pos lexer.Position

	IntValue    *int    `@Number`
	BoolTrue    *bool   `| @True`
	BoolFalse   *bool   `| @False`
	StringValue *string `| @String`
}

type Variable struct {
	Pos lexer.Position

	CopyIdentifier *CopyVariable  `@@`
	LocalVariable  *LocalVariable `@@`
}

type UnaryExpression struct {
	Pos lexer.Position

	UnaryOperator *string     `@ "not"`
	Expression    *Expression `@@`
}

type BinaryExpression struct {
	Pos lexer.Position

	Left     *Expression `@@`
	Operator *string     `@( "+" | "-" | "*" | "/" | "%" )`
	Right    *Expression `@@`
}

type Expression struct {
	Pos lexer.Position

	Operand               *Operand          `@@`
	Variable              *Variable         `| @@`
	UniaryExpression      *UnaryExpression  `| @@`
	BinaryExpression      *BinaryExpression `| @@`
	ParenthesisExpression *Expression       `| "(" @@ ")"`
}
