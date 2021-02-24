package intermediate

import (
	"fmt"
	"strings"

	"github.com/aumahesh/goprose/internal/parser"
	"github.com/aumahesh/goprose/internal/util"
)

type ExpressionType int

const (
	ExpressionTypeInvalid ExpressionType = iota
	ExpressionTypeLocal
	ExpressionTypeRemote
)

func mergeExpressionTypes(e1, e2 ExpressionType) ExpressionType {
	if e1 == ExpressionTypeInvalid || e2 == ExpressionTypeInvalid {
		return ExpressionTypeInvalid
	}
	if e1 == ExpressionTypeRemote || e2 == ExpressionTypeRemote {
		return ExpressionTypeRemote
	}
	return ExpressionTypeLocal
}

type Expression struct {
	expr        *parser.Expr
	manager     *TempsManager
	constants   map[string]*Variable
	variables   map[string]*Variable
	sensorId    string
	Code        []string
	Temps       []string
	FinalResult string
}

func NewExpression(expr *parser.Expr, sensorId string, constants, variables map[string]*Variable, manager *TempsManager) *Expression {
	e := &Expression{
		expr:      expr,
		manager:   manager,
		constants: constants,
		variables: variables,
		sensorId:  sensorId,
		Code:      []string{},
		Temps:     []string{},
	}

	return e
}

func (e *Expression) GetExpressionType() (ExpressionType, error) {
	if e.expr.Assignment == nil {
		return ExpressionTypeInvalid, nil
	}

	return e.getAssignmentType(e.expr.Assignment)
}

func (e *Expression) getAssignmentType(assignment *parser.Assignment) (ExpressionType, error) {
	b1, err := e.getEqualityType(assignment.Equality)
	if err != nil {
		return ExpressionTypeInvalid, err
	}
	if assignment.Next == nil {
		return b1, err
	}
	b2, err := e.getEqualityType(assignment.Next)
	if err != nil {
		return ExpressionTypeInvalid, err
	}
	return mergeExpressionTypes(b1, b2), nil
}

func (e *Expression) getEqualityType(equality *parser.Equality) (ExpressionType, error) {
	b1, err := e.getLogicalType(equality.Logical)
	if err != nil {
		return ExpressionTypeInvalid, err
	}
	if equality.Next == nil {
		return b1, err
	}
	b2, err := e.getEqualityType(equality.Next)
	if err != nil {
		return ExpressionTypeInvalid, err
	}
	return mergeExpressionTypes(b1, b2), nil
}

func (e *Expression) getLogicalType(logical *parser.Logical) (ExpressionType, error) {
	b1, err := e.getComparisonType(logical.Comparison)
	if err != nil {
		return ExpressionTypeInvalid, err
	}
	if logical.Next == nil {
		return b1, err
	}
	b2, err := e.getLogicalType(logical.Next)
	if err != nil {
		return ExpressionTypeInvalid, err
	}
	return mergeExpressionTypes(b1, b2), nil
}

func (e *Expression) getComparisonType(comparison *parser.Comparison) (ExpressionType, error) {
	b1, err := e.getAdditionType(comparison.Addition)
	if err != nil {
		return ExpressionTypeInvalid, err
	}
	if comparison.Next == nil {
		return b1, err
	}
	b2, err := e.getComparisonType(comparison.Next)
	if err != nil {
		return ExpressionTypeInvalid, err
	}
	return mergeExpressionTypes(b1, b2), nil
}

func (e *Expression) getAdditionType(addition *parser.Addition) (ExpressionType, error) {
	b1, err := e.getMultiplicationType(addition.Multiplication)
	if err != nil {
		return ExpressionTypeInvalid, err
	}
	if addition.Next == nil {
		return b1, err
	}
	b2, err := e.getAdditionType(addition.Next)
	if err != nil {
		return ExpressionTypeInvalid, err
	}
	return mergeExpressionTypes(b1, b2), nil
}

func (e *Expression) getMultiplicationType(multiplication *parser.Multiplication) (ExpressionType, error) {
	b1, err := e.getUnaryType(multiplication.Unary)
	if err != nil {
		return ExpressionTypeInvalid, err
	}
	if multiplication.Next == nil {
		return b1, err
	}
	b2, err := e.getMultiplicationType(multiplication.Next)
	if err != nil {
		return ExpressionTypeInvalid, err
	}
	return mergeExpressionTypes(b1, b2), nil
}

func (e *Expression) getUnaryType(unary *parser.Unary) (ExpressionType, error) {
	if unary.Unary != nil {
		return e.getUnaryType(unary.Unary)
	}
	if unary.Primary != nil {
		return e.getPrimaryType(unary.Primary)
	}
	return ExpressionTypeInvalid, fmt.Errorf("invalid")
}

func (e *Expression) getPrimaryType(primary *parser.Primary) (ExpressionType, error) {
	if primary.NumberValue != nil || primary.StringValue != nil || primary.BoolValue != nil {
		return ExpressionTypeLocal, nil
	}
	if primary.Id != nil {
		return e.getVariableType(primary.Id)
	}
	if primary.FuncCall != nil {
		return e.getFuncCallType(primary.FuncCall)
	}
	if primary.ForAll != nil {
		return e.getForAllType(primary.ForAll)
	}
	if primary.SubExpression != nil {
		expr := NewExpression(primary.SubExpression, e.sensorId, e.constants, e.variables, e.manager)
		return expr.GetExpressionType()
	}
	return ExpressionTypeInvalid, fmt.Errorf("invalid")
}

