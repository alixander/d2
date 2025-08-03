package godagre

import (
	"fmt"
)

type Graph struct {
	compound   bool
	multigraph bool
	directed   bool
	
	// Graph attributes
	attrs map[string]interface{}
	
	// Node storage
	nodes     map[string]*Node
	nodeCount int
	
	// Edge storage - using a map of edge key to edge
	edges     map[string]*Edge
	edgeCount int
	
	// Parent-child relationships for compound graphs
	parent   map[string]string
	children map[string][]string
	
	// Algorithm state
	maxRank int
	minRank int
}

type Node struct {
	ID     string
	Width  float64
	Height float64
	X      float64
	Y      float64
	Rank   int
	Order  int
	
	// For network simplex
	Low      int
	Lim      int
	Parent   string
	Cutvalue float64
	
	// For crossing minimization
	In         []*Edge // incoming edges
	Out        []*Edge // outgoing edges
	Barycenter float64
	Weight     float64
	
	// For coordinate assignment
	Dummy     bool
	BorderTop string
	BorderBottom string
	BorderLeft   []*Node
	BorderRight  []*Node
	
	// Additional attributes
	attrs map[string]interface{}
}

type Edge struct {
	V      string // source node
	W      string // target node
	Name   string // edge name for multigraphs
	Width  float64
	Height float64
	
	// Edge properties
	Weight float64
	Minlen int
	
	// For network simplex
	Cutvalue float64
	Tree     bool
	Reversed bool
	
	// Layout properties
	Points []Point
	X      float64
	Y      float64
	LabelRank int
	LabelOffset float64
	
	// Additional attributes
	attrs map[string]interface{}
}

type Point struct {
	X float64
	Y float64
}

type GraphOptions struct {
	Compound   bool
	Multigraph bool
	Directed   bool
}

// NewGraph creates a new graph with the given options
func NewGraph(opts GraphOptions) *Graph {
	return &Graph{
		compound:   opts.Compound,
		multigraph: opts.Multigraph,
		directed:   opts.Directed,
		attrs:      make(map[string]interface{}),
		nodes:      make(map[string]*Node),
		edges:      make(map[string]*Edge),
		parent:     make(map[string]string),
		children:   make(map[string][]string),
	}
}

// SetGraph sets graph-level attributes
func (g *Graph) SetGraph(attrs map[string]interface{}) {
	for k, v := range attrs {
		g.attrs[k] = v
	}
}

// GetGraph returns a graph attribute
func (g *Graph) GetGraph(key string) interface{} {
	return g.attrs[key]
}

// SetNode adds or updates a node
func (g *Graph) SetNode(id string, attrs map[string]interface{}) {
	if _, exists := g.nodes[id]; !exists {
		g.nodeCount++
	}
	
	node := &Node{
		ID:    id,
		attrs: make(map[string]interface{}),
		In:    make([]*Edge, 0),
		Out:   make([]*Edge, 0),
	}
	
	// Extract known attributes
	if w, ok := attrs["width"].(float64); ok {
		node.Width = w
	} else if w, ok := attrs["width"].(int); ok {
		node.Width = float64(w)
	}
	
	if h, ok := attrs["height"].(float64); ok {
		node.Height = h
	} else if h, ok := attrs["height"].(int); ok {
		node.Height = float64(h)
	}
	
	// Store remaining attributes
	for k, v := range attrs {
		if k != "width" && k != "height" {
			node.attrs[k] = v
		}
	}
	
	g.nodes[id] = node
}

// GetNode returns a node by ID
func (g *Graph) GetNode(id string) *Node {
	return g.nodes[id]
}

// Nodes returns all node IDs
func (g *Graph) Nodes() []string {
	ids := make([]string, 0, len(g.nodes))
	for id := range g.nodes {
		ids = append(ids, id)
	}
	return ids
}

// NodeCount returns the number of nodes
func (g *Graph) NodeCount() int {
	return g.nodeCount
}

