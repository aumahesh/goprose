package main

import (
	"flag"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/aumahesh/goprose/internal/intermediate"
	"github.com/aumahesh/goprose/internal/parser"
	"github.com/aumahesh/goprose/internal/sim"
	"github.com/aumahesh/goprose/internal/viz"
)

// nodeStateOverrides is a repeatable flag: --node-state n1.isEvaderHere=true
type nodeStateOverrides []string

func (f *nodeStateOverrides) String() string  { return strings.Join(*f, ", ") }
func (f *nodeStateOverrides) Set(v string) error { *f = append(*f, v); return nil }

func main() {
	log.SetLevel(log.InfoLevel)

	var (
		proseFile    = flag.String("p", "", "path to the .prose file (required)")
		topoFile     = flag.String("topo-file", "", "path to a YAML topology file")
		numNodes     = flag.Int("nodes", 3, "number of nodes (used with --topology)")
		topology     = flag.String("topology", "ring", "topology style: ring, fully-connected, random")
		addr         = flag.String("addr", ":8080", "HTTP listen address")
		templatesDir = flag.String("templates", "templates/", "path to the templates/ directory")
		nodeStates   nodeStateOverrides
	)
	flag.Var(&nodeStates, "node-state", "per-node state override: <nodeID>.<varName>=<value> (repeatable)")
	flag.Parse()

	if *proseFile == "" {
		log.Fatal("--p <prose-file> is required")
	}

	// Parse the prose file
	p, err := parser.NewProSeParser(*proseFile, false)
	if err != nil {
		log.Fatalf("parser: %v", err)
	}
	if err := p.Parse(); err != nil {
		log.Fatalf("parse error: %v", err)
	}
	parsedProg, err := p.GetParsedProgram()
	if err != nil {
		log.Fatalf("get program: %v", err)
	}

	// Build the intermediate representation (for variable metadata)
	interProg, err := intermediate.GenerateIntermediateProgram(parsedProg)
	if err != nil {
		log.Fatalf("intermediate: %v", err)
	}

	// Load or generate topology
	var topo *sim.Topology
	if *topoFile != "" {
		topo, err = sim.LoadTopology(*topoFile)
		if err != nil {
			log.Fatalf("topology: %v", err)
		}
	} else {
		rng := rand.New(rand.NewSource(42))
		topo, err = sim.GenerateTopology(*numNodes, *topology, rng)
		if err != nil {
			log.Fatalf("topology: %v", err)
		}
	}

	// Apply per-node state overrides (--node-state n1.isEvaderHere=true)
	if err := applyNodeStateOverrides(topo, nodeStates); err != nil {
		log.Fatalf("--node-state: %v", err)
	}

	// Build simulation engine
	engine, err := sim.NewEngine(parsedProg, interProg, topo)
	if err != nil {
		log.Fatalf("engine: %v", err)
	}

	// Start web server
	server := viz.NewServer(engine, *proseFile, *templatesDir, parsedProg)
	if err := server.Start(*addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// applyNodeStateOverrides parses entries of the form "<nodeID>.<varName>=<value>"
// and writes them into the matching topology node's State map.
func applyNodeStateOverrides(topo *sim.Topology, overrides []string) error {
	// Build a quick lookup from node ID to TopologyNode
	nodeByID := make(map[string]*sim.TopologyNode, len(topo.Nodes))
	for _, n := range topo.Nodes {
		nodeByID[n.ID] = n
	}

	for _, o := range overrides {
		eq := strings.IndexByte(o, '=')
		if eq < 0 {
			return fmt.Errorf("invalid format %q: want <nodeID>.<varName>=<value>", o)
		}
		lhs, rawVal := o[:eq], o[eq+1:]

		dot := strings.IndexByte(lhs, '.')
		if dot < 0 {
			return fmt.Errorf("invalid format %q: missing '.' between nodeID and varName", o)
		}
		nodeID, varName := lhs[:dot], lhs[dot+1:]

		n, ok := nodeByID[nodeID]
		if !ok {
			return fmt.Errorf("node %q not found in topology", nodeID)
		}
		if n.State == nil {
			n.State = make(map[string]interface{})
		}
		n.State[varName] = parseScalar(rawVal)
	}
	return nil
}

// parseScalar converts a string value to bool, int64, or string.
func parseScalar(s string) interface{} {
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	return s
}
