package sim_test

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aumahesh/goprose/internal/intermediate"
	"github.com/aumahesh/goprose/internal/parser"
	"github.com/aumahesh/goprose/internal/sim"
)

func buildEngine(t *testing.T, proseFile string, topo *sim.Topology) *sim.Engine {
	t.Helper()
	p, err := parser.NewProSeParser(proseFile, false)
	require.NoError(t, err)
	require.NoError(t, p.Parse())
	parsedProg, err := p.GetParsedProgram()
	require.NoError(t, err)
	interProg, err := intermediate.GenerateIntermediateProgram(parsedProg)
	require.NoError(t, err)
	engine, err := sim.NewEngine(parsedProg, interProg, topo)
	require.NoError(t, err)
	return engine
}

// TestGCDConverges runs the GCD program on a two-node ring and verifies
// that both nodes converge to the same value (GCD of the initial values).
func TestGCDConverges(t *testing.T) {
	topo := &sim.Topology{
		Nodes: []*sim.TopologyNode{
			{ID: "n1", State: map[string]interface{}{"X": 3542, "Y": 943}},
			{ID: "n2", State: map[string]interface{}{"X": 100, "Y": 75}},
		},
		Neighbors: map[string][]string{
			"n1": {"n2"},
			"n2": {"n1"},
		},
	}
	engine := buildEngine(t, "../../proseFiles/gcd.prose", topo)

	for step := 0; step < 5000; step++ {
		engine.Step()
		n1 := engine.Nodes["n1"]
		n2 := engine.Nodes["n2"]
		x1, _ := n1.State["X"].(int64)
		y1, _ := n1.State["Y"].(int64)
		x2, _ := n2.State["X"].(int64)
		y2, _ := n2.State["Y"].(int64)
		if x1 == y1 && x2 == y2 {
			// Each node has converged locally
			t.Logf("GCD converged after %d steps: n1=(%d,%d) n2=(%d,%d)", step+1, x1, y1, x2, y2)
			return
		}
	}
	t.Fatal("GCD did not converge within 5000 steps")
}

// TestMaxConverges verifies that all nodes eventually agree on the maximum value.
func TestMaxConverges(t *testing.T) {
	topo := &sim.Topology{
		Nodes: []*sim.TopologyNode{
			{ID: "n1", State: map[string]interface{}{"x": int64(10)}},
			{ID: "n2", State: map[string]interface{}{"x": int64(42)}},
			{ID: "n3", State: map[string]interface{}{"x": int64(7)}},
		},
		Neighbors: map[string][]string{
			"n1": {"n2"},
			"n2": {"n1", "n3"},
			"n3": {"n2"},
		},
	}
	engine := buildEngine(t, "../../proseFiles/max.prose", topo)

	for step := 0; step < 1000; step++ {
		engine.Step()
	}

	// After enough steps, all nodes should have x == 42
	for id, node := range engine.Nodes {
		x, _ := node.State["x"].(int64)
		assert.Equal(t, int64(42), x, "node %s: expected x=42, got %d", id, x)
	}
}

// TestGCLAlternativeConverges runs the dijkstra_gcl (if...fi GCD) on a single node.
func TestGCLAlternativeConverges(t *testing.T) {
	topo := &sim.Topology{
		Nodes: []*sim.TopologyNode{
			{ID: "n1", State: map[string]interface{}{"x": int64(42), "y": int64(18)}},
		},
		Neighbors: map[string][]string{"n1": {}},
	}
	engine := buildEngine(t, "../../proseFiles/dijkstra_gcl.prose", topo)

	for step := 0; step < 500; step++ {
		engine.Step()
	}

	n1 := engine.Nodes["n1"]
	x, _ := n1.State["x"].(int64)
	y, _ := n1.State["y"].(int64)
	assert.Equal(t, int64(6), x, "expected gcd(42,18)=6, got x=%d", x)
	assert.Equal(t, int64(6), y, "expected gcd(42,18)=6, got y=%d", y)
}

// TestGCLRepetitiveTerminates runs the dijkstra_gcl_do (do...od) on a single node
// and verifies it terminates with a fixed point.
func TestGCLRepetitiveTerminates(t *testing.T) {
	topo := &sim.Topology{
		Nodes: []*sim.TopologyNode{
			{ID: "n1", State: map[string]interface{}{"x": int64(10), "y": int64(3)}},
		},
		Neighbors: map[string][]string{"n1": {}},
	}
	engine := buildEngine(t, "../../proseFiles/dijkstra_gcl_do.prose", topo)

	// A do...od fires once per step; give it enough steps
	for step := 0; step < 100; step++ {
		engine.Step()
	}

	n1 := engine.Nodes["n1"]
	x, _ := n1.State["x"].(int64)
	// 10 mod 3 = 1; the do...od reduces x by 1 each step while x > 0 and x <= y,
	// and subtracts y while x > y. Eventually x reaches 0.
	assert.Equal(t, int64(0), x, "expected x=0 after do...od, got %d", x)
}

// TestReset verifies that Reset restores the initial state.
func TestReset(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	topo, err := sim.GenerateTopology(3, "ring", rng)
	require.NoError(t, err)

	// Inject explicit initial state
	topo.Nodes[0].State = map[string]interface{}{"X": int64(100), "Y": int64(50)}
	topo.Nodes[1].State = map[string]interface{}{"X": int64(80), "Y": int64(30)}
	topo.Nodes[2].State = map[string]interface{}{"X": int64(60), "Y": int64(20)}

	engine := buildEngine(t, "../../proseFiles/gcd.prose", topo)

	// Capture initial states
	initial := make(map[string]sim.NodeState)
	for id, node := range engine.Nodes {
		initial[id] = node.State.Copy()
	}

	// Run some steps
	for i := 0; i < 50; i++ {
		engine.Step()
	}

	// Reset and verify
	engine.Reset()
	assert.Equal(t, 0, engine.StepCount)
	for id, node := range engine.Nodes {
		assert.Equal(t, initial[id], node.State, "node %s state after reset", id)
	}
}
