package godagre

import (
	"math"
	"sort"
)

// edgeRouter handles sophisticated edge routing
type edgeRouter struct {
	g        *Graph
	rankDir  string
	rankSep  float64
	nodeSep  float64
	edgeSep  float64
	ranks    map[int][]*Node
}

// newEdgeRouter creates a new edge router
func newEdgeRouter(g *Graph) *edgeRouter {
	er := &edgeRouter{
		g:       g,
		rankDir: "TB",
		rankSep: 50,
		nodeSep: 50,
		edgeSep: 20,
		ranks:   make(map[int][]*Node),
	}
	
	// Get configuration
	if rd, ok := g.attrs["rankdir"].(string); ok {
		er.rankDir = rd
	}
	if rs, ok := g.attrs["ranksep"].(float64); ok {
		er.rankSep = rs
	}
	if ns, ok := g.attrs["nodesep"].(float64); ok {
		er.nodeSep = ns
	}
	if es, ok := g.attrs["edgesep"].(float64); ok {
		er.edgeSep = es
	}
	
	// Build rank information
	er.buildRanks()
	
	return er
}

// buildRanks organizes nodes by rank
func (er *edgeRouter) buildRanks() {
	for _, node := range er.g.nodes {
		er.ranks[node.Rank] = append(er.ranks[node.Rank], node)
	}
	
	// Sort nodes in each rank by position
	for _, nodes := range er.ranks {
		sort.Slice(nodes, func(i, j int) bool {
			if er.rankDir == "TB" || er.rankDir == "BT" {
				return nodes[i].X < nodes[j].X
			}
			return nodes[i].Y < nodes[j].Y
		})
	}
}

// routeAllEdges routes all edges in the graph
func (er *edgeRouter) routeAllEdges() {
	// Group edges by endpoints for bundling
	edgeGroups := er.groupEdges()
	
	// Route each edge group
	for _, edges := range edgeGroups {
		er.routeEdgeGroup(edges)
	}
}

// groupEdges groups parallel edges between same endpoints
func (er *edgeRouter) groupEdges() map[string][]*Edge {
	groups := make(map[string][]*Edge)
	
	for _, edge := range er.g.edges {
		// Create a key for the edge endpoints
		key := edge.V + "->" + edge.W
		groups[key] = append(groups[key], edge)
	}
	
	return groups
}

// routeEdgeGroup routes a group of parallel edges
func (er *edgeRouter) routeEdgeGroup(edges []*Edge) {
	if len(edges) == 0 {
		return
	}
	
	// Get the first edge as representative
	edge := edges[0]
	src := er.g.GetNode(edge.V)
	dst := er.g.GetNode(edge.W)
	
	if src == nil || dst == nil {
		return
	}
	
	// Route the main path
	mainPath := er.routeSingleEdge(src, dst)
	
	// Handle parallel edges
	if len(edges) == 1 {
		edge.Points = mainPath
	} else {
		// Distribute parallel edges
		er.distributeParallelEdges(edges, mainPath)
	}
	
	// Set label positions
	for _, e := range edges {
		if len(e.Points) >= 2 {
			mid := len(e.Points) / 2
			e.X = e.Points[mid].X
			e.Y = e.Points[mid].Y
		}
	}
}

// routeSingleEdge routes a single edge between two nodes
func (er *edgeRouter) routeSingleEdge(src, dst *Node) []Point {
	if src.Rank == dst.Rank {
		return er.routeSameRankEdge(src, dst)
	}
	return er.routeDifferentRankEdge(src, dst)
}

// routeSameRankEdge routes edges between nodes on the same rank
func (er *edgeRouter) routeSameRankEdge(src, dst *Node) []Point {
	points := []Point{}
	
	if er.rankDir == "TB" || er.rankDir == "BT" {
		// Vertical layout - route with arc
		startX, startY := src.X, src.Y
		endX, endY := dst.X, dst.Y
		
		// Determine arc direction
		arcHeight := er.rankSep / 3
		if er.rankDir == "BT" {
			arcHeight = -arcHeight
		}
		
		// Create arc points
		midX := (startX + endX) / 2
		midY := startY - arcHeight
		
		points = append(points,
			Point{X: startX, Y: startY},
			Point{X: startX, Y: midY},
			Point{X: midX, Y: midY},
			Point{X: endX, Y: midY},
			Point{X: endX, Y: endY},
		)
	} else {
		// Horizontal layout
		startX, startY := src.X, src.Y
		endX, endY := dst.X, dst.Y
		
		// Determine arc direction
		arcWidth := er.rankSep / 3
		if er.rankDir == "RL" {
			arcWidth = -arcWidth
		}
		
		// Create arc points
		midX := startX - arcWidth
		midY := (startY + endY) / 2
		
		points = append(points,
			Point{X: startX, Y: startY},
			Point{X: midX, Y: startY},
			Point{X: midX, Y: midY},
			Point{X: midX, Y: endY},
			Point{X: endX, Y: endY},
		)
	}
	
	return points
}

// routeDifferentRankEdge routes edges between nodes on different ranks
func (er *edgeRouter) routeDifferentRankEdge(src, dst *Node) []Point {
	points := []Point{}
	
	// Start from source center
	points = append(points, Point{X: src.X, Y: src.Y})
	
	if er.rankDir == "TB" || er.rankDir == "BT" {
		// Vertical layout
		er.routeVerticalEdge(src, dst, &points)
	} else {
		// Horizontal layout
		er.routeHorizontalEdge(src, dst, &points)
	}
	
	// End at destination center
	points = append(points, Point{X: dst.X, Y: dst.Y})
	
	return points
}