func (e *Expression) getVariableType(variable *parser.Variable) (ExpressionType, error) {
	if variable.Id == nil {
		return ExpressionTypeInvalid, fmt.Errorf("variable id missing")
	}
	vid := StringValue(variable.Id)
	if variable.Source == nil {
		// constant
		_, ok := e.constants[vid]
		if !ok {
			if vid == e.sensorId {
				return ExpressionTypeLocal, nil
			}
			return ExpressionTypeRemote, nil
		}
		return ExpressionTypeLocal, nil
	}
	src := variable.Source
	// constant
	_, ok := e.variables[vid]
	if !ok {
		return ExpressionTypeInvalid, fmt.Errorf("variable %s is not defined", vid)
	}
	if src.Source != nil {
		// source is from an id (not a variable)
		vsrc := StringValue(src.Source)
		if vsrc == e.sensorId {
			return ExpressionTypeLocal, nil
		} else {
			return ExpressionTypeRemote, nil
		}
	}
	if src.VariableId != nil {
		return ExpressionTypeRemote, nil
	}
	return ExpressionTypeInvalid, fmt.Errorf("invalid code sequence")
}

func (e *Expression) getFuncCallType(funcCall *parser.FuncCall) (ExpressionType, error) {
	var etype ExpressionType = ExpressionTypeLocal
	for _, arg := range funcCall.Args {
		expr := NewExpression(arg, e.sensorId, e.constants, e.variables, e.manager)
		argType, err := expr.GetExpressionType()
		if err != nil {
			return ExpressionTypeInvalid, err
		}
		etype = mergeExpressionTypes(etype, argType)
		if etype == ExpressionTypeInvalid {
			return ExpressionTypeInvalid, fmt.Errorf("invalid type")
		}
		if etype == ExpressionTypeRemote {
			return ExpressionTypeRemote, nil
		}
	}
	return etype, nil
}

func (e *Expression) getForAllType(forall *parser.ForAllExpr) (ExpressionType, error) {
	return ExpressionTypeLocal, nil
}

func (e *Expression) GenerateCode() error {
	if e.expr.Assignment == nil {
		return nil
	}

	t, err := e.Assignment(e.expr.Assignment)
	if err != nil {
		return err
	}

	e.FinalResult = t

	return nil
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
		newExpr := NewExpression(primary.SubExpression, e.sensorId, e.constants, e.variables, e.manager)
		err := newExpr.GenerateCode()
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
	loopOver := NewExpression(forall.LoopOver, e.sensorId, e.constants, e.variables, e.manager)
	err := loopOver.GenerateCode()
	if err != nil {
		return "", fmt.Errorf("loop over expression error: %s", err)
	}
	loopExpr := NewExpression(forall.Expr, e.sensorId, e.constants, e.variables, e.manager)
	err = loopExpr.GenerateCode()
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
		expr := NewExpression(argExpr, e.sensorId, e.constants, e.variables, e.manager)
		err := expr.GenerateCode()
		if err != nil {
			return "", fmt.Errorf("Argument #%d in function call %s.%s failed",
				index, StringValue(funcCall.PackageId), StringValue(funcCall.FunctionName))
		}
		e.Code = append(e.Code, expr.Code...)
		e.Temps = append(e.Temps, expr.Temps...)
		argsTemp = append(argsTemp, expr.FinalResult)
	}

	funcTemp := e.manager.NewTempVariable("unknown")
	fcall := fmt.Sprintf("%s.%s", StringValue(funcCall.PackageId), StringValue(funcCall.FunctionName))
	// special handling
	if StringValue(funcCall.PackageId) == "time" && StringValue(funcCall.FunctionName) == "Now" {
		fcall = "time.Now().Unix"
	}
	code := fmt.Sprintf("%s := %s(%s)",
		funcTemp, fcall, strings.Join(argsTemp, ", "))

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
			// vid is an id
			if vid == e.sensorId {
				vtemp := e.manager.NewTempVariable(GetProseTypeString(ProseTypeString))
				code := fmt.Sprintf("%s := this.id", vtemp)
				e.Code = append(e.Code, code)
				e.Temps = append(e.Temps, vtemp)
				return vtemp, nil
			} else {
				vtemp := e.manager.NewTempVariable(GetProseTypeString(ProseTypeString))
				code := fmt.Sprintf("%s := neighbor.id", vtemp)
				e.Code = append(e.Code, code)
				e.Temps = append(e.Temps, vtemp)
				return vtemp, nil
			}
		} else {
			vtemp := e.manager.NewTempVariable(GetProseTypeString(c.ProseType))
			code := fmt.Sprintf("%s := %s", vtemp, c.Name)
			e.Code = append(e.Code, code)
			e.Temps = append(e.Temps, vtemp)
			return vtemp, nil
		}
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
			vtemp := e.manager.NewTempVariable(GetProseTypeString(v.ProseType))
			code := fmt.Sprintf("%s := neighbor.state.%s", vtemp, util.ToCamelCase(v.Name))
			e.Code = append(e.Code, code)
			e.Temps = append(e.Temps, vtemp)
			return vtemp, nil
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
