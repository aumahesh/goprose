package parser

import (
	"github.com/alecthomas/participle/v2/lexer"
)

type Program struct {
	Pos lexer.Position

	Name                 *string                `"program" @Ident`
	Sensor               *string                `"sensor" @Ident`
	VariableDeclarations []*VariableDeclaration `"var" @@ ";" ( ( @@ ";" )* )?`
	Statements           []*Statement           `"begin" @@ ( ( "|" @@ )* )? "end"`
	InitialState         []*Assignment          `( "init" "state" ( @@ )+ )?`
}

type VariableDeclaration struct {
	Pos lexer.Position

	Access *string   `@ ( "public" | "private" )?`
	Type   *string   `@ ( "int" | "string" | "bool" )`
	Name   *Variable `@@`
}

type Variable struct {
	Pos lexer.Position

	Identifier *string         `@Ident`
	Source     *VariableSource `("." @@)`
}

type VariableSource struct {
	Pos lexer.Position

	Identifier *string   `@Ident`
	Variable   *Variable `| "(" @@ ")"`
}

type Statement struct {
	Pos lexer.Position

	Guard       *Guard        `@@ "-" ">"`
	Assignments []*Assignment `( @@ )+`
}

type BinaryGuard struct {
	Pos lexer.Position

	Left     *Expression `@@`
	Operator *string     `@( ">" | "<" | "=" "=" | "!" "=" | ">" "=" | "<" "=" )`
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

	Left     *Guard  `@@ "&" "&"`
	Right    *Guard  `@@`
}

type DisjunctionGuard struct {
	Pos lexer.Position

	Left     *Guard  `@@ "|" "|"`
	Right    *Guard  `@@`
}

type Guard struct {
	Pos lexer.Position

	// ParenthesisGuard   *Guard              `"(" @@ ")"`
	// ConjunctionGuard *ConjunctionGuard `| @@`
	// DisjunctionGuard *DisjunctionGuard `| @@`
	// NegateGuard        *Guard              `| "!" @@`
	// BinaryGuard        *BinaryGuard        `| @@`
	// ForAllGuard        *ForAllGuard        `| @@`
	// Expression         *Expression         `| @@`

	ParenthesisGuard   *Guard              `"(" @@ ")"`
	BinaryGuard        *BinaryGuard        `| @@`
	Expression         *Expression         `| @@`
}

type Assignment struct {
	Pos lexer.Position

	Variables   []*Variable   `@@ ( ( "," @@ )* )? "="`
	Expressions []*Expression `@@ ( ( "," @@ )* )? ";"`
}

type Operand struct {
	Pos lexer.Position

	IntValue    *int    `@Number`
	BoolValue    *string `| @ ( "true" | "false" )`
	StringValue *string `| @String`
	Identifier *string `| @Ident`
}

type BinaryExpression struct {
	Pos lexer.Position

	Left     *Expression `@@`
	Operator *string     `@( "+" | "-" | "*" | "/" | "%" )`
	Right    *Expression `@@`
}

type Expression struct {
	Pos lexer.Position

	// ParenthesisExpression *Expression       `"(" @@ ")"`
	// NegateExpression      *Expression       `| "!" @@`
	// BinaryExpression      *BinaryExpression `| @@`
	// Variable              *Variable         `| @@`
	// Operand               *Operand          `| @@`

	Variable              *Variable         `@@`
	Operand               *Operand          `| @@`

}