// routeVerticalEdge routes an edge in vertical layout
func (er *edgeRouter) routeVerticalEdge(src, dst *Node, points *[]Point) {
	startX, startY := src.X, src.Y
	endX, endY := dst.X, dst.Y
	
	// Determine direction
	dir := 1.0
	if src.Rank > dst.Rank {
		dir = -1.0
	}
	if er.rankDir == "BT" {
		dir = -dir
	}
	
	// Exit source
	exitY := startY + dir*src.Height/2
	*points = append(*points, Point{X: startX, Y: exitY})
	
	// Route through intermediate ranks
	currX := startX
	currY := exitY
	
	for r := src.Rank + int(dir); r != dst.Rank; r += int(dir) {
		// Move to rank midpoint
		rankY := er.getRankY(r)
		midY := currY + (rankY-currY)*0.5
		
		// Check if we need to adjust X
		progress := float64(r-src.Rank) / float64(dst.Rank-src.Rank)
		targetX := startX + (endX-startX)*progress
		
		if math.Abs(targetX-currX) > 1e-6 {
			// Add horizontal segment
			*points = append(*points, Point{X: currX, Y: midY})
			*points = append(*points, Point{X: targetX, Y: midY})
			currX = targetX
		} else {
			*points = append(*points, Point{X: currX, Y: midY})
		}
		
		currY = rankY
	}
	
	// Enter destination
	enterY := endY - dir*dst.Height/2
	if math.Abs(currX-endX) > 1e-6 {
		midY := currY + (enterY-currY)*0.5
		*points = append(*points, Point{X: currX, Y: midY})
		*points = append(*points, Point{X: endX, Y: midY})
	}
	*points = append(*points, Point{X: endX, Y: enterY})
}

// routeHorizontalEdge routes an edge in horizontal layout
func (er *edgeRouter) routeHorizontalEdge(src, dst *Node, points *[]Point) {
	startX, startY := src.X, src.Y
	endX, endY := dst.X, dst.Y
	
	// Determine direction
	dir := 1.0
	if src.Rank > dst.Rank {
		dir = -1.0
	}
	if er.rankDir == "RL" {
		dir = -dir
	}
	
	// Exit source
	exitX := startX + dir*src.Width/2
	*points = append(*points, Point{X: exitX, Y: startY})
	
	// Route through intermediate ranks
	currX := exitX
	currY := startY
	
	for r := src.Rank + int(dir); r != dst.Rank; r += int(dir) {
		// Move to rank midpoint
		rankX := er.getRankX(r)
		midX := currX + (rankX-currX)*0.5
		
		// Check if we need to adjust Y
		progress := float64(r-src.Rank) / float64(dst.Rank-src.Rank)
		targetY := startY + (endY-startY)*progress
		
		if math.Abs(targetY-currY) > 1e-6 {
			// Add vertical segment
			*points = append(*points, Point{X: midX, Y: currY})
			*points = append(*points, Point{X: midX, Y: targetY})
			currY = targetY
		} else {
			*points = append(*points, Point{X: midX, Y: currY})
		}
		
		currX = rankX
	}
	
	// Enter destination
	enterX := endX - dir*dst.Width/2
	if math.Abs(currY-endY) > 1e-6 {
		midX := currX + (enterX-currX)*0.5
		*points = append(*points, Point{X: midX, Y: currY})
		*points = append(*points, Point{X: midX, Y: endY})
	}
	*points = append(*points, Point{X: enterX, Y: endY})
}

// distributeParallelEdges distributes multiple edges between same endpoints
func (er *edgeRouter) distributeParallelEdges(edges []*Edge, basePath []Point) {
	n := len(edges)
	
	// Calculate offset for each edge
	totalSep := float64(n-1) * er.edgeSep
	startOffset := -totalSep / 2
	
	for i, edge := range edges {
		offset := startOffset + float64(i)*er.edgeSep
		edge.Points = er.offsetPath(basePath, offset)
	}
}

// offsetPath creates an offset version of a path
func (er *edgeRouter) offsetPath(path []Point, offset float64) []Point {
	if len(path) < 2 {
		return path
	}
	
	result := make([]Point, len(path))
	
	for i, p := range path {
		if i == 0 || i == len(path)-1 {
			// Keep endpoints unchanged
			result[i] = p
		} else {
			// Offset intermediate points
			// Calculate perpendicular direction
			var dx, dy float64
			if i < len(path)-1 {
				dx = path[i+1].X - path[i].X
				dy = path[i+1].Y - path[i].Y
			} else {
				dx = path[i].X - path[i-1].X
				dy = path[i].Y - path[i-1].Y
			}
			
			// Normalize and rotate 90 degrees
			length := math.Sqrt(dx*dx + dy*dy)
			if length > 0 {
				perpX := -dy / length
				perpY := dx / length
				
				result[i] = Point{
					X: p.X + perpX*offset,
					Y: p.Y + perpY*offset,
				}
			} else {
				result[i] = p
			}
		}
	}
	
	return result
}

// getRankY gets the Y coordinate for a rank in vertical layout
func (er *edgeRouter) getRankY(rank int) float64 {
	return float64(rank) * er.rankSep
}

// getRankX gets the X coordinate for a rank in horizontal layout
func (er *edgeRouter) getRankX(rank int) float64 {
	return float64(rank) * er.rankSep
}