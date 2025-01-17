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
	expr          *parser.Expr
	manager       *TempsManager
	constants     map[string]*Variable
	variables     map[string]*Variable
	variableState map[string]bool
	sensorId      string
	Code          []string
	Temps         []string
	FinalResult   string
}

func NewExpression(expr *parser.Expr, sensorId string, constants, variables map[string]*Variable, vs map[string]bool, manager *TempsManager) *Expression {
	e := &Expression{
		expr:          expr,
		manager:       manager,
		constants:     constants,
		variables:     variables,
		variableState: vs,
		sensorId:      sensorId,
		Code:          []string{},
		Temps:         []string{},
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
		expr := NewExpression(primary.SubExpression, e.sensorId, e.constants, e.variables, e.variableState, e.manager)
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
		expr := NewExpression(arg, e.sensorId, e.constants, e.variables, e.variableState, e.manager)
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
	return fmt.Sprintf("(%s %s %s)", t1, op, t2)
}

func (e *Expression) generateUnaryOperationCode(t1, op string) string {
	return fmt.Sprintf("(%s %s)", op, t1)
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

	if comparison.Op == "=>" {
		negateT1 := e.manager.NewTempVariable(GetProseTypeString(ProseTypeBool))
		code := fmt.Sprintf("%s := !%s", negateT1, t1)
		e.Code = append(e.Code, code)
		e.Temps = append(e.Temps, negateT1)
		return e.generateBinaryOperationCode(negateT1, "||", t2), nil
	} else {
		return e.generateBinaryOperationCode(t1, comparison.Op, t2), nil
	}
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
			fallthrough
		case "-":
			return e.generateUnaryOperationCode(t1, unary.Op), nil
		default:
			return t1, nil
		}
	}
	if unary.Primary != nil {
		return e.Primary(unary.Primary)
	}

	return "", fmt.Errorf("unary: no rule to process: %+v", unary)
}

func (e *Expression) Primary(primary *parser.Primary) (string, error) {
	if primary.NumberValue != nil {
		return fmt.Sprintf("int64(%d)", Int64Value(primary.NumberValue)), nil
	}
	if primary.StringValue != nil {
		return fmt.Sprintf(`"%s"`, StringValue(primary.StringValue)), nil
	}
	if primary.BoolValue != nil {
		return StringValue(primary.BoolValue), nil
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
		newExpr := NewExpression(primary.SubExpression, e.sensorId, e.constants, e.variables, e.variableState, e.manager)
		err := newExpr.GenerateCode()
		if err != nil {
			return "", err
		}
		e.Code = append(e.Code, newExpr.Code...)
		e.Temps = append(e.Temps, newExpr.Temps...)
		return newExpr.FinalResult, nil
	}

	return "", fmt.Errorf("primary: no rule to process: %+v", primary)
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
	loopOver := NewExpression(forall.LoopOver, e.sensorId, e.constants, e.variables, e.variableState, e.manager)
	err := loopOver.GenerateCode()
	if err != nil {
		return "", fmt.Errorf("loop over expression error: %s", err)
	}
	loopExpr := NewExpression(forall.Expr, e.sensorId, e.constants, e.variables, e.variableState, e.manager)
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
		fmt.Sprintf("for _, neighbor := range %s {", loopOver.FinalResult),
	}
	code = append(code, loopCode...)
	code = append(code, []string{
		fmt.Sprintf("\tif %s && !%s {", loopResultTemp, loopExpr.FinalResult),
		fmt.Sprintf("\t\t%s = false", loopResultTemp),
		fmt.Sprintf("\t\tbreak"),
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
		expr := NewExpression(argExpr, e.sensorId, e.constants, e.variables, e.variableState, e.manager)
		err := expr.GenerateCode()
		if err != nil {
			return "", fmt.Errorf("Argument #%d in function call %s.%s failed",
				index, StringValue(funcCall.PackageId), StringValue(funcCall.FunctionName))
		}
		e.Code = append(e.Code, expr.Code...)
		e.Temps = append(e.Temps, expr.Temps...)
		argsTemp = append(argsTemp, expr.FinalResult)
	}

	packageId := StringValue(funcCall.PackageId)
	functionName := StringValue(funcCall.FunctionName)

	funcTemp := e.manager.NewTempVariable("unknown")
	fcall := fmt.Sprintf("%s.%s", packageId, functionName)
	// special handling
	if packageId == "time" && functionName == "Now" {
		fcall = "time.Now().Unix"
	}
	if packageId == "neighborhood" {
		switch functionName {
		case "neighbors":
			fcall = "this.neighbors"
		case "up":
			fcall = "this.isNeighborUp"
		case "set":
			fcall = "this.setNeighbor"
		default:
			return "", fmt.Errorf("unknown function in %s package", packageId)
		}
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
		var vtemp string
		// constant
		c, ok := e.constants[vid]
		if !ok {
			// vid is an id
			if vid == e.sensorId {
				vtemp = "this.id"
			} else {
				vtemp = "neighbor.id"
			}
		} else {
			vtemp = c.Name
		}
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
		var vtemp string
		if vsrc == e.sensorId {
			vtemp = fmt.Sprintf("this.state.%s", util.ToCamelCase(v.Name))
		} else {
			vtemp = fmt.Sprintf("neighbor.state.%s", util.ToCamelCase(v.Name))
		}
		return vtemp, nil
	}
	if src.VariableId != nil {
		vvid := StringValue(src.VariableId)
		vv, ok := e.variables[vvid]
		if !ok {
			return "", fmt.Errorf("variable index %s to %s not defined", vvid, vid)
		}
		vvsrc := StringValue(src.VariableSource)
		if vvsrc == e.sensorId {
			vsName := fmt.Sprintf("%s", vv.Name)
			checkAdded, alreadyFound := e.variableState[vsName]
			if !alreadyFound || !checkAdded {
				code := []string{
					fmt.Sprintf(`if neighbor.id != this.state.%s {`, util.ToCamelCase(vv.Name)),
					fmt.Sprintf("\tcontinue"),
					fmt.Sprintf("}"),
				}
				e.Code = append(e.Code, code...)
				e.variableState[vv.Name] = true
			}
			return fmt.Sprintf("neighbor.state.%s", util.ToCamelCase(v.Name)), nil
		} else {
			vtemp := e.manager.NewTempVariable("unknown")
			code := []string{
				fmt.Sprintf("%s, err := this.getNeighbor(neighbor.state.%s)", vtemp, util.ToCamelCase(vv.Name)),
				fmt.Sprintf("if err != nil {"),
				fmt.Sprintf("\tlog.Errorf(\"Error finding neighbor\")"),
				fmt.Sprintf("\treturn false, neighbor"),
				fmt.Sprintf("}"),
			}
			e.Code = append(e.Code, code...)
			e.Temps = append(e.Temps, vtemp)
			e.variableState[vv.Name] = true
			return fmt.Sprintf("%s.state.%s", vtemp, util.ToCamelCase(v.Name)), nil
		}
	}
	return "", fmt.Errorf("invalid code sequence")
}
