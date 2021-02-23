package intermediate

import (
	"fmt"
	"strings"

	"github.com/aumahesh/goprose/internal/parser"
	"github.com/aumahesh/goprose/internal/util"
)

type Expression struct {
	manager     *TempsManager
	constants   map[string]*Variable
	variables   map[string]*Variable
	sensorId    string
	Code        []string
	Temps       []string
	FinalResult string
}

func NewExpression(expr *parser.Expr, sensorId string, constants, variables map[string]*Variable, manager *TempsManager) (*Expression, error) {
	e := &Expression{
		manager:   manager,
		constants: constants,
		variables: variables,
		sensorId:  sensorId,
		Code:      []string{},
	}

	if expr.Assignment == nil {
		return e, nil
	}

	t, err := e.Assignment(expr.Assignment)
	if err != nil {
		return nil, err
	}

	e.FinalResult = t

	return e, nil
}

func (e *Expression) generateBinaryOperationCode(t1, op, t2 string) string {
	newTemp := e.manager.NewTempVariable("unknown")

	code := fmt.Sprintf("%s := %s %s %s", newTemp, t1, op, t2)
	e.Code = append(e.Code, code)
	e.Temps = append(e.Temps, newTemp)

	return newTemp
}

func (e *Expression) generateUnaryOperationCode(t1, op string) string {
	newTemp := e.manager.NewTempVariable("unknown")

	code := fmt.Sprintf("%s := %s %s", newTemp, op, t1)
	e.Code = append(e.Code, code)
	e.Temps = append(e.Temps, newTemp)

	return newTemp
}

func (e *Expression) Assignment(assignment *parser.Assignment) (string, error) {
	if assignment.Equality == nil {
		return "", fmt.Errorf("no rule to process")
	}

	t1, err := e.Equality(assignment.Equality)
	if err != nil {
		return "", err
	}

	if assignment.Next == nil {
		return t1, nil
	}

	t2, err := e.Equality(assignment.Next)
	if err != nil {
		return "", err
	}

	return e.generateBinaryOperationCode(t1, assignment.Op, t2), nil
}

func (e *Expression) Equality(equality *parser.Equality) (string, error) {
	if equality.Logical == nil {
		return "", fmt.Errorf("no rule to process")
	}

	t1, err := e.Logical(equality.Logical)
	if err != nil {
		return "", err
	}

	if equality.Next == nil {
		return t1, nil
	}

	t2, err := e.Equality(equality.Next)
	if err != nil {
		return "", err
	}

	return e.generateBinaryOperationCode(t1, equality.Op, t2), nil
}

func (e *Expression) Logical(logical *parser.Logical) (string, error) {
	if logical.Comparison == nil {
		return "", fmt.Errorf("no rule to process")
	}

	t1, err := e.Comparison(logical.Comparison)
	if err != nil {
		return "", err
	}

	if logical.Next == nil {
		return t1, nil
	}

	t2, err := e.Logical(logical.Next)
	if err != nil {
		return "", err
	}

	return e.generateBinaryOperationCode(t1, logical.Op, t2), nil
}

func (e *Expression) Comparison(comparison *parser.Comparison) (string, error) {
	if comparison.Addition == nil {
		return "", fmt.Errorf("no rule to process")
	}

	t1, err := e.Addition(comparison.Addition)
	if err != nil {
		return "", err
	}

	if comparison.Next == nil {
		return t1, nil
	}

	t2, err := e.Comparison(comparison.Next)
	if err != nil {
		return "", err
	}

	return e.generateBinaryOperationCode(t1, comparison.Op, t2), nil
}

func (e *Expression) Addition(addition *parser.Addition) (string, error) {
	if addition.Multiplication == nil {
		return "", fmt.Errorf("no rule to process")
	}

	t1, err := e.Multiplication(addition.Multiplication)
	if err != nil {
		return "", err
	}

	if addition.Next == nil {
		return t1, nil
	}

	t2, err := e.Addition(addition.Next)
	if err != nil {
		return "", err
	}

	return e.generateBinaryOperationCode(t1, addition.Op, t2), nil
}

