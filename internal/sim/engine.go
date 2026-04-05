package sim

import (
	"fmt"
	"math/rand"
	"sort"

	log "github.com/sirupsen/logrus"

	"github.com/aumahesh/goprose/internal/intermediate"
	"github.com/aumahesh/goprose/internal/parser"
)

// NodeState maps variable names to their current values (int64, bool, or string).
type NodeState map[string]interface{}

// Copy returns a shallow copy of the state.
func (s NodeState) Copy() NodeState {
	c := make(NodeState, len(s))
	for k, v := range s {
		c[k] = v
	}
	return c
}

// StateChange records a single variable mutation.
type StateChange struct {
	Var  string      `json:"var"`
	From interface{} `json:"from"`
	To   interface{} `json:"to"`
}

// StepResult records the outcome of one simulation step.
type StepResult struct {
	Step       int           `json:"step"`
	NodeID     string        `json:"nodeId"`
	GuardIndex int           `json:"guardIndex"`
	GuardText  string        `json:"guardText"`
	Changes    []StateChange `json:"changes"`
	Fired      bool          `json:"fired"`
}

// Node represents a single process in the simulation.
type Node struct {
	ID           string
	State        NodeState
	NeighborIDs  []string
	initialState NodeState // for Reset
	priority     []int64   // current countdown (0 = eligible this step)
	configPrio   []int64   // configured priority per statement
}

func (n *Node) reset() {
	n.State = n.initialState.Copy()
	for i := range n.priority {
		n.priority[i] = 0
	}
}

// Engine runs the multi-node asynchronous ProSe simulation.
type Engine struct {
	Nodes     map[string]*Node
	NodeOrder []string // stable ordering for the UI
	StepCount int
	Log       []StepResult

	parsedProg *parser.Program
	interProg  *intermediate.Program
	varNames   []string          // ordered variable names for display
	sensorID   string
	constants  NodeState
	rng        *rand.Rand
}

// NewEngine constructs an Engine from a parsed program and a topology.
func NewEngine(parsedProg *parser.Program, interProg *intermediate.Program, topo *Topology) (*Engine, error) {
	sensorID := *parsedProg.Sensor

	// Build constant state
	constants := make(NodeState)
	for name, v := range interProg.Constants {
		constants[name] = v.DefaultValue
	}
	rng := rand.New(rand.NewSource(42))

	// Evaluate constant init expressions
	for name, expr := range interProg.ConstantInitFunctions {
		ctx := &EvalCtx{
			LocalState: make(NodeState),
			SensorID:   sensorID,
			Constants:  constants,
			RNG:        rng,
		}
		val, err := EvalExpr(expr.GetExpr(), ctx)
		if err != nil {
			log.Warnf("constant %s init expression failed: %v", name, err)
		} else {
			constants[name] = coerceValue(val, interProg.Constants[name].ProseType)
		}
	}

	// Collect variable names in a stable order
	varNames := make([]string, 0, len(interProg.Variables))
	for name := range interProg.Variables {
		varNames = append(varNames, name)
	}
	sort.Strings(varNames)

	// Collect configured priorities
	numStmts := len(parsedProg.Statements)
	configPrios := make([]int64, numStmts)
	for i, stmt := range parsedProg.Statements {
		if stmt.Guarded != nil && stmt.Guarded.Priority != nil {
			configPrios[i] = *stmt.Guarded.Priority
		} else {
			configPrios[i] = 1
		}
	}

	// Build nodes
	nodes := make(map[string]*Node, len(topo.Nodes))
	nodeOrder := make([]string, 0, len(topo.Nodes))
	for _, tn := range topo.Nodes {
		state, err := buildInitialState(tn, interProg, parsedProg, sensorID, constants, rng)
		if err != nil {
			return nil, fmt.Errorf("node %s: %w", tn.ID, err)
		}

		prios := make([]int64, numStmts)
		// all start at 0 (eligible on first step)
		for i := range prios {
			prios[i] = 0
		}
		cpios := make([]int64, numStmts)
		copy(cpios, configPrios)

		node := &Node{
			ID:           tn.ID,
			State:        state,
			NeighborIDs:  topo.Neighbors[tn.ID],
			initialState: state.Copy(),
			priority:     prios,
			configPrio:   cpios,
		}
		nodes[tn.ID] = node
		nodeOrder = append(nodeOrder, tn.ID)
	}
	sort.Strings(nodeOrder)

	return &Engine{
		Nodes:      nodes,
		NodeOrder:  nodeOrder,
		parsedProg: parsedProg,
		interProg:  interProg,
		varNames:   varNames,
		sensorID:   sensorID,
		constants:  constants,
		rng:        rng,
	}, nil
}

