package intermediate

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/aumahesh/goprose/internal/parser"
)

type GuardedStatement struct {
	Guard  *Expression
	Action []*Action
	Code   []string
}

func NewStatement(stmt *parser.Statement, sensorId string, constants, variables map[string]*Variable, manager *TempsManager) (*GuardedStatement, error) {
	s := &GuardedStatement{}

	s.Guard = NewExpression(stmt.Guard, sensorId, constants, variables, map[string]bool{}, manager)
	etype, err := s.Guard.GetExpressionType()
	if err != nil {
		return nil, err
	}
	if etype == ExpressionTypeInvalid {
		return nil, fmt.Errorf("invalid expression @ %s:", stmt.Guard.Pos)
	}

	if etype == ExpressionTypeRemote {
		s.Code = []string{
			"var found bool",
			"var neighbor *NeighborState",
			"for _, neighbor = range this.neighborState {",
		}
	}

	err = s.Guard.GenerateCode()
	if err != nil {
		log.Debugf("Expression Type: %d", etype)
		log.Debugf("Code so far: %+v", s.Guard.Code)
		return nil, err
	}

	for _, stmtAction := range stmt.Actions {
		action, err := NewAction(stmtAction, sensorId, constants, variables, manager)
		if err != nil {
			return nil, err
		}
		s.Action = append(s.Action, action)
	}

	codelines := []string{}
	if etype == ExpressionTypeRemote {
		for _, c := range s.Guard.Code {
			codelines = append(codelines, fmt.Sprintf("\t%s", c))
		}
		codelines = append(codelines, []string{
			fmt.Sprintf("\tif %s {", s.Guard.FinalResult),
			fmt.Sprintf("\t\tfound = true"),
			fmt.Sprintf("\t\tbreak"),
			fmt.Sprintf("\t}"),
			"}",
			"if found {",
		}...)
	} else {
		codelines = s.Guard.Code
		codelines = append(codelines, fmt.Sprintf("if %s {", s.Guard.FinalResult))
	}
	s.Code = append(s.Code, codelines...)
	for _, act := range s.Action {
		for _, c := range act.Code {
			s.Code = append(s.Code, fmt.Sprintf("\t%s", c))
		}
	}
	s.Code = append(s.Code, fmt.Sprintf("\tstateChanged = true"))
	s.Code = append(s.Code, "}")

	return s, nil
}
