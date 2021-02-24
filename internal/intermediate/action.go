package intermediate

import (
	"fmt"

	"github.com/aumahesh/goprose/internal/parser"
	"github.com/aumahesh/goprose/internal/util"
)

type Action struct {
	manager   *TempsManager
	constants map[string]*Variable
	variables map[string]*Variable
	sensorId  string
	Code      []string
}

func NewAction(act *parser.Action, sensorId string, constants, variables map[string]*Variable, manager *TempsManager) (*Action, error) {
	a := &Action{
		manager:   manager,
		constants: constants,
		variables: variables,
		sensorId:  sensorId,
		Code:      []string{},
	}

	if len(act.Variable) != len(act.Expr) {
		return nil, fmt.Errorf("Invalid assignment, %d on LHS and %d on RHS",
			len(act.Variable), len(act.Expr))
	}

	for index, actExpr := range act.Expr {
		aexpr := NewExpression(actExpr, sensorId, constants, variables, manager)
		err := aexpr.GenerateCode()
		if err != nil {
			return nil, fmt.Errorf("invalid expression @ index %d: %s", index, err)
		}

		a.Code = append(a.Code, aexpr.Code...)

		actVariable := act.Variable[index]
		vid := StringValue(actVariable.Id)
		if actVariable.Source != nil {
			if actVariable.Source.Source == nil {
				return nil, fmt.Errorf("cannot assign variables indexing over other actVariable")
			}
		}
		vsrc := StringValue(actVariable.Source.Source)
		if vsrc != sensorId {
			return nil, fmt.Errorf("cannot assign variables at other sensors %s %s", vid, vsrc)
		}

		v, ok := variables[vid]
		if !ok {
			return nil, fmt.Errorf("unknown variable %s", vid)
		}

		a.Code = append(a.Code,
			fmt.Sprintf("this.state.%s = %s", util.ToCamelCase(v.Name), aexpr.FinalResult))
	}

	return a, nil
}