// buildInitialState computes a node's initial variable state.
// rng is the shared engine RNG so that init expressions (e.g. rand.Int63n)
// draw different values for each node.
func buildInitialState(tn *TopologyNode, interProg *intermediate.Program, parsedProg *parser.Program, sensorID string, constants NodeState, rng *rand.Rand) (NodeState, error) {
	state := make(NodeState)

	// Set defaults from intermediate variables
	for name, v := range interProg.Variables {
		if tn.State != nil {
			if val, ok := tn.State[name]; ok {
				state[name] = coerceValue(val, v.ProseType)
				continue
			}
		}
		// Evaluate init expression if present
		if initExpr, ok := interProg.VariableInitFunctions[name]; ok {
			ctx := &EvalCtx{
				LocalState: state,
				SensorID:   sensorID,
				NodeID:     tn.ID,
				Constants:  constants,
				RNG:        rng,
			}
			val, err := EvalExpr(initExpr.GetExpr(), ctx)
			if err != nil {
				log.Warnf("variable %s init failed: %v, using default", name, err)
				state[name] = v.DefaultValue
			} else {
				state[name] = coerceValue(val, v.ProseType)
			}
		} else {
			state[name] = v.DefaultValue
		}
	}
	return state, nil
}

// coerceValue converts a value to the appropriate Go type for a ProSe type.
func coerceValue(val interface{}, proseType intermediate.ProseType) interface{} {
	switch proseType {
	case intermediate.ProseTypeInt:
		i, err := toInt64(val)
		if err == nil {
			return i
		}
		return int64(0)
	case intermediate.ProseTypeBool:
		b, err := toBool(val)
		if err == nil {
			return b
		}
		return false
	case intermediate.ProseTypeString:
		if s, ok := val.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", val)
	}
	return val
}

// VarNames returns the ordered list of variable names.
func (e *Engine) VarNames() []string {
	return e.varNames
}

// Step picks a random node and advances it one step.
func (e *Engine) Step() StepResult {
	idx := e.rng.Intn(len(e.NodeOrder))
	nodeID := e.NodeOrder[idx]
	node := e.Nodes[nodeID]

	result := e.stepNode(node)
	result.Step = e.StepCount + 1
	e.StepCount++
	e.Log = append(e.Log, result)
	return result
}

func (e *Engine) stepNode(node *Node) StepResult {
	// Build neighbor context
	allNeighbors := e.buildNeighborList(node)
	neighborStates := e.buildNeighborStates(node)

	// Evaluate guards for all eligible statements
	type candidate struct {
		index    int
		neighbor *NeighborInfo
		text     string
	}
	var candidates []candidate

	for i, stmt := range e.parsedProg.Statements {
		// Check and update priority
		if node.priority[i] > 0 {
			node.priority[i]--
			continue
		}

		guardText, nbr, ok := e.evalGuard(node, stmt, allNeighbors, neighborStates)
		if ok {
			candidates = append(candidates, candidate{i, nbr, guardText})
		}
		// Reset priority counter after eligibility (whether guard was true or not)
		node.priority[i] = node.configPrio[i] - 1
	}

	if len(candidates) == 0 {
		return StepResult{NodeID: node.ID, Fired: false}
	}

	// Pick a random true guard
	pick := candidates[e.rng.Intn(len(candidates))]

	prevState := node.State.Copy()
	e.execAction(node, e.parsedProg.Statements[pick.index], pick.neighbor, allNeighbors, neighborStates)

	changes := diffState(prevState, node.State, e.varNames)
	return StepResult{
		NodeID:     node.ID,
		GuardIndex: pick.index,
		GuardText:  pick.text,
		Changes:    changes,
		Fired:      true,
	}
}