func (e *Expression) Multiplication(multiplication *parser.Multiplication) (string, error) {
	if multiplication.Unary == nil {
		return "", fmt.Errorf("no rule to process")
	}

	t1, err := e.Unary(multiplication.Unary)
	if err != nil {
		return "", err
	}

	if multiplication.Next == nil {
		return t1, nil
	}

	t2, err := e.Multiplication(multiplication.Next)
	if err != nil {
		return "", err
	}

	return e.generateBinaryOperationCode(t1, multiplication.Op, t2), nil
}

func (e *Expression) Unary(unary *parser.Unary) (string, error) {
	if unary.Unary != nil {
		t1, err := e.Unary(unary.Unary)
		if err != nil {
			return "", err
		}
		switch unary.Op {
		case "!":
		case "-":
			return e.generateUnaryOperationCode(t1, unary.Op), nil
		default:
			return t1, nil
		}
	}
	if unary.Primary != nil {
		return e.Primary(unary.Primary)
	}

	return "", fmt.Errorf("no rule to process")
}

func (e *Expression) Primary(primary *parser.Primary) (string, error) {
	if primary.NumberValue != nil {
		t1 := e.manager.NewTempVariable("int")
		code := fmt.Sprintf("%s := int64(%d)", t1, Int64Value(primary.NumberValue))
		e.Code = append(e.Code, code)
		e.Temps = append(e.Temps, t1)
		return t1, nil
	}
	if primary.StringValue != nil {
		t1 := e.manager.NewTempVariable("string")
		code := fmt.Sprintf("%s := \"%s\"", t1, StringValue(primary.StringValue))
		e.Code = append(e.Code, code)
		e.Temps = append(e.Temps, t1)
		return t1, nil
	}
	if primary.BoolValue != nil {
		t1 := e.manager.NewTempVariable("bool")
		code := fmt.Sprintf("%s := %s", t1, StringValue(primary.BoolValue))
		e.Code = append(e.Code, code)
		e.Temps = append(e.Temps, t1)
		return t1, nil
	}
	if primary.ForAll != nil {
		return e.ForAll(primary.ForAll)
	}
	if primary.FuncCall != nil {
		return e.FunctionCall(primary.FuncCall)
	}
	if primary.Id != nil {
		return e.VariableAssignment(primary.Id)
	}
	if primary.SubExpression != nil {
		newExpr, err := NewExpression(primary.SubExpression, e.sensorId, e.constants, e.variables, e.manager)
		if err != nil {
			return "", err
		}
		e.Code = append(e.Code, newExpr.Code...)
		e.Temps = append(e.Temps, newExpr.Temps...)
		return newExpr.FinalResult, nil
	}

	return "", fmt.Errorf("no rule to process")
}

func (e *Expression) ForAll(forall *parser.ForAllExpr) (string, error) {
	if forall.LoopVariable == nil || forall.LoopVariable2 == nil {
		return "", fmt.Errorf("no loop variable found")
	}
	l1 := StringValue(forall.LoopVariable)
	l2 := StringValue(forall.LoopVariable2)
	if l1 != l2 {
		return "", fmt.Errorf("loop variable mismatch: %s %s", l1, l2)
	}
	loopOver, err := NewExpression(forall.LoopOver, e.sensorId, e.constants, e.variables, e.manager)
	if err != nil {
		return "", fmt.Errorf("loop over expression error: %s", err)
	}
	loopExpr, err := NewExpression(forall.Expr, e.sensorId, e.constants, e.variables, e.manager)
	if err != nil {
		return "", fmt.Errorf("loop expression error: %s", err)
	}
	e.Code = append(e.Code, loopOver.Code...)
	e.Temps = append(e.Temps, loopOver.Temps...)

	loopCode := []string{}
	for _, c := range loopExpr.Code {
		loopCode = append(loopCode, fmt.Sprintf("\t%s", c))
	}

	loopResultTemp := e.manager.NewTempVariable("unknown")
	code := []string{
		fmt.Sprintf("%s := true", loopResultTemp),
		fmt.Sprintf("for _, %s := range %s {", l1, loopOver.FinalResult),
	}
	code = append(code, loopCode...)
	code = append(code, []string{
		fmt.Sprintf("\tif %s && !%s {", loopResultTemp, loopOver.FinalResult),
		fmt.Sprintf("\t\t%s = false", loopResultTemp),
		fmt.Sprintf("\t}"),
		fmt.Sprintf("}"),
	}...)

	e.Code = append(e.Code, code...)
	e.Temps = append(e.Temps, loopResultTemp)

	return loopResultTemp, nil
}

