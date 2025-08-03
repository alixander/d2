package godagre

import (
	"math"
	"sort"
)

// position assigns x-coordinates using Brandes-Köpf algorithm
func position(g *Graph) {
	// Get graph configuration
	rankSep := 50.0
	nodeSep := 50.0
	rankDir := "TB"
	
	if rs, ok := g.attrs["ranksep"].(float64); ok {
		rankSep = rs
	}
	if ns, ok := g.attrs["nodesep"].(float64); ok {
		nodeSep = ns
	}
	if rd, ok := g.attrs["rankdir"].(string); ok {
		rankDir = rd
	}
	
	// Build layer structure
	layers := buildLayerMatrix(g)
	
	// Run four alignments and take average
	xs := make([]map[string]float64, 4)
	
	// Top-left alignment
	xs[0] = horizontalCompaction(g, layers, true, true, nodeSep)
	
	// Top-right alignment  
	xs[1] = horizontalCompaction(g, layers, true, false, nodeSep)
	
	// Bottom-left alignment
	xs[2] = horizontalCompaction(g, layers, false, true, nodeSep)
	
	// Bottom-right alignment
	xs[3] = horizontalCompaction(g, layers, false, false, nodeSep)
	
	// Average positions
	finalX := make(map[string]float64)
	for id := range g.nodes {
		sum := 0.0
		count := 0
		for i := 0; i < 4; i++ {
			if x, ok := xs[i][id]; ok {
				sum += x
				count++
			}
		}
		if count > 0 {
			finalX[id] = sum / float64(count)
		}
	}
	
	// Assign final positions
	for id, node := range g.nodes {
		node.X = finalX[id]
		
		// Y position based on rank
		if rankDir == "TB" || rankDir == "BT" {
			node.Y = float64(node.Rank) * rankSep
		} else {
			node.X = float64(node.Rank) * rankSep
			node.Y = finalX[id]
		}
	}
	
	// Handle rank direction
	if rankDir == "BT" || rankDir == "RL" {
		flipCoordinates(g, rankDir)
	}
}

// buildLayerMatrix builds a matrix of nodes organized by rank and order
func buildLayerMatrix(g *Graph) [][]*Node {
	maxRank := 0
	for _, node := range g.nodes {
		if node.Rank > maxRank {
			maxRank = node.Rank
		}
	}
	
	layers := make([][]*Node, maxRank+1)
	for _, node := range g.nodes {
		layers[node.Rank] = append(layers[node.Rank], node)
	}
	
	// Sort by order within each layer
	for _, layer := range layers {
		sort.Slice(layer, func(i, j int) bool {
			return layer[i].Order < layer[j].Order
		})
	}
	
	return layers
}

// horizontalCompaction assigns x-coordinates with given alignment
func horizontalCompaction(g *Graph, layers [][]*Node, topAlign, leftAlign bool, nodeSep float64) map[string]float64 {
	// Initialize data structures
	root := make(map[string]string)
	align := make(map[string]string)
	pos := make(map[string]float64)
	shift := make(map[string]float64)
	sink := make(map[string]string)
	
	// Initialize root and align
	for _, node := range g.nodes {
		root[node.ID] = node.ID
		align[node.ID] = node.ID
		sink[node.ID] = node.ID
		shift[node.ID] = 0
	}
	
	// Vertical alignment
	if topAlign {
		// Top to bottom
		for i := 1; i < len(layers); i++ {
			verticalAlignment(g, layers[i-1], layers[i], root, align, pos, leftAlign)
		}
	} else {
		// Bottom to top
		for i := len(layers) - 2; i >= 0; i-- {
			verticalAlignment(g, layers[i+1], layers[i], root, align, pos, leftAlign)
		}
	}
	
	// Horizontal compaction
	xs := make(map[string]float64)
	
	// Process each layer
	for _, layer := range layers {
		// Separate into blocks
		blocks := make(map[string][]*Node)
		for _, v := range layer {
			r := root[v.ID]
			blocks[r] = append(blocks[r], v)
		}
		
		// Place blocks
		x := 0.0
		orderedRoots := make([]string, 0, len(blocks))
		for r := range blocks {
			orderedRoots = append(orderedRoots, r)
		}
		
		// Sort roots by leftmost node order
		sort.Slice(orderedRoots, func(i, j int) bool {
			minI, minJ := math.MaxInt32, math.MaxInt32
			for _, v := range blocks[orderedRoots[i]] {
				if v.Order < minI {
					minI = v.Order
				}
			}
			for _, v := range blocks[orderedRoots[j]] {
				if v.Order < minJ {
					minJ = v.Order
				}
			}
			return minI < minJ
		})
		
		// Assign positions
		for _, r := range orderedRoots {
			block := blocks[r]
			
			// Sort block by order
			sort.Slice(block, func(i, j int) bool {
				return block[i].Order < block[j].Order
			})
			
			// Position nodes in block
			for _, v := range block {
				xs[v.ID] = x + shift[v.ID]
				x += v.Width + nodeSep
			}
		}
	}
	
	return xs
}

// verticalAlignment creates vertical alignment between layers
func verticalAlignment(g *Graph, layer1, layer2 []*Node, root, align map[string]string, 
	pos map[string]float64, leftAlign bool) {
	
	// Build position maps
	pos1 := make(map[string]int)
	pos2 := make(map[string]int)
	
	for i, v := range layer1 {
		pos1[v.ID] = i
	}
	for i, v := range layer2 {
		pos2[v.ID] = i
	}
	
	// Process nodes in layer2
	for _, v := range layer2 {
		// Find median neighbor
		neighbors := findNeighbors(g, v, layer1)
		
		if len(neighbors) == 0 {
			continue
		}
		
		// Sort neighbors by position
		sort.Slice(neighbors, func(i, j int) bool {
			return pos1[neighbors[i].ID] < pos1[neighbors[j].ID]
		})
		
		// Select median
		var u *Node
		if leftAlign {
			u = neighbors[0]
		} else {
			u = neighbors[len(neighbors)-1]
		}
		
		// Create alignment
		align[v.ID] = u.ID
		root[v.ID] = root[u.ID]
		
		// Update position
		if leftAlign {
			pos[v.ID] = pos[u.ID]
		} else {
			pos[v.ID] = pos[u.ID] + u.Width - v.Width
		}
	}
}

// findNeighbors finds connected nodes in the other layer
func findNeighbors(g *Graph, node *Node, otherLayer []*Node) []*Node {
	neighborSet := make(map[string]bool)
	
	// Check incoming edges
	for _, edge := range node.In {
		neighborSet[edge.V] = true
	}
	
	// Check outgoing edges
	for _, edge := range node.Out {
		neighborSet[edge.W] = true
	}
	
	// Filter to nodes in other layer
	var neighbors []*Node
	for _, other := range otherLayer {
		if neighborSet[other.ID] {
			neighbors = append(neighbors, other)
		}
	}
	
	return neighbors
}

// flipCoordinates handles bottom-up and right-left layouts
func flipCoordinates(g *Graph, rankDir string) {
	switch rankDir {
	case "BT":
		// Flip Y coordinates
		maxY := 0.0
		for _, node := range g.nodes {
			if node.Y > maxY {
				maxY = node.Y
			}
		}
		for _, node := range g.nodes {
			node.Y = maxY - node.Y
		}
		
	case "RL":
		// Flip X coordinates
		maxX := 0.0
		for _, node := range g.nodes {
			if node.X > maxX {
				maxX = node.X
			}
		}
		for _, node := range g.nodes {
			node.X = maxX - node.X
		}
	}
}