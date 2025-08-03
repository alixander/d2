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
	
	// Phase 1: Make the graph acyclic by reversing edges
	reversedEdges := makeAcyclic(g)
	
	// Phase 2: Assign ranks (vertical levels) to nodes
	assignRanks(g)
	
	// Phase 3: Order nodes within ranks to minimize crossings
	orderNodes(g)
	
	// Phase 4: Assign positions to nodes
	assignPositions(g)
	
	// Phase 4.5: Adjust container sizes and positions
	adjustContainerSizes(g)
	
	// Phase 5: Route edges
	routeEdges(g)
	
	// Restore reversed edges
	for _, e := range reversedEdges {
		// Swap source and target back
		e.V, e.W = e.W, e.V
		// Reverse the points
		for i, j := 0, len(e.Points)-1; i < j; i, j = i+1, j-1 {
			e.Points[i], e.Points[j] = e.Points[j], e.Points[i]
		}
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

// assignRanks assigns vertical levels to nodes
func assignRanks(g *Graph) {
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
}

// orderNodes orders nodes within each rank to minimize edge crossings
func orderNodes(g *Graph) {
	// Group nodes by rank
	ranks := make(map[int][]*Node)
	maxRank := 0
	
	for _, id := range g.Nodes() {
		node := g.GetNode(id)
		if node.Rank > maxRank {
			maxRank = node.Rank
		}
		ranks[node.Rank] = append(ranks[node.Rank], node)
	}
	
	// Simple ordering: maintain relative order within each rank
	for r := 0; r <= maxRank; r++ {
		nodes := ranks[r]
		for i, node := range nodes {
			node.Order = i
		}
	}
	
	// TODO: Implement crossing minimization algorithm
	// For now, we just use the initial order
}

// assignPositions assigns x,y coordinates to nodes
func assignPositions(g *Graph) {
	// Group nodes by rank
	ranks := make(map[int][]*Node)
	maxRank := 0
	
	for _, id := range g.Nodes() {
		node := g.GetNode(id)
		if node.Rank > maxRank {
			maxRank = node.Rank
		}
		ranks[node.Rank] = append(ranks[node.Rank], node)
	}
	
	nodeSep := g.GetGraph("nodesep").(float64)
	rankSep := g.GetGraph("ranksep").(float64)
	rankDir := g.GetGraph("rankdir").(string)
	
	// Assign positions based on rank and order
	for r := 0; r <= maxRank; r++ {
		nodes := ranks[r]
		
		// Sort by order
		for i := 0; i < len(nodes)-1; i++ {
			for j := i + 1; j < len(nodes); j++ {
				if nodes[i].Order > nodes[j].Order {
					nodes[i], nodes[j] = nodes[j], nodes[i]
				}
			}
		}
		
		// Assign positions
		x := 0.0
		for _, node := range nodes {
			switch rankDir {
			case "TB", "BT":
				node.X = x + node.Width/2
				node.Y = float64(r)*rankSep + node.Height/2
				x += node.Width + nodeSep
			case "LR", "RL":
				node.Y = x + node.Height/2
				node.X = float64(r)*rankSep + node.Width/2
				x += node.Height + nodeSep
			}
		}
		
		// Center the rank
		if len(nodes) > 0 {
			totalWidth := x - nodeSep
			offset := -totalWidth / 2
			for _, node := range nodes {
				switch rankDir {
				case "TB", "BT":
					node.X += offset
				case "LR", "RL":
					node.Y += offset
				}
			}
		}
	}
	
	// Handle rank direction
	switch rankDir {
	case "BT":
		// Bottom to top - flip Y coordinates
		maxY := 0.0
		for _, id := range g.Nodes() {
			node := g.GetNode(id)
			if node.Y > maxY {
				maxY = node.Y
			}
		}
		for _, id := range g.Nodes() {
			node := g.GetNode(id)
			node.Y = maxY - node.Y
		}
	case "RL":
		// Right to left - flip X coordinates
		maxX := 0.0
		for _, id := range g.Nodes() {
			node := g.GetNode(id)
			if node.X > maxX {
				maxX = node.X
			}
		}
		for _, id := range g.Nodes() {
			node := g.GetNode(id)
			node.X = maxX - node.X
		}
	}
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
	// First, adjust container sizes to fit their children
	adjustContainerSizes(g)
	
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

// adjustContainerSizes adjusts container node sizes to fit their children
func adjustContainerSizes(g *Graph) {
	if !g.compound {
		return
	}
	
	// Process nodes in reverse topological order (children before parents)
	// Build a list of all parent nodes
	parents := make(map[string]bool)
	for child := range g.parent {
		if parent := g.parent[child]; parent != "" {
			parents[parent] = true
		}
	}
	
	// For each parent, calculate bounding box of children
	for parentID := range parents {
		parent := g.GetNode(parentID)
		if parent == nil {
			continue
		}
		
		minX, minY := math.Inf(1), math.Inf(1)
		maxX, maxY := math.Inf(-1), math.Inf(-1)
		hasChildren := false
		
		// Find bounds of all children
		for childID, p := range g.parent {
			if p == parentID {
				child := g.GetNode(childID)
				if child != nil {
					hasChildren = true
					minX = math.Min(minX, child.X-child.Width/2)
					maxX = math.Max(maxX, child.X+child.Width/2)
					minY = math.Min(minY, child.Y-child.Height/2)
					maxY = math.Max(maxY, child.Y+child.Height/2)
				}
			}
		}
		
		if hasChildren {
			// Add padding
			padding := 30.0
			minX -= padding
			minY -= padding
			maxX += padding
			maxY += padding
			
			// Update parent size and position
			parent.X = (minX + maxX) / 2
			parent.Y = (minY + maxY) / 2
			parent.Width = maxX - minX
			parent.Height = maxY - minY
		}
	}
}