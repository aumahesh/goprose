package intermediate

import (
	"fmt"

	"github.com/aumahesh/goprose/internal/parser"
)

type GuardedStatement struct {
	Guard  *Expression
	Action []*Action
	Code   []string
}

func NewStatement(stmt *parser.Statement, sensorId string, constants, variables map[string]*Variable, manager *TempsManager) (*GuardedStatement, error) {
	s := &GuardedStatement{}

	guard, err := NewExpression(stmt.Guard, sensorId, constants, variables, manager)
	if err != nil {
		return nil, err
	}
	s.Guard = guard

	for _, stmtAction := range stmt.Actions {
		action, err := NewAction(stmtAction, sensorId, constants, variables, manager)
		if err != nil {
			return nil, err
		}
		s.Action = append(s.Action, action)
	}

	s.Code = []string{}
	s.Code = append(s.Code, s.Guard.Code...)
	s.Code = append(s.Code, fmt.Sprintf("if %s {", s.Guard.FinalResult))
	for _, act := range s.Action {
		for _, c := range act.Code {
			s.Code = append(s.Code, fmt.Sprintf("\t%s", c))
		}
	}
	s.Code = append(s.Code, fmt.Sprintf("\tstateChanged = true"))
	s.Code = append(s.Code, "}")

	return s, nil
}