// evalGuard evaluates the outer guard of a statement.
// Returns (guardText, selectedNeighbor, true) if guard fires.
func (e *Engine) evalGuard(node *Node, stmt *parser.Statement, allNeighbors []*NeighborInfo, neighborStates map[string]NodeState) (string, *NeighborInfo, bool) {
	if stmt.Alternative != nil {
		return "if...fi", nil, true
	}
	if stmt.Repetitive != nil {
		return "do...od", nil, true
	}
	if stmt.Guarded == nil {
		return "", nil, false
	}

	g := stmt.Guarded
	guardText := PrintExpr(g.Guard)

	// Check if the guard references any remote (neighbor) variable
	ctx := &EvalCtx{
		LocalState:     node.State,
		SensorID:       e.sensorID,
		NodeID:         node.ID,
		Constants:      e.constants,
		AllNeighbors:   allNeighbors,
		NeighborStates: neighborStates,
		RNG:            e.rng,
	}

	exprType, _ := getExpressionType(g.Guard, e.sensorID, e.interProg.Constants, e.interProg.Variables)

	if exprType != exprTypeRemote {
		// Local guard: single evaluation
		val, err := EvalExpr(g.Guard, ctx)
		if err != nil {
			log.Debugf("guard eval error: %v", err)
			return guardText, nil, false
		}
		ok, _ := toBool(val)
		return guardText, nil, ok
	}

	// Remote guard: try each neighbor
	for _, nbr := range allNeighbors {
		ctx2 := *ctx
		ctx2.Neighbor = nbr
		ctx2.NeighborID = nbr.ID
		val, err := EvalExpr(g.Guard, &ctx2)
		if err != nil {
			continue
		}
		ok, _ := toBool(val)
		if ok {
			return guardText, nbr, true
		}
	}
	return guardText, nil, false
}

// execAction executes the action part of a chosen statement.
func (e *Engine) execAction(node *Node, stmt *parser.Statement, selectedNbr *NeighborInfo, allNeighbors []*NeighborInfo, neighborStates map[string]NodeState) {
	ctx := &EvalCtx{
		LocalState:     node.State,
		Neighbor:       selectedNbr,
		NeighborStates: neighborStates,
		SensorID:       e.sensorID,
		NodeID:         node.ID,
		Constants:      e.constants,
		AllNeighbors:   allNeighbors,
		RNG:            e.rng,
	}
	if selectedNbr != nil {
		ctx.NeighborID = selectedNbr.ID
	}

	if stmt.Guarded != nil {
		for _, act := range stmt.Guarded.Actions {
			if err := e.execGclAction(node, act, ctx); err != nil {
				log.Debugf("action error: %v", err)
			}
		}
		return
	}
	if stmt.Alternative != nil {
		e.execAlternativeConstruct(node, stmt.Alternative, ctx)
		return
	}
	if stmt.Repetitive != nil {
		e.execRepetitiveConstruct(node, stmt.Repetitive, ctx)
		return
	}
}

func (e *Engine) execGclAction(node *Node, act *parser.GclAction, ctx *EvalCtx) error {
	if act.Assignment != nil {
		return e.execAssignment(node, act.Assignment, ctx)
	}
	if act.AlternativeConstruct != nil {
		e.execAlternativeConstruct(node, act.AlternativeConstruct, ctx)
		return nil
	}
	if act.RepetitiveConstruct != nil {
		e.execRepetitiveConstruct(node, act.RepetitiveConstruct, ctx)
		return nil
	}
	return fmt.Errorf("empty GCL action")
}

