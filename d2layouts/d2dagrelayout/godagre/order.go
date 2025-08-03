package godagre

import (
	"math"
	"sort"
)

// order implements crossing minimization using the barycenter heuristic
func order(g *Graph) {
	// Build layers
	layers := buildLayers(g)
	
	// Add dummy nodes for edges spanning multiple ranks
	addDummyNodes(g, layers)
	
	// Initialize node ordering within each layer
	initOrder(g, layers)
	
	// Crossing minimization iterations
	bestCC := math.MaxInt32
	bestLayers := copyLayers(layers)
	
	for i := 0; i < 24; i++ { // 4 iterations * 6 passes (3 down, 3 up)
		sweepLayerGraphs(g, layers, i)
		cc := crossingCount(g, layers)
		if cc < bestCC {
			bestCC = cc
			bestLayers = copyLayers(layers)
		}
	}
	
	// Restore best ordering
	restoreOrder(g, bestLayers)
	
	// Remove dummy nodes
	removeDummyNodes(g)
}

// buildLayers groups nodes by rank
func buildLayers(g *Graph) [][]*Node {
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
	
	return layers
}

// addDummyNodes adds dummy nodes for edges spanning multiple ranks
func addDummyNodes(g *Graph, layers [][]*Node) {
	dummyCount := 0
	var edgesToRemove []*Edge
	
	for _, edge := range g.edges {
		v := g.GetNode(edge.V)
		w := g.GetNode(edge.W)
		
		if v == nil || w == nil {
			continue
		}
		
		if math.Abs(float64(w.Rank-v.Rank)) > 1 {
			// Edge spans multiple ranks, add dummy nodes
			edgesToRemove = append(edgesToRemove, edge)
			
			prev := v
			for r := v.Rank + 1; r < w.Rank; r++ {
				// Create dummy node
				dummyID := "_d" + string(rune(dummyCount))
				dummyCount++
				
				g.SetNode(dummyID, map[string]interface{}{})
				dummy := g.GetNode(dummyID)
				dummy.Dummy = true
				dummy.Rank = r
				dummy.Width = 0
				dummy.Height = 0
				
				// Add to layer
				layers[r] = append(layers[r], dummy)
				
				// Create edge from prev to dummy
				g.SetEdge(prev.ID, dummyID, map[string]interface{}{}, "")
				
				prev = dummy
			}
			
			// Create final edge to target
			g.SetEdge(prev.ID, w.ID, map[string]interface{}{}, "")
		}
	}
	
	// Remove original long edges
	for _, edge := range edgesToRemove {
		g.RemoveEdge(edge.V, edge.W, edge.Name)
	}
}

// initOrder initializes the order of nodes within each layer
func initOrder(g *Graph, layers [][]*Node) {
	// Build in/out edge lists for each node
	for _, node := range g.nodes {
		node.In = nil
		node.Out = nil
	}
	
	for _, edge := range g.edges {
		if v := g.GetNode(edge.V); v != nil {
			v.Out = append(v.Out, edge)
		}
		if w := g.GetNode(edge.W); w != nil {
			w.In = append(w.In, edge)
		}
	}
	
	// Sort nodes in each layer by ID initially
	for _, layer := range layers {
		sort.Slice(layer, func(i, j int) bool {
			return layer[i].ID < layer[j].ID
		})
		
		// Assign initial order
		for i, node := range layer {
			node.Order = i
		}
	}
}

// sweepLayerGraphs performs crossing minimization sweeps
func sweepLayerGraphs(g *Graph, layers [][]*Node, iter int) {
	if iter%2 == 0 {
		// Even iterations: sweep down
		for i := 1; i < len(layers); i++ {
			sweepLayer(g, layers, i, true)
		}
	} else {
		// Odd iterations: sweep up
		for i := len(layers) - 2; i >= 0; i-- {
			sweepLayer(g, layers, i, false)
		}
	}
}

// sweepLayer minimizes crossings for a single layer
func sweepLayer(g *Graph, layers [][]*Node, layerIdx int, downward bool) {
	layer := layers[layerIdx]
	
	// Calculate barycenter for each node
	for _, node := range layer {
		var sum float64
		var weight float64
		
		edges := node.In
		if downward {
			edges = node.Out
		}
		
		for _, edge := range edges {
			var other *Node
			if downward {
				other = g.GetNode(edge.W)
			} else {
				other = g.GetNode(edge.V)
			}
			
			if other != nil {
				sum += float64(other.Order) * edge.Weight
				weight += edge.Weight
			}
		}
		
		if weight > 0 {
			node.Barycenter = sum / weight
			node.Weight = weight
		} else {
			// No connections, keep current position
			node.Barycenter = float64(node.Order)
			node.Weight = 0
		}
	}
	
	// Sort by barycenter
	sort.Slice(layer, func(i, j int) bool {
		if math.Abs(layer[i].Barycenter-layer[j].Barycenter) < 1e-6 {
			// Tie breaking
			return layer[i].ID < layer[j].ID
		}
		return layer[i].Barycenter < layer[j].Barycenter
	})
	
	// Update order
	for i, node := range layer {
		node.Order = i
	}
}

// crossingCount counts the number of edge crossings
func crossingCount(g *Graph, layers [][]*Node) int {
	cc := 0
	
	for i := 0; i < len(layers)-1; i++ {
		cc += bilayerCrossCount(g, layers[i], layers[i+1])
	}
	
	return cc
}

// bilayerCrossCount counts crossings between two adjacent layers
func bilayerCrossCount(g *Graph, layer1, layer2 []*Node) int {
	// Build position map for layer2
	pos2 := make(map[string]int)
	for i, node := range layer2 {
		pos2[node.ID] = i
	}
	
	count := 0
	
	// Check all pairs of edges from layer1
	for i := 0; i < len(layer1); i++ {
		for j := i + 1; j < len(layer1); j++ {
			node1 := layer1[i]
			node2 := layer1[j]
			
			// Check all edge pairs
			for _, e1 := range node1.Out {
				p1 := pos2[e1.W]
				for _, e2 := range node2.Out {
					p2 := pos2[e2.W]
					
					// Crossing if positions are inverted
					if p1 > p2 {
						count++
					}
				}
			}
		}
	}
	
	return count
}

// copyLayers creates a deep copy of the layer structure
func copyLayers(layers [][]*Node) [][]*Node {
	newLayers := make([][]*Node, len(layers))
	for i, layer := range layers {
		newLayers[i] = make([]*Node, len(layer))
		copy(newLayers[i], layer)
	}
	return newLayers
}

// restoreOrder restores the best found ordering
func restoreOrder(g *Graph, layers [][]*Node) {
	for _, layer := range layers {
		for i, node := range layer {
			node.Order = i
		}
	}
}

// removeDummyNodes removes dummy nodes after ordering
func removeDummyNodes(g *Graph) {
	var dummyNodes []string
	
	for id, node := range g.nodes {
		if node.Dummy {
			dummyNodes = append(dummyNodes, id)
		}
	}
	
	for _, id := range dummyNodes {
		// Reconnect edges through dummy
		var inEdges, outEdges []*Edge
		
		for _, edge := range g.edges {
			if edge.W == id {
				inEdges = append(inEdges, edge)
			}
			if edge.V == id {
				outEdges = append(outEdges, edge)
			}
		}
		
		// Create direct edges
		for _, inEdge := range inEdges {
			for _, outEdge := range outEdges {
				// Combine edge properties
				g.SetEdge(inEdge.V, outEdge.W, map[string]interface{}{}, "")
			}
		}
		
		// Remove dummy
		g.RemoveNode(id)
	}
}