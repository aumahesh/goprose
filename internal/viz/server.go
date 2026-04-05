package viz

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/aumahesh/goprose/internal/intermediate"
	"github.com/aumahesh/goprose/internal/parser"
	"github.com/aumahesh/goprose/internal/sim"
	"github.com/aumahesh/goprose/internal/templates"
)

// Server wraps the simulation engine with an HTTP interface.
type Server struct {
	engine        *sim.Engine
	proseFile     string
	templatesDir  string
	parsedProg    *parser.Program
}

// NewServer creates a Server for the given engine and prose source file.
// templatesDir is the path to the templates/ directory for the compile endpoint.
func NewServer(engine *sim.Engine, proseFile, templatesDir string, parsedProg *parser.Program) *Server {
	return &Server{
		engine:       engine,
		proseFile:    proseFile,
		templatesDir: templatesDir,
		parsedProg:   parsedProg,
	}
}

// Start registers handlers and begins listening on addr (e.g., ":8080").
func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleUI)
	mux.HandleFunc("/api/state", s.handleState)
	mux.HandleFunc("/api/step", s.handleStep)
	mux.HandleFunc("/api/reset", s.handleReset)
	mux.HandleFunc("/api/compile", s.handleCompile)

	log.Infof("prose-sim: listening on http://localhost%s", addr)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	return srv.ListenAndServe()
}

// --- state types ---

type nodeDTO struct {
	ID        string                 `json:"id"`
	State     map[string]interface{} `json:"state"`
	Neighbors []string               `json:"neighbors"`
}

type edgeDTO struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type stateResponse struct {
	Step     int       `json:"step"`
	Nodes    []nodeDTO `json:"nodes"`
	Edges    []edgeDTO `json:"edges"`
	VarNames []string  `json:"varNames"`
}

type stepResponse struct {
	stateResponse
	Result sim.StepResult `json:"result"`
}

func (s *Server) buildStateResponse() stateResponse {
	nodes := make([]nodeDTO, 0, len(s.engine.NodeOrder))
	seen := make(map[string]bool)
	var edges []edgeDTO

	for _, id := range s.engine.NodeOrder {
		node := s.engine.Nodes[id]
		nodes = append(nodes, nodeDTO{
			ID:        id,
			State:     map[string]interface{}(node.State),
			Neighbors: node.NeighborIDs,
		})
		for _, nbrID := range node.NeighborIDs {
			key := edgeKey(id, nbrID)
			if !seen[key] {
				seen[key] = true
				edges = append(edges, edgeDTO{Source: id, Target: nbrID})
			}
		}
	}

	return stateResponse{
		Step:     s.engine.StepCount,
		Nodes:    nodes,
		Edges:    edges,
		VarNames: s.engine.VarNames(),
	}
}

func edgeKey(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return a + "~" + b
}

// --- handlers ---

func (s *Server) handleUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(uiHTML))
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.buildStateResponse())
}

func (s *Server) handleStep(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	result := s.engine.Step()
	writeJSON(w, stepResponse{
		stateResponse: s.buildStateResponse(),
		Result:        result,
	})
}

func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	s.engine.Reset()
	writeJSON(w, s.buildStateResponse())
}

func (s *Server) handleCompile(w http.ResponseWriter, r *http.Request) {
	tmpDir, err := os.MkdirTemp("", "prose-compile-*")
	if err != nil {
		http.Error(w, "failed to create temp dir: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	// Re-parse and compile the prose file
	p, err := parser.NewProSeParser(s.proseFile, false)
	if err != nil {
		http.Error(w, "parse error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := p.Parse(); err != nil {
		http.Error(w, "parse error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	prog, err := p.GetParsedProgram()
	if err != nil {
		http.Error(w, "parse error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	ip, err := intermediate.GenerateIntermediateProgram(prog)
	if err != nil {
		http.Error(w, "codegen error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tm, err := templates.NewTemplateManager(s.templatesDir, tmpDir, ip)
	if err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tm.Render(); err != nil {
		http.Error(w, "render error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Zip the generated module directory
	modDir := filepath.Join(tmpDir, ip.ModuleName)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.zip"`, ip.ModuleName))

	zw := zip.NewWriter(w)
	defer zw.Close()

	err = filepath.Walk(modDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(tmpDir, path)
		f, err := zw.Create(rel)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = f.Write(data)
		return err
	})
	if err != nil {
		log.Errorf("zip error: %v", err)
	}
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Errorf("JSON encode error: %v", err)
	}
}
