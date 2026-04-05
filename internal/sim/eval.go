package sim

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/aumahesh/goprose/internal/parser"
)

// EvalCtx holds the context needed to evaluate a ProSe expression at runtime.
type EvalCtx struct {
	LocalState     NodeState            // this node's variables
	Neighbor       *NeighborInfo        // currently-considered neighbor (nil = local only)
	NeighborStates map[string]NodeState // all neighbors by ID, for inv.(p.j) access
	SensorID       string               // prose sensor name, e.g. "j"
	NodeID         string               // actual runtime node ID, e.g. "n1"
	NeighborID     string               // runtime neighbor ID (when Neighbor != nil)
	Constants      NodeState            // constant values
	AllNeighbors   []*NeighborInfo      // for forall
	RNG            *rand.Rand
}

// NeighborInfo holds a neighbor's ID and its last-known state.
type NeighborInfo struct {
	ID    string
	State NodeState
}

// neighborCollection is the sentinel return type for neighborhood.neighbors().
type neighborCollection []*NeighborInfo

// EvalExpr evaluates a ProSe expression and returns a Go value (int64, bool, or string).
func EvalExpr(expr *parser.Expr, ctx *EvalCtx) (interface{}, error) {
	if expr == nil || expr.Assignment == nil {
		return nil, fmt.Errorf("nil expression")
	}
	return evalAssignment(expr.Assignment, ctx)
}

func evalAssignment(a *parser.Assignment, ctx *EvalCtx) (interface{}, error) {
	t1, err := evalEquality(a.Equality, ctx)
	if err != nil {
		return nil, err
	}
	if a.Next == nil {
		return t1, nil
	}
	t2, err := evalEquality(a.Next, ctx)
	if err != nil {
		return nil, err
	}
	return valEquals(t1, t2), nil
}

func evalEquality(eq *parser.Equality, ctx *EvalCtx) (interface{}, error) {
	t1, err := evalLogical(eq.Logical, ctx)
	if err != nil {
		return nil, err
	}
	if eq.Next == nil {
		return t1, nil
	}
	t2, err := evalEquality(eq.Next, ctx)
	if err != nil {
		return nil, err
	}
	switch eq.Op {
	case "==":
		return valEquals(t1, t2), nil
	case "!=":
		return !valEquals(t1, t2), nil
	}
	return nil, fmt.Errorf("unknown equality op: %s", eq.Op)
}

func evalLogical(l *parser.Logical, ctx *EvalCtx) (interface{}, error) {
	t1, err := evalComparison(l.Comparison, ctx)
	if err != nil {
		return nil, err
	}
	if l.Next == nil {
		return t1, nil
	}
	// Short-circuit evaluation
	b1, err := toBool(t1)
	if err != nil {
		return nil, fmt.Errorf("logical op %s requires bool operands", l.Op)
	}
	if l.Op == "&&" && !b1 {
		return false, nil
	}
	if l.Op == "||" && b1 {
		return true, nil
	}
	t2, err := evalLogical(l.Next, ctx)
	if err != nil {
		return nil, err
	}
	b2, err := toBool(t2)
	if err != nil {
		return nil, fmt.Errorf("logical op %s requires bool operands", l.Op)
	}
	switch l.Op {
	case "&&":
		return b1 && b2, nil
	case "||":
		return b1 || b2, nil
	}
	return nil, fmt.Errorf("unknown logical op: %s", l.Op)
}

func evalComparison(c *parser.Comparison, ctx *EvalCtx) (interface{}, error) {
	t1, err := evalAddition(c.Addition, ctx)
	if err != nil {
		return nil, err
	}
	if c.Next == nil {
		return t1, nil
	}
	t2, err := evalComparison(c.Next, ctx)
	if err != nil {
		return nil, err
	}
	switch c.Op {
	case "=>":
		b1, e1 := toBool(t1)
		b2, e2 := toBool(t2)
		if e1 != nil || e2 != nil {
			return nil, fmt.Errorf("=> requires bool operands")
		}
		return !b1 || b2, nil
	case "<":
		i1, e1 := toInt64(t1)
		i2, e2 := toInt64(t2)
		if e1 != nil || e2 != nil {
			s1, s2 := fmt.Sprintf("%v", t1), fmt.Sprintf("%v", t2)
			return s1 < s2, nil
		}
		return i1 < i2, nil
	case ">":
		i1, e1 := toInt64(t1)
		i2, e2 := toInt64(t2)
		if e1 != nil || e2 != nil {
			s1, s2 := fmt.Sprintf("%v", t1), fmt.Sprintf("%v", t2)
			return s1 > s2, nil
		}
		return i1 > i2, nil
	case "<=":
		i1, _ := toInt64(t1)
		i2, _ := toInt64(t2)
		return i1 <= i2, nil
	case ">=":
		i1, _ := toInt64(t1)
		i2, _ := toInt64(t2)
		return i1 >= i2, nil
	}
	return nil, fmt.Errorf("unknown comparison op: %s", c.Op)
}

