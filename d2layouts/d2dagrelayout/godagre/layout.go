package godagre

import (
	"math"
)

// LayoutOptions configures the layout algorithm
type LayoutOptions struct {
	// NodeSep is the separation between nodes in the same rank
	NodeSep float64
	// EdgeSep is the separation between edges
	EdgeSep float64
	// RankSep is the separation between ranks
	RankSep float64
	// RankDir is the direction of the layout: TB, BT, LR, RL
	RankDir string
	// Align is the alignment of nodes: UL, UR, DL, DR
	Align string
	// Ranker is the ranking algorithm: network-simplex, tight-tree, longest-path
	Ranker string
	// Acyclicer is the algorithm to break cycles: greedy
	Acyclicer string
}

// DefaultLayoutOptions returns sensible defaults
func DefaultLayoutOptions() LayoutOptions {
	return LayoutOptions{
		NodeSep:   50,
		EdgeSep:   20,
		RankSep:   50,
		RankDir:   "TB",
		Align:     "UL",
		Ranker:    "network-simplex",
		Acyclicer: "greedy",
	}
}

// Layout performs the dagre layout algorithm on the graph
func Layout(g *Graph, opts LayoutOptions) error {
	// Store options in graph
	g.SetGraph(map[string]interface{}{
		"nodesep":   opts.NodeSep,
		"edgesep":   opts.EdgeSep,
		"ranksep":   opts.RankSep,
		"rankdir":   opts.RankDir,
		"align":     opts.Align,
		"ranker":    opts.Ranker,
		"acyclicer": opts.Acyclicer,
	})
	
	// Pre-process compound graphs if needed
	if g.compound {
		// Adjust container dimensions before layout
		adjustContainerDimensions(g)
	}
	
	// Phase 1: Make the graph acyclic by reversing edges
	reversedEdges := makeAcyclic(g)
	
	// Phase 2: Assign ranks using network simplex
	switch opts.Ranker {
	case "network-simplex":
		networkSimplex(g)
	case "longest-path":
		longestPathRanking(g)
	default:
		networkSimplex(g)
	}
	
	// Phase 3: Order nodes within ranks to minimize crossings
	order(g)
	
	// Phase 4: Assign positions to nodes
	position(g)
	
	// Phase 5: Route edges
	edgeRouter := newEdgeRouter(g)
	edgeRouter.routeAllEdges()
	
	// Restore reversed edges
	for _, e := range reversedEdges {
		// Swap source and target back
		e.V, e.W = e.W, e.V
		// Reverse the points
		for i, j := 0, len(e.Points)-1; i < j; i, j = i+1, j-1 {
			e.Points[i], e.Points[j] = e.Points[j], e.Points[i]
		}
	}
	
	// Post-process compound graphs
	if g.compound {
		postProcessCompoundGraph(g)
	}
	
	// Calculate graph dimensions
	calculateGraphDimensions(g)
	
	return nil
}

// makeAcyclic removes cycles from the graph by reversing edges
func makeAcyclic(g *Graph) []*Edge {
	var reversedEdges []*Edge
	
	// Simple greedy algorithm: do a DFS and reverse back edges
	visited := make(map[string]bool)
	onStack := make(map[string]bool)
	
	var dfs func(v string)
	dfs = func(v string) {
		visited[v] = true
		onStack[v] = true
		
		for _, edge := range g.OutEdges(v) {
			w := edge.W
			if !visited[w] {
				dfs(w)
			} else if onStack[w] {
				// Back edge found - reverse it
				edge.V, edge.W = edge.W, edge.V
				reversedEdges = append(reversedEdges, edge)
			}
		}
		
		onStack[v] = false
	}
	
	// Run DFS from all unvisited nodes
	for _, v := range g.Nodes() {
		if !visited[v] {
			dfs(v)
		}
	}
	
	return reversedEdges
}

// longestPathRanking assigns ranks using longest path algorithm
func longestPathRanking(g *Graph) {
	// Simple longest path algorithm
	rank := make(map[string]int)
	
	// Initialize all ranks to 0
	for _, v := range g.Nodes() {
		rank[v] = 0
	}
	
	// Keep updating ranks until stable
	changed := true
	for changed {
		changed = false
		for _, edge := range g.Edges() {
			if rank[edge.W] <= rank[edge.V] {
				rank[edge.W] = rank[edge.V] + 1
				changed = true
			}
		}
	}
	
	// Update node ranks
	for id, r := range rank {
		if node := g.GetNode(id); node != nil {
			node.Rank = r
		}
	}
	
	// Update rank bounds
	updateRankBounds(g)
}



