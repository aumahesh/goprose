package sim

import (
	"fmt"
	"io/ioutil"
	"math/rand"

	"gopkg.in/yaml.v2"
)

// TopologyNode describes a single node in the topology.
type TopologyNode struct {
	ID    string                 `yaml:"id"`
	State map[string]interface{} `yaml:"state"`
}

// Topology describes the full simulation topology.
type Topology struct {
	Nodes     []*TopologyNode     `yaml:"nodes"`
	Neighbors map[string][]string `yaml:"neighbors"`
}

// LoadTopology reads a topology from a YAML file.
func LoadTopology(path string) (*Topology, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading topology file %s: %w", path, err)
	}
	var topo Topology
	if err := yaml.Unmarshal(data, &topo); err != nil {
		return nil, fmt.Errorf("parsing topology YAML: %w", err)
	}
	if topo.Neighbors == nil {
		topo.Neighbors = make(map[string][]string)
	}
	return &topo, nil
}

// GenerateTopology creates a topology for n nodes with the given style.
// style is one of "ring", "fully-connected", or "random".
func GenerateTopology(n int, style string, rng *rand.Rand) (*Topology, error) {
	if n < 1 {
		return nil, fmt.Errorf("need at least 1 node")
	}

	nodes := make([]*TopologyNode, n)
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		ids[i] = fmt.Sprintf("n%d", i+1)
		nodes[i] = &TopologyNode{ID: ids[i]}
	}

	neighbors := make(map[string][]string, n)
	switch style {
	case "ring":
		for i := 0; i < n; i++ {
			prev := ids[(i-1+n)%n]
			next := ids[(i+1)%n]
			if n == 1 {
				neighbors[ids[i]] = []string{}
			} else if n == 2 {
				neighbors[ids[i]] = []string{ids[1-i]}
			} else {
				neighbors[ids[i]] = dedup([]string{prev, next})
			}
		}
	case "fully-connected":
		for i := 0; i < n; i++ {
			nbrs := make([]string, 0, n-1)
			for j := 0; j < n; j++ {
				if i != j {
					nbrs = append(nbrs, ids[j])
				}
			}
			neighbors[ids[i]] = nbrs
		}
	case "random":
		if n < 2 {
			for _, id := range ids {
				neighbors[id] = []string{}
			}
			break
		}
		// Guarantee connectivity via a random spanning tree, then add extra edges
		perm := rng.Perm(n)
		for i := 1; i < n; i++ {
			a := ids[perm[i]]
			b := ids[perm[rng.Intn(i)]]
			neighbors[a] = append(neighbors[a], b)
			neighbors[b] = append(neighbors[b], a)
		}
		// Optionally add a few random extra edges
		extra := rng.Intn(n / 2)
		for k := 0; k < extra; k++ {
			i, j := rng.Intn(n), rng.Intn(n)
			if i != j {
				neighbors[ids[i]] = append(neighbors[ids[i]], ids[j])
				neighbors[ids[j]] = append(neighbors[ids[j]], ids[i])
			}
		}
		// Deduplicate
		for k := range neighbors {
			neighbors[k] = dedup(neighbors[k])
		}
	default:
		return nil, fmt.Errorf("unknown topology style %q (want ring, fully-connected, or random)", style)
	}

	return &Topology{
		Nodes:     nodes,
		Neighbors: neighbors,
	}, nil
}

func dedup(ss []string) []string {
	seen := make(map[string]bool, len(ss))
	out := ss[:0]
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