func evalAddition(a *parser.Addition, ctx *EvalCtx) (interface{}, error) {
	t1, err := evalMultiplication(a.Multiplication, ctx)
	if err != nil {
		return nil, err
	}
	if a.Next == nil {
		return t1, nil
	}
	t2, err := evalAddition(a.Next, ctx)
	if err != nil {
		return nil, err
	}
	i1, e1 := toInt64(t1)
	i2, e2 := toInt64(t2)
	if e1 != nil || e2 != nil {
		if a.Op == "+" {
			return fmt.Sprintf("%v", t1) + fmt.Sprintf("%v", t2), nil
		}
		return nil, fmt.Errorf("arithmetic %s requires numeric operands", a.Op)
	}
	switch a.Op {
	case "+":
		return i1 + i2, nil
	case "-":
		return i1 - i2, nil
	}
	return nil, fmt.Errorf("unknown addition op: %s", a.Op)
}

func evalMultiplication(m *parser.Multiplication, ctx *EvalCtx) (interface{}, error) {
	t1, err := evalUnary(m.Unary, ctx)
	if err != nil {
		return nil, err
	}
	if m.Next == nil {
		return t1, nil
	}
	t2, err := evalMultiplication(m.Next, ctx)
	if err != nil {
		return nil, err
	}
	i1, _ := toInt64(t1)
	i2, _ := toInt64(t2)
	switch m.Op {
	case "*":
		return i1 * i2, nil
	case "/":
		if i2 == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return i1 / i2, nil
	case "%":
		if i2 == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		return i1 % i2, nil
	}
	return nil, fmt.Errorf("unknown multiplication op: %s", m.Op)
}

func evalUnary(u *parser.Unary, ctx *EvalCtx) (interface{}, error) {
	if u.Unary != nil {
		inner, err := evalUnary(u.Unary, ctx)
		if err != nil {
			return nil, err
		}
		switch u.Op {
		case "!":
			b, err := toBool(inner)
			if err != nil {
				return nil, err
			}
			return !b, nil
		case "-":
			i, err := toInt64(inner)
			if err != nil {
				return nil, err
			}
			return -i, nil
		}
	}
	if u.Primary != nil {
		return evalPrimary(u.Primary, ctx)
	}
	return nil, fmt.Errorf("invalid unary expression")
}

func evalPrimary(p *parser.Primary, ctx *EvalCtx) (interface{}, error) {
	if p.NumberValue != nil {
		return *p.NumberValue, nil
	}
	if p.StringValue != nil {
		// Strip surrounding quotes from the parsed string value
		s := *p.StringValue
		if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
			s = s[1 : len(s)-1]
		}
		return s, nil
	}
	if p.BoolValue != nil {
		return *p.BoolValue == "true", nil
	}
	if p.FuncCall != nil {
		return evalFuncCall(p.FuncCall, ctx)
	}
	if p.ForAll != nil {
		return evalForAll(p.ForAll, ctx)
	}
	if p.Id != nil {
		return evalVariable(p.Id, ctx)
	}
	if p.SubExpression != nil {
		return EvalExpr(p.SubExpression, ctx)
	}
	return nil, fmt.Errorf("empty primary expression")
}

