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
	Variables          map[string]string
	InitialState       map[string]interface{}
}

func (pv *Program) GetType(key string) (string, error) {
	v, ok := pv.Variables[key]
	if !ok {
		return "", fmt.Errorf("%s not found", key)
	}
	return v, nil
}

func (pv *Program) IsType(key string, tgt string) bool {
	t, err := pv.GetType(key)
	if err != nil {
		return false
	}
	return t == tgt
}

func GenerateIntermediateProgram(parsedProgram *parser.Program) (*Program, error) {
	return nil, fmt.Errorf("not implemented")
}
