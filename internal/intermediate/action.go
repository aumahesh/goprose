package intermediate

import (
	"fmt"

	"github.com/aumahesh/goprose/internal/parser"
	"github.com/aumahesh/goprose/internal/util"
)

type Action struct {
	manager       *TempsManager
	constants     map[string]*Variable
	variables     map[string]*Variable
	variableState map[string]bool
	sensorId      string
	Code          []string
}

// NewAction creates an Action from a parser.Action (regular assignment).
func NewAction(act *parser.Action, sensorId string, constants, variables map[string]*Variable, vs map[string]bool, manager *TempsManager) (*Action, error) {
	a := &Action{
		manager:       manager,
		constants:     constants,
		variables:     variables,
		variableState: vs,
		sensorId:      sensorId,
		Code:          []string{},
	}
	code, err := generateAssignmentCode(act, sensorId, constants, variables, vs, manager)
	if err != nil {
		return nil, err
	}
	a.Code = code
	return a, nil
}

// NewGclAction creates an Action from a parser.GclAction (assignment, alternative, or repetitive construct).
func NewGclAction(act *parser.GclAction, sensorId string, constants, variables map[string]*Variable, vs map[string]bool, manager *TempsManager) (*Action, error) {
	a := &Action{
		manager:       manager,
		constants:     constants,
		variables:     variables,
		variableState: vs,
		sensorId:      sensorId,
		Code:          []string{},
	}
	code, err := generateGclActionCode(act, sensorId, constants, variables, vs, manager)
	if err != nil {
		return nil, err
	}
	a.Code = code
	return a, nil
}

func generateGclActionCode(act *parser.GclAction, sensorId string, constants, variables map[string]*Variable, vs map[string]bool, manager *TempsManager) ([]string, error) {
	if act.Assignment != nil {
		return generateAssignmentCode(act.Assignment, sensorId, constants, variables, vs, manager)
	}
	if act.AlternativeConstruct != nil {
		return generateAlternativeConstructCode(act.AlternativeConstruct, sensorId, constants, variables, vs, manager)
	}
	if act.RepetitiveConstruct != nil {
		return generateRepetitiveConstructCode(act.RepetitiveConstruct, sensorId, constants, variables, vs, manager)
	}
	return nil, fmt.Errorf("empty GCL action")
}

func generateAssignmentCode(act *parser.Action, sensorId string, constants, variables map[string]*Variable, vs map[string]bool, manager *TempsManager) ([]string, error) {
	code := []string{}

	if len(act.Variable) != len(act.Expr) {
		return nil, fmt.Errorf("Invalid assignment, %d on LHS and %d on RHS",
			len(act.Variable), len(act.Expr))
	}

	for index, actExpr := range act.Expr {
		aexpr := NewExpression(actExpr, sensorId, constants, variables, vs, manager)
		err := aexpr.GenerateCode()
		if err != nil {
			return nil, fmt.Errorf("invalid expression @ index %d: %s", index, err)
		}

		code = append(code, aexpr.Code...)

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

		code = append(code,
			fmt.Sprintf("this.state.%s = %s", util.ToCamelCase(v.Name), aexpr.FinalResult))
	}

	return code, nil
}

