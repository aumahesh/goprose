package main

import (
	"flag"
	"math/rand"

	log "github.com/sirupsen/logrus"

	"github.com/aumahesh/goprose/internal/intermediate"
	"github.com/aumahesh/goprose/internal/parser"
	"github.com/aumahesh/goprose/internal/sim"
	"github.com/aumahesh/goprose/internal/viz"
)

func main() {
	log.SetLevel(log.InfoLevel)

	var (
		proseFile    = flag.String("p", "", "path to the .prose file (required)")
		topoFile     = flag.String("topo-file", "", "path to a YAML topology file")
		numNodes     = flag.Int("nodes", 3, "number of nodes (used with --topology)")
		topology     = flag.String("topology", "ring", "topology style: ring, fully-connected, random")
		addr         = flag.String("addr", ":8080", "HTTP listen address")
		templatesDir = flag.String("templates", "templates/", "path to the templates/ directory")
	)
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