func (e *Engine) execAssignment(node *Node, act *parser.Action, ctx *EvalCtx) error {
	if len(act.Variable) != len(act.Expr) {
		return fmt.Errorf("assignment: %d LHS vs %d RHS", len(act.Variable), len(act.Expr))
	}

	// Evaluate all RHS values before any assignment (simultaneous semantics)
	newVals := make([]interface{}, len(act.Expr))
	for i, expr := range act.Expr {
		val, err := EvalExpr(expr, ctx)
		if err != nil {
			return fmt.Errorf("RHS %d: %w", i, err)
		}
		newVals[i] = val
	}

	// Apply assignments
	for i, v := range act.Variable {
		if v.Source == nil || v.Source.Source == nil {
			return fmt.Errorf("assignment LHS %d: missing source", i)
		}
		if *v.Source.Source != e.sensorID {
			return fmt.Errorf("cannot assign to remote variable %s", *v.Id)
		}
		varName := *v.Id
		iv, ok := e.interProg.Variables[varName]
		if !ok {
			return fmt.Errorf("unknown variable %s", varName)
		}
		node.State[varName] = coerceValue(newVals[i], iv.ProseType)
		// Keep ctx.LocalState in sync so subsequent assignments see the updated value
		ctx.LocalState = node.State
	}
	return nil
}

func (e *Engine) execAlternativeConstruct(node *Node, ac *parser.AlternativeConstruct, ctx *EvalCtx) {
	// Evaluate all inner guards, collect true ones, pick one at random
	var trueIdx []int
	for i, cmd := range ac.Commands {
		innerCtx := *ctx
		val, err := EvalExpr(cmd.Guard, &innerCtx)
		if err != nil {
			continue
		}
		ok, _ := toBool(val)
		if ok {
			trueIdx = append(trueIdx, i)
		}
	}
	if len(trueIdx) == 0 {
		return
	}
	pick := trueIdx[e.rng.Intn(len(trueIdx))]
	for _, act := range ac.Commands[pick].Actions {
		if err := e.execGclAction(node, act, ctx); err != nil {
			log.Debugf("alternative action error: %v", err)
		}
	}
}

func (e *Engine) execRepetitiveConstruct(node *Node, rc *parser.RepetitiveConstruct, ctx *EvalCtx) {
	const maxIter = 10000
	for iter := 0; iter < maxIter; iter++ {
		var trueIdx []int
		for i, cmd := range rc.Commands {
			innerCtx := *ctx
			val, err := EvalExpr(cmd.Guard, &innerCtx)
			if err != nil {
				continue
			}
			ok, _ := toBool(val)
			if ok {
				trueIdx = append(trueIdx, i)
			}
		}
		if len(trueIdx) == 0 {
			return
		}
		pick := trueIdx[e.rng.Intn(len(trueIdx))]
		for _, act := range rc.Commands[pick].Actions {
			if err := e.execGclAction(node, act, ctx); err != nil {
				log.Debugf("repetitive action error: %v", err)
			}
		}
	}
	log.Warnf("do...od exceeded %d iterations, terminating", maxIter)
}

// Reset restores all nodes to their initial states.
func (e *Engine) Reset() {
	e.StepCount = 0
	e.Log = nil
	for _, node := range e.Nodes {
		node.reset()
	}
}

// buildNeighborList returns NeighborInfo for all of node's neighbors.
func (e *Engine) buildNeighborList(node *Node) []*NeighborInfo {
	nbrs := make([]*NeighborInfo, 0, len(node.NeighborIDs))
	for _, nid := range node.NeighborIDs {
		if n, ok := e.Nodes[nid]; ok {
			nbrs = append(nbrs, &NeighborInfo{ID: nid, State: n.State})
		}
	}
	return nbrs
}

func (e *Engine) buildNeighborStates(node *Node) map[string]NodeState {
	m := make(map[string]NodeState, len(node.NeighborIDs))
	for _, nid := range node.NeighborIDs {
		if n, ok := e.Nodes[nid]; ok {
			m[nid] = n.State
		}
	}
	return m
}

// diffState returns the list of variables that changed between before and after.
func diffState(before, after NodeState, varNames []string) []StateChange {
	var changes []StateChange
	for _, name := range varNames {
		b := before[name]
		a := after[name]
		if !valEquals(b, a) {
			changes = append(changes, StateChange{Var: name, From: b, To: a})
		}
	}
	return changes
}

// getExpressionType is a lightweight check for whether an expression references remote variables.
type exprType int

const (
	exprTypeLocal  exprType = iota
	exprTypeRemote exprType = iota
)

