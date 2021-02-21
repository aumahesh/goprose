package intermediate

import (
	"github.com/aumahesh/goprose/internal/parser"
)

const (
	defaultOrg       = "aumahesh.com/prose"
	KeywordNeighbors = "nbrs"
)

type translator struct {
	parsedProgram       *parser.Program
	intermediateProgram *Program
	tempsManager        *TempsManager
}

func (g *translator) do() error {
	g.intermediateProgram.Org = defaultOrg
	return nil
}
