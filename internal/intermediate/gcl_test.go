package intermediate

import (
	"strings"
	"testing"

	"github.com/aumahesh/goprose/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const gclProseFile = "../../proseFiles/dijkstra_gcl.prose"

func parseAndGenerate(t *testing.T, proseFile string) *Program {
	t.Helper()
	p, err := parser.NewProSeParser(proseFile, false)
	require.NoError(t, err)
	require.NoError(t, p.Parse())

	parsed, err := p.GetParsedProgram()
	require.NoError(t, err)

	prog, err := GenerateIntermediateProgram(parsed)
	require.NoError(t, err)
	return prog
}

// TestGCLAlternativeConstructCodeGen verifies that an if...fi construct generates
// code with the expected non-deterministic selection pattern.
func TestGCLAlternativeConstructCodeGen(t *testing.T) {
	prog := parseAndGenerate(t, gclProseFile)

	require.Len(t, prog.Statements, 1, "dijkstra_gcl.prose should have exactly one statement")

	stmt := prog.Statements[0]
	actionCode := strings.Join(stmt.ActionCode, "\n")

	// Must allocate a choice slice
	assert.Contains(t, actionCode, "[]int", "action should declare a choices slice")
	// Must use non-deterministic selection
	assert.Contains(t, actionCode, "rand.Intn", "action should use rand.Intn for non-deterministic choice")
	// Must dispatch via switch
	assert.Contains(t, actionCode, "switch", "action should use switch to dispatch the chosen command")
	// Both branches of the if...fi must be present
	assert.Contains(t, actionCode, "case 0:", "action should have case 0 for first guarded command")
	assert.Contains(t, actionCode, "case 1:", "action should have case 1 for second guarded command")
	// Both assignments must be emitted
	assert.Contains(t, actionCode, "this.state.X", "action should assign to X")
	assert.Contains(t, actionCode, "this.state.Y", "action should assign to Y")
}

// TestGCLAlternativeConstructSkipsWhenNoGuardTrue verifies that when neither guard
// of an if...fi is true the construct is a no-op (len(choices)==0 branch).
func TestGCLAlternativeConstructSkipsWhenNoGuardTrue(t *testing.T) {
	prog := parseAndGenerate(t, gclProseFile)
	stmt := prog.Statements[0]
	actionCode := strings.Join(stmt.ActionCode, "\n")

	// The generated code must guard execution on a non-empty choice set.
	assert.Contains(t, actionCode, "len(", "action should check length of choices")
	assert.Contains(t, actionCode, "> 0", "action should only execute when at least one guard is true")
}

// TestRepetitiveConstructCodeGen verifies that do...od generates a for loop
// that breaks when no guard is true.
func TestRepetitiveConstructCodeGen(t *testing.T) {
	const doOdProse = "../../proseFiles/dijkstra_gcl_do.prose"
	prog := parseAndGenerate(t, doOdProse)

	require.NotEmpty(t, prog.Statements)
	stmt := prog.Statements[0]
	actionCode := strings.Join(stmt.ActionCode, "\n")

	assert.Contains(t, actionCode, "for {", "do...od should generate a for loop")
	assert.Contains(t, actionCode, "break", "do...od should break when no guard is true")
	assert.Contains(t, actionCode, "rand.Intn", "do...od should use rand.Intn for non-deterministic choice")
}