func evalVariable(v *parser.Variable, ctx *EvalCtx) (interface{}, error) {
	id := *v.Id

	if v.Source == nil {
		// Bare identifier: sensor ID, constant, or neighbor-ID reference
		if id == ctx.SensorID {
			return ctx.NodeID, nil
		}
		if val, ok := ctx.Constants[id]; ok {
			return val, nil
		}
		// Treat as the neighbor's ID in a neighbor context
		return ctx.NeighborID, nil
	}

	src := v.Source

	if src.Source != nil {
		srcName := *src.Source
		if srcName == ctx.SensorID {
			val, ok := ctx.LocalState[id]
			if !ok {
				return nil, fmt.Errorf("variable %s not in local state", id)
			}
			return val, nil
		}
		// Remote: X.k
		if ctx.Neighbor == nil {
			return nil, fmt.Errorf("remote variable %s.%s requires neighbor context", id, srcName)
		}
		val, ok := ctx.Neighbor.State[id]
		if !ok {
			return nil, fmt.Errorf("variable %s not in neighbor state", id)
		}
		return val, nil
	}

	if src.VariableId != nil {
		// inv.(p.j) — indexed access via another variable
		idxVar := *src.VariableId
		idxSrc := *src.VariableSource

		var nbrID string
		if idxSrc == ctx.SensorID {
			val, ok := ctx.LocalState[idxVar]
			if !ok {
				return nil, fmt.Errorf("index variable %s not in local state", idxVar)
			}
			var ok2 bool
			nbrID, ok2 = val.(string)
			if !ok2 {
				nbrID = fmt.Sprintf("%v", val)
			}
		} else {
			if ctx.Neighbor == nil {
				return nil, fmt.Errorf("index variable %s.%s requires neighbor", idxVar, idxSrc)
			}
			val, ok := ctx.Neighbor.State[idxVar]
			if !ok {
				return nil, fmt.Errorf("index variable %s not in neighbor state", idxVar)
			}
			var ok2 bool
			nbrID, ok2 = val.(string)
			if !ok2 {
				nbrID = fmt.Sprintf("%v", val)
			}
		}

		if nbrID == "" {
			// Empty pointer — return zero value (nil) rather than error
			return nil, nil
		}
		nbrState, ok := ctx.NeighborStates[nbrID]
		if !ok {
			return nil, fmt.Errorf("neighbor %q not found for indexed access of %s", nbrID, id)
		}
		val, ok := nbrState[id]
		if !ok {
			return nil, fmt.Errorf("variable %s not in neighbor %s state", id, nbrID)
		}
		return val, nil
	}

	return nil, fmt.Errorf("invalid variable access: %s", id)
}

func evalFuncCall(fc *parser.FuncCall, ctx *EvalCtx) (interface{}, error) {
	pkg := *fc.PackageId
	fn := *fc.FunctionName

	args := make([]interface{}, len(fc.Args))
	for i, arg := range fc.Args {
		val, err := EvalExpr(arg, ctx)
		if err != nil {
			return nil, fmt.Errorf("arg %d of %s.%s: %w", i, pkg, fn, err)
		}
		args[i] = val
	}

	switch pkg {
	case "neighborhood":
		switch fn {
		case "neighbors":
			return neighborCollection(ctx.AllNeighbors), nil
		case "up":
			// All neighbors are always up in the simulator
			return true, nil
		case "set":
			return true, nil
		}
	case "rand":
		switch fn {
		case "Int63n", "Intn":
			if len(args) == 1 {
				n, _ := toInt64(args[0])
				if n <= 0 {
					n = 1
				}
				if ctx.RNG != nil {
					return ctx.RNG.Int63n(n), nil
				}
				return rand.Int63n(n), nil
			}
		}
	case "math":
		switch fn {
		case "Abs":
			if len(args) == 1 {
				i, _ := toInt64(args[0])
				if i < 0 {
					return -i, nil
				}
				return i, nil
			}
		}
	case "time":
		switch fn {
		case "Now":
			return time.Now().Unix(), nil
		}
	}

	return nil, fmt.Errorf("unsupported function %s.%s", pkg, fn)
}

func evalForAll(fa *parser.ForAllExpr, ctx *EvalCtx) (interface{}, error) {
	loopOverVal, err := EvalExpr(fa.LoopOver, ctx)
	if err != nil {
		return nil, fmt.Errorf("forall loopOver: %w", err)
	}

	var neighbors []*NeighborInfo
	switch lv := loopOverVal.(type) {
	case neighborCollection:
		neighbors = []*NeighborInfo(lv)
	default:
		neighbors = ctx.AllNeighbors
	}

	for _, nbr := range neighbors {
		innerCtx := *ctx
		innerCtx.Neighbor = nbr
		innerCtx.NeighborID = nbr.ID
		val, err := EvalExpr(fa.Expr, &innerCtx)
		if err != nil {
			// Treat eval error as false for robustness
			return false, nil
		}
		b, err := toBool(val)
		if err != nil {
			return nil, fmt.Errorf("forall body must yield bool: %w", err)
		}
		if !b {
			return false, nil
		}
	}
	return true, nil
}

