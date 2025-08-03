package godagre

import (
	"math"
)

// networkSimplex implements the network simplex algorithm for rank assignment
func networkSimplex(g *Graph) {
	// Initialize the simplex graph
	simplex := initNetworkSimplex(g)
	
	// Construct initial feasible tree
	longestPath(simplex)
	feasibleTree(simplex)
	
	// Initialize edge cutvalues
	initCutValues(simplex)
	
	// Main optimization loop
	var e, f *Edge
	for e = leaveEdge(simplex); e != nil; e = leaveEdge(simplex) {
		f = enterEdge(simplex, e)
		exchangeEdges(simplex, e, f)
	}
	
	// Normalize ranks
	normalize(simplex)
	
	// Copy ranks back to original graph
	for _, node := range g.nodes {
		if sNode := simplex.GetNode(node.ID); sNode != nil {
			node.Rank = sNode.Rank
		}
	}
	
	// Update graph rank bounds
	updateRankBounds(g)
}

// initNetworkSimplex creates a simplified graph for the network simplex algorithm
func initNetworkSimplex(g *Graph) *Graph {
	sg := NewGraph(GraphOptions{
		Directed: true,
	})
	
	// Copy nodes
	for id, node := range g.nodes {
		attrs := map[string]interface{}{}
		sg.SetNode(id, attrs)
		sNode := sg.GetNode(id)
		sNode.Width = node.Width
		sNode.Height = node.Height
	}
	
	// Copy edges with weight and minlen
	for _, edge := range g.edges {
		weight := 1.0
		if w, ok := edge.attrs["weight"].(float64); ok {
			weight = w
		} else if w, ok := edge.attrs["weight"].(int); ok {
			weight = float64(w)
		}
		
		minlen := 1
		if ml, ok := edge.attrs["minlen"].(int); ok {
			minlen = ml
		}
		
		sg.SetEdge(edge.V, edge.W, map[string]interface{}{}, edge.Name)
		sEdge := sg.GetEdge(edge.V, edge.W, edge.Name)
		if sEdge != nil {
			sEdge.Weight = weight
			sEdge.Minlen = minlen
		}
	}
	
	return sg
}

// longestPath assigns initial ranks using longest path algorithm
func longestPath(g *Graph) {
	// Reset all ranks
	for _, node := range g.nodes {
		node.Rank = 0
	}
	
	// Compute longest paths
	changed := true
	for changed {
		changed = false
		for _, edge := range g.edges {
			v := g.GetNode(edge.V)
			w := g.GetNode(edge.W)
			if v != nil && w != nil {
				expectedRank := v.Rank + edge.Minlen
				if w.Rank < expectedRank {
					w.Rank = expectedRank
					changed = true
				}
			}
		}
	}
}

// feasibleTree builds an initial feasible spanning tree
func feasibleTree(g *Graph) {
	// Start with an empty tree
	for _, edge := range g.edges {
		edge.Tree = false
		edge.Cutvalue = 0
	}
	
	// Build spanning tree using DFS
	visited := make(map[string]bool)
	
	var dfs func(v string)
	dfs = func(v string) {
		visited[v] = true
		node := g.GetNode(v)
		
		// Process outgoing edges
		for _, edge := range g.edges {
			if edge.V == v && !visited[edge.W] {
				edge.Tree = true
				w := g.GetNode(edge.W)
				w.Parent = v
				w.Low = node.Lim + 1
				w.Lim = w.Low
				dfs(edge.W)
				node.Lim = w.Lim
			}
		}
	}
	
	// Find roots and start DFS
	roots := findRoots(g)
	for i, root := range roots {
		if !visited[root] {
			node := g.GetNode(root)
			node.Low = i * 10000 // Separate trees
			node.Lim = node.Low
			dfs(root)
		}
	}
}

// findRoots finds nodes with no incoming edges
func findRoots(g *Graph) []string {
	hasIncoming := make(map[string]bool)
	for _, edge := range g.edges {
		hasIncoming[edge.W] = true
	}
	
	var roots []string
	for id := range g.nodes {
		if !hasIncoming[id] {
			roots = append(roots, id)
		}
	}
	
	// If no roots (cyclic), use arbitrary node
	if len(roots) == 0 && len(g.nodes) > 0 {
		for id := range g.nodes {
			roots = append(roots, id)
			break
		}
	}
	
	return roots
}

// initCutValues initializes cut values for all tree edges
func initCutValues(g *Graph) {
	// Calculate cut values for tree edges
	for _, edge := range g.edges {
		if edge.Tree {
			edge.Cutvalue = calcCutValue(g, edge)
		}
	}
}

// calcCutValue calculates the cut value for a tree edge
func calcCutValue(g *Graph, edge *Edge) float64 {
	v := g.GetNode(edge.V)
	w := g.GetNode(edge.W)
	
	// Determine which side is the tail component
	var tailNode *Node
	if v.Lim < w.Lim && w.Lim <= v.Lim + (w.Lim - w.Low + 1) {
		tailNode = w
	} else {
		tailNode = v
	}
	
	// Sum weights of edges crossing the cut
	cutvalue := 0.0
	
	for _, e := range g.edges {
		vNode := g.GetNode(e.V)
		wNode := g.GetNode(e.W)
		
		vInTail := tailNode.Low <= vNode.Lim && vNode.Lim <= tailNode.Lim
		wInTail := tailNode.Low <= wNode.Lim && wNode.Lim <= tailNode.Lim
		
		// Edge crosses cut if one endpoint is in tail and other is not
		if vInTail != wInTail {
			if vInTail == (tailNode == w) {
				// Forward edge
				cutvalue += e.Weight
			} else {
				// Backward edge
				cutvalue -= e.Weight
			}
		}
	}
	
	return cutvalue
}

