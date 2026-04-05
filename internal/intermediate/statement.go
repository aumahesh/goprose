package intermediate

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/aumahesh/goprose/internal/parser"
)

type GuardedStatement struct {
	Priority   int64
	GuardName  string
	ActionName string
	Guard      *Expression
	Action     []*Action
	GuardCode  []string
	ActionCode []string
}

func NewStatement(index int, stmt *parser.Statement, sensorId string, constants, variables map[string]*Variable, manager *TempsManager) (*GuardedStatement, error) {
	s := &GuardedStatement{
		Priority:   1,
		GuardName:  fmt.Sprintf("evaluateGuard%d", index),
		ActionName: fmt.Sprintf("executeAction%d", index),
	}

	if stmt.Alternative != nil {
		return newFromAlternativeConstruct(s, stmt.Alternative, sensorId, constants, variables, manager)
	}
	if stmt.Repetitive != nil {
		return newFromRepetitiveConstruct(s, stmt.Repetitive, sensorId, constants, variables, manager)
	}
	if stmt.Guarded != nil {
		return newFromGuardedProSeStatement(s, stmt.Guarded, sensorId, constants, variables, manager)
	}
	return nil, fmt.Errorf("empty statement at %s", stmt.Pos)
}

// newFromAlternativeConstruct generates a guarded statement for a top-level if...fi.
// The outer guard always fires; the GCL construct handles internal guard selection.
func newFromAlternativeConstruct(s *GuardedStatement, ac *parser.AlternativeConstruct, sensorId string, constants, variables map[string]*Variable, manager *TempsManager) (*GuardedStatement, error) {
	s.GuardCode = []string{"takeAction = true"}

	actionCode, err := generateAlternativeConstructCode(ac, sensorId, constants, variables, map[string]bool{}, manager)
	if err != nil {
		return nil, err
	}
	s.ActionCode = actionCode
	return s, nil
}

// newFromRepetitiveConstruct generates a guarded statement for a top-level do...od.
// The outer guard always fires; the loop terminates internally when no guard is true.
func newFromRepetitiveConstruct(s *GuardedStatement, rc *parser.RepetitiveConstruct, sensorId string, constants, variables map[string]*Variable, manager *TempsManager) (*GuardedStatement, error) {
	s.GuardCode = []string{"takeAction = true"}

	actionCode, err := generateRepetitiveConstructCode(rc, sensorId, constants, variables, map[string]bool{}, manager)
	if err != nil {
		return nil, err
	}
	s.ActionCode = actionCode
	return s, nil
}

// newFromGuardedProSeStatement handles the original ProSe guarded statement form.
func newFromGuardedProSeStatement(s *GuardedStatement, stmt *parser.GuardedProSeStatement, sensorId string, constants, variables map[string]*Variable, manager *TempsManager) (*GuardedStatement, error) {
	if stmt.Priority != nil {
		s.Priority = Int64Value(stmt.Priority)
	}

	s.Guard = NewExpression(stmt.Guard, sensorId, constants, variables, map[string]bool{}, manager)
	etype, err := s.Guard.GetExpressionType()
	if err != nil {
		return nil, err
	}
	if etype == ExpressionTypeInvalid {
		return nil, fmt.Errorf("invalid expression @ %s:", stmt.Guard.Pos)
	}

	if etype == ExpressionTypeRemote {
		s.GuardCode = []string{
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
		action, err := NewGclAction(stmtAction, sensorId, constants, variables, s.Guard.variableState, manager)
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
			fmt.Sprintf("\t\ttakeAction = true"),
			fmt.Sprintf("\t\tbreak"),
			fmt.Sprintf("\t}"),
			"}",
		}...)
	} else {
		codelines = s.Guard.Code
		codelines = append(codelines, []string{
			fmt.Sprintf("if %s {", s.Guard.FinalResult),
			"\ttakeAction = true",
			"}",
		}...)
	}
	s.GuardCode = append(s.GuardCode, codelines...)

	s.ActionCode = []string{}
	if etype == ExpressionTypeRemote {
		s.ActionCode = append(s.ActionCode, []string{
			"if neighbor == nil {",
			"\tlog.Errorf(\"invalid neighbor, nil received\")",
			"\treturn false, nil",
			"}",
		}...)
	}
	for _, act := range s.Action {
		s.ActionCode = append(s.ActionCode, act.Code...)
	}

	return s, nil
}