// --- type helpers ---

func toInt64(v interface{}) (int64, error) {
	switch x := v.(type) {
	case int64:
		return x, nil
	case int:
		return int64(x), nil
	case int32:
		return int64(x), nil
	case float64:
		return int64(x), nil
	case bool:
		if x {
			return 1, nil
		}
		return 0, nil
	}
	return 0, fmt.Errorf("cannot convert %T to int64", v)
}

func toBool(v interface{}) (bool, error) {
	if b, ok := v.(bool); ok {
		return b, nil
	}
	return false, fmt.Errorf("cannot convert %T to bool", v)
}

func valEquals(a, b interface{}) bool {
	switch av := a.(type) {
	case int64:
		bv, err := toInt64(b)
		return err == nil && av == bv
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case nil:
		return b == nil
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// --- expression printer ---

// PrintExpr returns a human-readable string representation of an expression.
func PrintExpr(expr *parser.Expr) string {
	if expr == nil || expr.Assignment == nil {
		return ""
	}
	return printAssignment(expr.Assignment)
}

func printAssignment(a *parser.Assignment) string {
	s := printEquality(a.Equality)
	if a.Next != nil {
		s += " = " + printEquality(a.Next)
	}
	return s
}

func printEquality(eq *parser.Equality) string {
	s := printLogical(eq.Logical)
	if eq.Next != nil {
		s += " " + eq.Op + " " + printEquality(eq.Next)
	}
	return s
}

func printLogical(l *parser.Logical) string {
	s := printComparison(l.Comparison)
	if l.Next != nil {
		s += " " + l.Op + " " + printLogical(l.Next)
	}
	return s
}

func printComparison(c *parser.Comparison) string {
	s := printAddition(c.Addition)
	if c.Next != nil {
		s += " " + c.Op + " " + printComparison(c.Next)
	}
	return s
}

func printAddition(a *parser.Addition) string {
	s := printMultiplication(a.Multiplication)
	if a.Next != nil {
		s += " " + a.Op + " " + printAddition(a.Next)
	}
	return s
}

func printMultiplication(m *parser.Multiplication) string {
	s := printUnary(m.Unary)
	if m.Next != nil {
		s += " " + m.Op + " " + printMultiplication(m.Next)
	}
	return s
}

func printUnary(u *parser.Unary) string {
	if u.Unary != nil {
		return u.Op + printUnary(u.Unary)
	}
	if u.Primary != nil {
		return printPrimary(u.Primary)
	}
	return ""
}

func printPrimary(p *parser.Primary) string {
	if p.NumberValue != nil {
		return fmt.Sprintf("%d", *p.NumberValue)
	}
	if p.StringValue != nil {
		return *p.StringValue
	}
	if p.BoolValue != nil {
		return *p.BoolValue
	}
	if p.FuncCall != nil {
		args := make([]string, len(p.FuncCall.Args))
		for i, a := range p.FuncCall.Args {
			args[i] = PrintExpr(a)
		}
		return fmt.Sprintf("%s.%s(%s)", *p.FuncCall.PackageId, *p.FuncCall.FunctionName, strings.Join(args, ", "))
	}
	if p.ForAll != nil {
		return fmt.Sprintf("forall %s in %s: %s", *p.ForAll.LoopVariable, PrintExpr(p.ForAll.LoopOver), PrintExpr(p.ForAll.Expr))
	}
	if p.Id != nil {
		return printVariable(p.Id)
	}
	if p.SubExpression != nil {
		return "(" + PrintExpr(p.SubExpression) + ")"
	}
	return ""
}

func printVariable(v *parser.Variable) string {
	if v.Source == nil {
		return *v.Id
	}
	src := v.Source
	if src.Source != nil {
		return *v.Id + "." + *src.Source
	}
	if src.VariableId != nil {
		return fmt.Sprintf("%s.(%s.%s)", *v.Id, *src.VariableId, *src.VariableSource)
	}
	return *v.Id
}