func (e *Expression) FunctionCall(funcCall *parser.FuncCall) (string, error) {
	if funcCall.PackageId == nil || funcCall.FunctionName == nil {
		return "", fmt.Errorf("function name is missing")
	}

	argsTemp := []string{}

	for index, argExpr := range funcCall.Args {
		expr, err := NewExpression(argExpr, e.sensorId, e.constants, e.variables, e.manager)
		if err != nil {
			return "", fmt.Errorf("Argument #%d in function call %s.%s failed",
				index, funcCall.PackageId, funcCall.FunctionName)
		}
		e.Code = append(e.Code, expr.Code...)
		e.Temps = append(e.Temps, expr.Temps...)
		argsTemp = append(argsTemp, expr.FinalResult)
	}

	funcTemp := e.manager.NewTempVariable("unknown")
	code := fmt.Sprintf("%s := %s.%s(%s)",
		funcTemp, funcCall.PackageId, funcCall.FunctionName, strings.Join(argsTemp, ", "))

	e.Code = append(e.Code, code)
	e.Temps = append(e.Temps, funcTemp)

	return funcTemp, nil
}

func (e *Expression) VariableAssignment(variable *parser.Variable) (string, error) {
	if variable.Id == nil {
		return "", fmt.Errorf("variable id missing")
	}
	vid := StringValue(variable.Id)
	if variable.Source == nil {
		// constant
		c, ok := e.constants[vid]
		if !ok {
			return "", fmt.Errorf("constant %s is not defined", vid)
		}
		vtemp := e.manager.NewTempVariable(GetProseTypeString(c.ProseType))
		code := fmt.Sprintf("%s := %s", vtemp, c.Name)
		e.Code = append(e.Code, code)
		e.Temps = append(e.Temps, vtemp)
		return vtemp, nil
	}
	src := variable.Source
	// constant
	v, ok := e.variables[vid]
	if !ok {
		return "", fmt.Errorf("variable %s is not defined", vid)
	}
	if src.Source != nil {
		// source is from an id (not a variable)
		vsrc := StringValue(src.Source)
		if vsrc == e.sensorId {
			vtemp := e.manager.NewTempVariable(GetProseTypeString(v.ProseType))
			code := fmt.Sprintf("%s := this.state.%s", vtemp, util.ToCamelCase(v.Name))
			e.Code = append(e.Code, code)
			e.Temps = append(e.Temps, vtemp)
			return vtemp, nil
		} else {
			// TODO
			return "", nil
		}
	}
	if src.VariableId != nil {
		vvid := StringValue(src.VariableId)
		vv, ok := e.variables[vvid]
		if !ok {
			return "", fmt.Errorf("variable index %s to %s not defined", vvid, vid)
		}
		vvsrc := StringValue(src.VariableSource)
		if vvsrc == e.sensorId {
			vtemp := e.manager.NewTempVariable("unknown")
			code := []string{
				fmt.Sprintf("var %s %s", vtemp, GetProseTypeString(v.ProseType)),
				fmt.Sprintf(`if this.id == this.state.%s`, util.ToCamelCase(vv.Name)),
				fmt.Sprintf("/t%s = this.state.%s", vtemp, util.ToCamelCase(v.Name)),
				fmt.Sprintf("} else {"),
				fmt.Sprintf("\tfor _, nbr := range this.neighborState {"),
				fmt.Sprintf("\t\tif nbr.id == this.state.%s {", util.ToCamelCase(vv.Name)),
				fmt.Sprintf("\t\t%s = nbr.%s", vtemp, util.ToCamelCase(v.Name)),
				fmt.Sprintf("\t\t\tbreak"),
				fmt.Sprintf("\t\t}"),
				fmt.Sprintf("\t}"),
				fmt.Sprintf("}"),
			}
			e.Code = append(e.Code, code...)
			e.Temps = append(e.Temps, vtemp)
			return vtemp, nil
		} else {
			// TODO
			return "", nil
		}
	}
	return "", fmt.Errorf("invalid code sequence")
}
