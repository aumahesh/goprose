package intermediate

import (
	"fmt"

	"github.com/aumahesh/goprose/internal/parser"
)

type Program struct {
	Org                string
	ModuleName         string
	PackageName        string
	InterfaceName      string
	ImplementationName string
	Variables          map[string]*Variable
}

func (pv *Program) GetType(key string) (string, error) {
	v, ok := pv.Variables[key]
	if !ok {
		return "", fmt.Errorf("%s not found", key)
	}
	return GetProseTypeString(v.ProseType), nil
}

func (pv *Program) IsType(key string, tgt string) bool {
	t, err := pv.GetType(key)
	if err != nil {
		return false
	}
	return t == tgt
}

func GenerateIntermediateProgram(parsedProgram *parser.Program) (*Program, error) {
	g := &translator{
		parsedProgram:       parsedProgram,
		intermediateProgram: &Program{},
		tempsManager:        NewTempsManager(),
	}
	err := g.do()
	if err != nil {
		return nil, err
	}
	return g.intermediateProgram, nil
}
