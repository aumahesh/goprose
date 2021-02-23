package parser

import (
	"os"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/alecthomas/participle/v2/lexer/stateful"
	"github.com/alecthomas/repr"
	log "github.com/sirupsen/logrus"
)

type ProSeParser struct {
	programFile    string
	program        *Program
	parsed         bool
	error          error
	printParseTree bool
	lex            *stateful.Definition
}

func NewProSeParser(programFile string, printParseTree bool) (*ProSeParser, error) {
	lex := stateful.MustSimple([]stateful.Rule{
		{"comment", `(?:#|//)[^\n]*\n?`, nil},
		{"whitespace", `[ \t\n\r]+`, nil},
		{`String`, `"(?:\\.|[^"])*"`, nil},
		{"Punct", `[-[!%&*()+_=\|:;"<,>.?/]|]`, nil},
		{"Number", `[-+]?\d+`, nil},
		{"Ident", `[a-zA-Z_][a-zA-Z0-9_]*`, nil},
	})

	return &ProSeParser{
		programFile:    programFile,
		program:        &Program{},
		lex:            lex,
		parsed:         false,
		error:          nil,
		printParseTree: printParseTree,
	}, nil
}

func (p *ProSeParser) Parse() error {
	parser := participle.MustBuild(p.program,
		participle.Lexer(p.lex),
		participle.Unquote("String"),
		participle.UseLookahead(4),
	)
	r, err := os.Open(p.programFile)
	if err != nil {
		return err
	}
	defer r.Close()
	err = parser.Parse(p.programFile, r, p.program, participle.AllowTrailing(true))

	if err == nil {
		if p.printParseTree {
			repr.Println(p.program, repr.Hide(&lexer.Position{}))
		}
		p.parsed = true
	} else {
		perr := err.(participle.Error)
		log.Errorf("Error: %s", perr)
		p.error = err
	}
	return err
}

func (p *ProSeParser) GetParsedProgram() (*Program, error) {
	if p.error == nil {
		return p.program, nil
	}
	return nil, p.error
}