// generateAlternativeConstructCode generates Go code for Dijkstra's if...fi construct.
// It evaluates all guards, then non-deterministically executes one of the true-guarded commands.
// If no guard is true the construct is a no-op (undefined in strict GCL; we skip silently).
func generateAlternativeConstructCode(ac *parser.AlternativeConstruct, sensorId string, constants, variables map[string]*Variable, vs map[string]bool, manager *TempsManager) ([]string, error) {
	choicesVar := manager.NewTempVariable("[]int")
	chosenVar := manager.NewTempVariable("int")

	code := []string{fmt.Sprintf("var %s []int", choicesVar)}

	for i, cmd := range ac.Commands {
		guardExpr := NewExpression(cmd.Guard, sensorId, constants, variables, vs, manager)
		etype, err := guardExpr.GetExpressionType()
		if err != nil {
			return nil, fmt.Errorf("alternative construct guard %d: %s", i, err)
		}
		if etype == ExpressionTypeRemote {
			return nil, fmt.Errorf("remote variables not supported in alternative construct guard %d", i)
		}
		err = guardExpr.GenerateCode()
		if err != nil {
			return nil, fmt.Errorf("alternative construct guard %d code generation: %s", i, err)
		}
		code = append(code, guardExpr.Code...)
		code = append(code, fmt.Sprintf("if %s {", guardExpr.FinalResult))
		code = append(code, fmt.Sprintf("\t%s = append(%s, %d)", choicesVar, choicesVar, i))
		code = append(code, "}")
	}

	code = append(code, fmt.Sprintf("if len(%s) > 0 {", choicesVar))
	code = append(code, fmt.Sprintf("\t%s := %s[rand.Intn(len(%s))]", chosenVar, choicesVar, choicesVar))
	code = append(code, fmt.Sprintf("\tswitch %s {", chosenVar))

	for i, cmd := range ac.Commands {
		code = append(code, fmt.Sprintf("\tcase %d:", i))
		for _, act := range cmd.Actions {
			actCode, err := generateGclActionCode(act, sensorId, constants, variables, vs, manager)
			if err != nil {
				return nil, fmt.Errorf("alternative construct command %d action: %s", i, err)
			}
			for _, line := range actCode {
				code = append(code, fmt.Sprintf("\t\t%s", line))
			}
		}
	}

	code = append(code, "\t}")
	code = append(code, "}")

	return code, nil
}

// generateRepetitiveConstructCode generates Go code for Dijkstra's do...od construct.
// It loops, each iteration non-deterministically executing one true-guarded command,
// until no guard is true.
func generateRepetitiveConstructCode(rc *parser.RepetitiveConstruct, sensorId string, constants, variables map[string]*Variable, vs map[string]bool, manager *TempsManager) ([]string, error) {
	choicesVar := manager.NewTempVariable("[]int")
	chosenVar := manager.NewTempVariable("int")

	code := []string{"for {"}
	code = append(code, fmt.Sprintf("\tvar %s []int", choicesVar))

	for i, cmd := range rc.Commands {
		guardExpr := NewExpression(cmd.Guard, sensorId, constants, variables, vs, manager)
		etype, err := guardExpr.GetExpressionType()
		if err != nil {
			return nil, fmt.Errorf("repetitive construct guard %d: %s", i, err)
		}
		if etype == ExpressionTypeRemote {
			return nil, fmt.Errorf("remote variables not supported in repetitive construct guard %d", i)
		}
		err = guardExpr.GenerateCode()
		if err != nil {
			return nil, fmt.Errorf("repetitive construct guard %d code generation: %s", i, err)
		}
		for _, line := range guardExpr.Code {
			code = append(code, fmt.Sprintf("\t%s", line))
		}
		code = append(code, fmt.Sprintf("\tif %s {", guardExpr.FinalResult))
		code = append(code, fmt.Sprintf("\t\t%s = append(%s, %d)", choicesVar, choicesVar, i))
		code = append(code, "\t}")
	}

	code = append(code, fmt.Sprintf("\tif len(%s) == 0 {", choicesVar))
	code = append(code, "\t\tbreak")
	code = append(code, "\t}")

	code = append(code, fmt.Sprintf("\t%s := %s[rand.Intn(len(%s))]", chosenVar, choicesVar, choicesVar))
	code = append(code, fmt.Sprintf("\tswitch %s {", chosenVar))

	for i, cmd := range rc.Commands {
		code = append(code, fmt.Sprintf("\tcase %d:", i))
		for _, act := range cmd.Actions {
			actCode, err := generateGclActionCode(act, sensorId, constants, variables, vs, manager)
			if err != nil {
				return nil, fmt.Errorf("repetitive construct command %d action: %s", i, err)
			}
			for _, line := range actCode {
				code = append(code, fmt.Sprintf("\t\t%s", line))
			}
		}
	}

	code = append(code, "\t}")
	code = append(code, "}")

	return code, nil
}