// SetParent sets the parent of a node (for compound graphs)
func (g *Graph) SetParent(child, parent string) error {
	if !g.compound {
		return fmt.Errorf("cannot set parent on non-compound graph")
	}
	
	// Remove from old parent's children
	if oldParent, exists := g.parent[child]; exists {
		g.removeChild(oldParent, child)
	}
	
	// Set new parent
	g.parent[child] = parent
	
	// Add to new parent's children
	if g.children[parent] == nil {
		g.children[parent] = []string{}
	}
	g.children[parent] = append(g.children[parent], child)
	
	return nil
}

// GetParent returns the parent of a node
func (g *Graph) GetParent(node string) string {
	return g.parent[node]
}

// Children returns the children of a node
func (g *Graph) Children(node string) []string {
	return g.children[node]
}

// SetEdge adds or updates an edge
func (g *Graph) SetEdge(v, w string, attrs map[string]interface{}, name string) {
	key := g.edgeKey(v, w, name)
	
	if _, exists := g.edges[key]; !exists {
		g.edgeCount++
	}
	
	edge := &Edge{
		V:     v,
		W:     w,
		Name:  name,
		attrs: make(map[string]interface{}),
	}
	
	// Extract known attributes
	if width, ok := attrs["width"].(float64); ok {
		edge.Width = width
	} else if width, ok := attrs["width"].(int); ok {
		edge.Width = float64(width)
	}
	
	if height, ok := attrs["height"].(float64); ok {
		edge.Height = height
	} else if height, ok := attrs["height"].(int); ok {
		edge.Height = float64(height)
	}
	
	// Store remaining attributes
	for k, v := range attrs {
		if k != "width" && k != "height" {
			edge.attrs[k] = v
		}
	}
	
	g.edges[key] = edge
}

// GetEdge returns an edge
func (g *Graph) GetEdge(v, w string, name string) *Edge {
	key := g.edgeKey(v, w, name)
	return g.edges[key]
}

// Edges returns all edges
func (g *Graph) Edges() []*Edge {
	edges := make([]*Edge, 0, len(g.edges))
	for _, edge := range g.edges {
		edges = append(edges, edge)
	}
	return edges
}

// OutEdges returns all outgoing edges from a node
func (g *Graph) OutEdges(v string) []*Edge {
	var result []*Edge
	for _, edge := range g.edges {
		if edge.V == v {
			result = append(result, edge)
		}
	}
	return result
}

// InEdges returns all incoming edges to a node
func (g *Graph) InEdges(w string) []*Edge {
	var result []*Edge
	for _, edge := range g.edges {
		if edge.W == w {
			result = append(result, edge)
		}
	}
	return result
}

// RemoveNode removes a node and all its edges
func (g *Graph) RemoveNode(id string) {
	if _, exists := g.nodes[id]; !exists {
		return
	}
	
	// Remove all edges connected to this node
	for key, edge := range g.edges {
		if edge.V == id || edge.W == id {
			delete(g.edges, key)
			g.edgeCount--
		}
	}
	
	// Remove from parent's children
	if parent, exists := g.parent[id]; exists {
		g.removeChild(parent, id)
		delete(g.parent, id)
	}
	
	// Remove node
	delete(g.nodes, id)
	g.nodeCount--
}

// RemoveEdge removes an edge
func (g *Graph) RemoveEdge(v, w string, name string) {
	key := g.edgeKey(v, w, name)
	if _, exists := g.edges[key]; exists {
		delete(g.edges, key)
		g.edgeCount--
	}
}

// Helper functions

func (g *Graph) edgeKey(v, w string, name string) string {
	if g.multigraph && name != "" {
		return fmt.Sprintf("%s->%s[%s]", v, w, name)
	}
	return fmt.Sprintf("%s->%s", v, w)
}

func (g *Graph) removeChild(parent, child string) {
	children := g.children[parent]
	for i, c := range children {
		if c == child {
			g.children[parent] = append(children[:i], children[i+1:]...)
			break
		}
	}
}