func getExpressionType(expr *parser.Expr, sensorID string, constants, variables map[string]*intermediate.Variable) (exprType, error) {
	if expr == nil || expr.Assignment == nil {
		return exprTypeLocal, nil
	}
	return getAssignmentType(expr.Assignment, sensorID, constants, variables)
}

func getAssignmentType(a *parser.Assignment, sensorID string, constants, variables map[string]*intermediate.Variable) (exprType, error) {
	t, err := getEqualityType(a.Equality, sensorID, constants, variables)
	if err != nil || t == exprTypeRemote {
		return t, err
	}
	if a.Next != nil {
		return getEqualityType(a.Next, sensorID, constants, variables)
	}
	return t, nil
}

func getEqualityType(eq *parser.Equality, sensorID string, constants, variables map[string]*intermediate.Variable) (exprType, error) {
	t, err := getLogicalType(eq.Logical, sensorID, constants, variables)
	if err != nil || t == exprTypeRemote {
		return t, err
	}
	if eq.Next != nil {
		return getEqualityType(eq.Next, sensorID, constants, variables)
	}
	return t, nil
}

func getLogicalType(l *parser.Logical, sensorID string, constants, variables map[string]*intermediate.Variable) (exprType, error) {
	t, err := getComparisonType(l.Comparison, sensorID, constants, variables)
	if err != nil || t == exprTypeRemote {
		return t, err
	}
	if l.Next != nil {
		return getLogicalType(l.Next, sensorID, constants, variables)
	}
	return t, nil
}

func getComparisonType(c *parser.Comparison, sensorID string, constants, variables map[string]*intermediate.Variable) (exprType, error) {
	t, err := getAdditionType(c.Addition, sensorID, constants, variables)
	if err != nil || t == exprTypeRemote {
		return t, err
	}
	if c.Next != nil {
		return getComparisonType(c.Next, sensorID, constants, variables)
	}
	return t, nil
}

func getAdditionType(a *parser.Addition, sensorID string, constants, variables map[string]*intermediate.Variable) (exprType, error) {
	t, err := getMultiplicationType(a.Multiplication, sensorID, constants, variables)
	if err != nil || t == exprTypeRemote {
		return t, err
	}
	if a.Next != nil {
		return getAdditionType(a.Next, sensorID, constants, variables)
	}
	return t, nil
}

func getMultiplicationType(m *parser.Multiplication, sensorID string, constants, variables map[string]*intermediate.Variable) (exprType, error) {
	t, err := getUnaryType(m.Unary, sensorID, constants, variables)
	if err != nil || t == exprTypeRemote {
		return t, err
	}
	if m.Next != nil {
		return getMultiplicationType(m.Next, sensorID, constants, variables)
	}
	return t, nil
}

func getUnaryType(u *parser.Unary, sensorID string, constants, variables map[string]*intermediate.Variable) (exprType, error) {
	if u.Unary != nil {
		return getUnaryType(u.Unary, sensorID, constants, variables)
	}
	if u.Primary != nil {
		return getPrimaryType(u.Primary, sensorID, constants, variables)
	}
	return exprTypeLocal, nil
}

func getPrimaryType(p *parser.Primary, sensorID string, constants, variables map[string]*intermediate.Variable) (exprType, error) {
	if p.Id != nil {
		return getVariableType(p.Id, sensorID, constants, variables)
	}
	if p.SubExpression != nil {
		return getExpressionType(p.SubExpression, sensorID, constants, variables)
	}
	if p.ForAll != nil {
		return exprTypeLocal, nil // handled separately by iterating all neighbors
	}
	return exprTypeLocal, nil
}

func getVariableType(v *parser.Variable, sensorID string, constants, variables map[string]*intermediate.Variable) (exprType, error) {
	if v.Source == nil {
		return exprTypeLocal, nil
	}
	src := v.Source
	if src.Source != nil {
		if *src.Source == sensorID {
			return exprTypeLocal, nil
		}
		return exprTypeRemote, nil
	}
	if src.VariableId != nil {
		return exprTypeRemote, nil
	}
	return exprTypeLocal, nil
}