// routeEdges creates edge paths
func routeEdges(g *Graph) {
	rankDir := g.GetGraph("rankdir").(string)
	
	for _, edge := range g.Edges() {
		src := g.GetNode(edge.V)
		dst := g.GetNode(edge.W)
		
		if src == nil || dst == nil {
			continue
		}
		
		// Create multi-point routes for edges between different ranks
		if src.Rank != dst.Rank {
			// For edges spanning multiple ranks, create intermediate points
			points := []Point{}
			
			// Start from source center
			startX, startY := src.X, src.Y
			endX, endY := dst.X, dst.Y
			
			// Add start point
			points = append(points, Point{X: startX, Y: startY})
			
			// For vertical layouts (TB/BT), route edges with intermediate points
			if rankDir == "TB" || rankDir == "BT" {
				
				if src.Rank < dst.Rank {
					// Going down - add intermediate points
					// Exit source at bottom
					exitY := startY + src.Height/2 + 10
					// Enter destination at top  
					enterY := endY - dst.Height/2 - 10
					// Mid point between shapes
					midY := (exitY + enterY) / 2
					
					points = append(points, Point{X: startX, Y: exitY})
					points = append(points, Point{X: startX, Y: midY})
					points = append(points, Point{X: endX, Y: midY})
					points = append(points, Point{X: endX, Y: enterY})
				} else {
					// Going up - add intermediate points  
					exitY := startY - src.Height/2 - 10
					enterY := endY + dst.Height/2 + 10
					midY := (exitY + enterY) / 2
					
					points = append(points, Point{X: startX, Y: exitY})
					points = append(points, Point{X: startX, Y: midY})
					points = append(points, Point{X: endX, Y: midY})
					points = append(points, Point{X: endX, Y: enterY})
				}
			} else {
				// Horizontal layouts (LR/RL)
				if src.Rank < dst.Rank {
					// Going right - add intermediate points
					exitX := startX + src.Width/2 + 10
					enterX := endX - dst.Width/2 - 10
					midX := (exitX + enterX) / 2
					
					points = append(points, Point{X: exitX, Y: startY})
					points = append(points, Point{X: midX, Y: startY})
					points = append(points, Point{X: midX, Y: endY})
					points = append(points, Point{X: enterX, Y: endY})
				} else {
					// Going left - add intermediate points
					exitX := startX - src.Width/2 - 10
					enterX := endX + dst.Width/2 + 10
					midX := (exitX + enterX) / 2
					
					points = append(points, Point{X: exitX, Y: startY})
					points = append(points, Point{X: midX, Y: startY})
					points = append(points, Point{X: midX, Y: endY})
					points = append(points, Point{X: enterX, Y: endY})
				}
			}
			
			// Add end point
			points = append(points, Point{X: endX, Y: endY})
			
			edge.Points = points
		} else {
			// Same rank - simple direct connection
			edge.Points = []Point{
				{X: src.X, Y: src.Y},
				{X: dst.X, Y: dst.Y},
			}
		}
		
		// Set label position at midpoint
		edge.X = (src.X + dst.X) / 2
		edge.Y = (src.Y + dst.Y) / 2
	}
}

// calculateGraphDimensions calculates the overall graph dimensions
func calculateGraphDimensions(g *Graph) {
	
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	
	for _, id := range g.Nodes() {
		node := g.GetNode(id)
		minX = math.Min(minX, node.X-node.Width/2)
		maxX = math.Max(maxX, node.X+node.Width/2)
		minY = math.Min(minY, node.Y-node.Height/2)
		maxY = math.Max(maxY, node.Y+node.Height/2)
	}
	
	// Translate graph so all coordinates are positive
	if minX < 0 || minY < 0 {
		padding := 10.0
		dx := -minX + padding
		dy := -minY + padding
		
		// Translate all nodes
		for _, id := range g.Nodes() {
			node := g.GetNode(id)
			node.X += dx
			node.Y += dy
		}
		
		// Translate all edge points
		for _, edge := range g.Edges() {
			for _, p := range edge.Points {
				p.X += dx
				p.Y += dy
			}
			edge.X += dx
			edge.Y += dy
		}
		
		// Update bounds
		maxX += dx
		maxY += dy
		minX = padding
		minY = padding
	}
	
	g.SetGraph(map[string]interface{}{
		"width":  maxX - minX,
		"height": maxY - minY,
	})
}