// leaveEdge finds the tree edge with minimum cut value to remove
func leaveEdge(g *Graph) *Edge {
	var minEdge *Edge
	minCutvalue := math.Inf(1)
	
	for _, edge := range g.edges {
		if edge.Tree && edge.Cutvalue < minCutvalue {
			minCutvalue = edge.Cutvalue
			minEdge = edge
		}
	}
	
	// Only return edge if it has negative cut value
	if minCutvalue < -1e-6 {
		return minEdge
	}
	
	return nil
}

// enterEdge finds the non-tree edge to replace the leaving edge
func enterEdge(g *Graph, leave *Edge) *Edge {
	v := g.GetNode(leave.V)
	w := g.GetNode(leave.W)
	
	// Determine tail component
	var tailNode *Node
	if v.Lim < w.Lim && w.Lim <= v.Lim + (w.Lim - w.Low + 1) {
		tailNode = w
	} else {
		tailNode = v
	}
	
	// Find best entering edge
	var bestEdge *Edge
	var bestSlack int = math.MaxInt32
	
	for _, edge := range g.edges {
		if !edge.Tree {
			vNode := g.GetNode(edge.V)
			wNode := g.GetNode(edge.W)
			
			vInTail := tailNode.Low <= vNode.Lim && vNode.Lim <= tailNode.Lim
			wInTail := tailNode.Low <= wNode.Lim && wNode.Lim <= tailNode.Lim
			
			// Edge must cross the cut
			if vInTail != wInTail {
				slack := wNode.Rank - vNode.Rank - edge.Minlen
				if slack < bestSlack {
					bestSlack = slack
					bestEdge = edge
				}
			}
		}
	}
	
	return bestEdge
}

// exchangeEdges swaps the leaving edge with entering edge and updates the tree
func exchangeEdges(g *Graph, leave, enter *Edge) {
	leave.Tree = false
	enter.Tree = true
	
	// Update ranks to maintain feasibility
	vNode := g.GetNode(enter.V)
	wNode := g.GetNode(enter.W)
	
	// Calculate rank adjustment
	delta := wNode.Rank - vNode.Rank - enter.Minlen
	
	// Update ranks in affected component
	updateRanks(g, enter, delta)
	
	// Rebuild tree structure
	updateTreeStructure(g)
	
	// Recalculate cut values
	initCutValues(g)
}

// updateRanks adjusts ranks after edge exchange
func updateRanks(g *Graph, enter *Edge, delta int) {
	// Determine which component to update based on tree structure
	v := g.GetNode(enter.V)
	w := g.GetNode(enter.W)
	
	// Find component to update (simplified)
	updateComponent := make(map[string]bool)
	if v.Parent == "" || w.Parent == "" {
		// Update w's component
		var collect func(string)
		collect = func(id string) {
			updateComponent[id] = true
			for _, child := range g.nodes {
				if child.Parent == id {
					collect(child.ID)
				}
			}
		}
		collect(w.ID)
	}
	
	// Apply rank adjustment
	for id := range updateComponent {
		if node := g.GetNode(id); node != nil {
			node.Rank -= delta
		}
	}
}

// updateTreeStructure rebuilds parent pointers and low/lim values
func updateTreeStructure(g *Graph) {
	// Reset parent pointers
	for _, node := range g.nodes {
		node.Parent = ""
		node.Low = 0
		node.Lim = 0
	}
	
	// Rebuild from tree edges
	visited := make(map[string]bool)
	postorder := 0
	
	var dfs func(v string)
	dfs = func(v string) {
		visited[v] = true
		node := g.GetNode(v)
		node.Low = postorder
		
		for _, edge := range g.edges {
			if edge.Tree && edge.V == v && !visited[edge.W] {
				w := g.GetNode(edge.W)
				w.Parent = v
				dfs(edge.W)
			}
		}
		
		node.Lim = postorder
		postorder++
	}
	
	// Process from roots
	roots := findRoots(g)
	for _, root := range roots {
		if !visited[root] {
			dfs(root)
		}
	}
}

// normalize adjusts ranks to start from 0
func normalize(g *Graph) {
	minRank := math.MaxInt32
	for _, node := range g.nodes {
		if node.Rank < minRank {
			minRank = node.Rank
		}
	}
	
	for _, node := range g.nodes {
		node.Rank -= minRank
	}
}

// updateRankBounds updates the min/max rank values in the graph
func updateRankBounds(g *Graph) {
	g.minRank = math.MaxInt32
	g.maxRank = math.MinInt32
	
	for _, node := range g.nodes {
		if node.Rank < g.minRank {
			g.minRank = node.Rank
		}
		if node.Rank > g.maxRank {
			g.maxRank = node.Rank
		}
	}
}