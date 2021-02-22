package parser

import (
	"github.com/alecthomas/participle/v2/lexer"
)

/*
program			::= name sensor imports consts vars "begin" statements "end" initialstate
name			::= "program" ident
sensor			::= "sensor" ident
imports			::= /empty/ | import_list
import_list		::= import_stmt | import_stmt import_list
import_stmt		::= "import" "\"" \import_name "\""
import_name		::= ident | ident "." import_name
consts			::= "consts" constsdec
constsdec		::= constdec ";" | constdec ";" constsdec
constdec		::= type ident
vars			::= "vars" varsdec
varsdec			::= vardec ";" | vardec ";" varsdec
vardec			::= access type varlist
varlist			::= variable | variable "," varlist
variable		::= ident source
source			::= /empty/ | "." variable | "." "(" variable ")
access			::= /empty/ | "public" | "private"
type			::= "int" | "string" | "bool"
statements		::= statement | statement "|" statements
statement		::= expr "->" assignments
initialstate	::= /empty/ | "init" "state" assignments
assignments		::= assignment ";" | assignment ";" assignments
assignment		::= var_list "=" expr_list
var_list		::= variable | variable "," var_list
expr_list		::= expr | expr "," expr_list
expr			::= intconst
				 |  string
				 |  "true"
				 | 	"false"
				 |  variable
				 |  unop expr
				 |  expr binop expr
				 |  function_call
				 |  forall
				 |  "(" expr ")"
function_call	::= ident "." ident "(" arg_list ")"
arg_list		::= /empty/ | var_list
forall			::= "forall" ident ":" ident "in" expr ":" expr
unop			::= "!" | "-"
binop			::= "+" | "-" | "*" | "/" | "%"
				 |  "<" | ">" | "==" | "!=" | "<=" | ">="
				 |  "&&" | "||"
*/

type Program struct {
	Pos lexer.Position

	Name                 *string                `"program" @Ident`
	Packages             []string               `("import" @String )*`
	Sensor               *string                `"sensor" @Ident`
	ConstDeclarations    []*ConstDeclaration    `("const" ( @@ ";" )+ )?`
	VariableDeclarations []*VariableDeclaration `"var" @@ ";" ( ( @@ ";" )* )?`
	Statements           []*Statement           `"begin" @@ ( ( "|" @@ )* )? "end"`
	InitialState         []*Assignment          `( "init" "state" ( @@ )+ )?`
}

type ConstDeclaration struct {
	Pos lexer.Position

	VariableType *string     `@("int" | "string" | "bool")`
	Ids          []*Variable `@@ ( ( "," @@ )* )?`
}

type VariableDeclaration struct {
	Pos lexer.Position

	Access       *string     `@("public" | "private")?`
	VariableType *string     `@("int" | "string" | "bool")`
	Ids          []*Variable `@@ ( ( "," @@ )* )?`
}

type Variable struct {
	Pos lexer.Position

	Id     *string         `@Ident`
	Source *VariableSource `("." @@)?`
}

type VariableSource struct {
	Pos lexer.Position

	Source         *string `@Ident`
	VariableId     *string `| ( "(" @Ident`
	VariableSource *string `"." @Ident ")" )?`
}

type Statement struct {
	Pos lexer.Position

	Guard   *Expr     `@@ "-" ">"`
	Actions []*Action `@@ ";" ( ( @@ ";" )* )?`
}

type Action struct {
	Pos lexer.Position

	Variable []*Variable `@@ ("," @@)*`
	Op       *string     `@"="`
	Expr     []*Expr     `@@ ("," @@)*`
}

type Expr struct {
	Pos lexer.Position

	Assignment *Assignment `@@`
}

type Assignment struct {
	Pos lexer.Position

	Equality *Equality `@@`
	Op       string    `( @"="`
	Next     *Equality `  @@ )?`
}

type Equality struct {
	Pos lexer.Position

	Logical *Logical  `@@`
	Op      string    `[ @( "!" "=" | "=" "=" )`
	Next    *Equality `  @@ ]`
}

type Logical struct {
	Pos lexer.Position

	Comparison *Comparison `@@`
	Op         string      `[ @( "&" "&" | "|" "|" )`
	Next       *Logical    `  @@ ]`
}

type Comparison struct {
	Pos lexer.Position

	Addition *Addition   `@@`
	Op       string      `[ @( ">" "=" | ">" | "<" "=" | "<" )`
	Next     *Comparison `  @@ ]`
}

type Addition struct {
	Pos lexer.Position

	Multiplication *Multiplication `@@`
	Op             string          `[ @( "-" | "+" )`
	Next           *Addition       `  @@ ]`
}

type Multiplication struct {
	Pos lexer.Position

	Unary *Unary          `@@`
	Op    string          `[ @( "/" | "*" )`
	Next  *Multiplication `  @@ ]`
}

type Unary struct {
	Pos lexer.Position

	Op      string   `( @( "!" | "-" )`
	Unary   *Unary   `  @@ )`
	Primary *Primary `| @@`
}

type ForAllExpr struct {
	Pos lexer.Position

	// forall k : k in nbrs.j: P.k !=  j

	LoopVariable  *string   `@Ident ":"`
	LoopVariable2 *string   `@Ident "in"`
	LoopOver      *Variable `@@ ":"`
	Expr          *Expr     `@@`
}

type FuncCall struct {
	Pos lexer.Position

	PackageId    *string `@Ident "."`
	FunctionName *string `@Ident`
	Args         []*Expr `"(" (@@ ( "," @@ )* )? ")"`
}

type Primary struct {
	Pos lexer.Position

	NumberValue   int         `@Number`
	StringValue   string      `| @String`
	BoolValue     string      `| @ ("true" | "false")`
	FuncCall      *FuncCall   `| @@`
	ForAll        *ForAllExpr `| "forall" @@`
	Id            *Variable   `| @@`
	SubExpression *Expr       `| "(" @@ ")"`
}
