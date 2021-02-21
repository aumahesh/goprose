package intermediate

import (
	"github.com/aumahesh/goprose/internal/parser"
)

type translator struct {
	parsedProgram       *parser.Program
	intermediateProgram *Program
	tempsManager        *TempsManager
}

func (g *translator) do() error {
	return nil
}
