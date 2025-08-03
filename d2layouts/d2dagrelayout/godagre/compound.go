package godagre

import (
	"math"
)

// processCompoundGraph handles special processing for compound graphs
func processCompoundGraph(g *Graph) {
	if !g.compound {
		return
	}
	
	// Phase 1: Remove edges to/from compound nodes
	collapsedEdges := collapseEdgesToCompounds(g)
	
	// Phase 2: Adjust container dimensions before layout
	adjustContainerDimensions(g)
	
	// After layout, restore edges
	defer restoreCollapsedEdges(g, collapsedEdges)
}

// collapseEdgesToCompounds redirects edges to/from containers to their border nodes
func collapseEdgesToCompounds(g *Graph) map[string]*Edge {
	collapsed := make(map[string]*Edge)
	
	// Find compound nodes (nodes with children)
	compounds := make(map[string]bool)
	for child, parent := range g.parent {
		if parent != "" {
			compounds[parent] = true
		}
		_ = child // avoid unused variable
	}
	
	// Process edges
	var toRemove []string
	for key, edge := range g.edges {
		srcIsCompound := compounds[edge.V]
		dstIsCompound := compounds[edge.W]
		
		if srcIsCompound || dstIsCompound {
			// Store original edge
			collapsed[key] = &Edge{
				V: edge.V,
				W: edge.W,
				Name: edge.Name,
				Weight: edge.Weight,
				Minlen: edge.Minlen,
			}
			
			// Redirect to border node
			newV, newW := edge.V, edge.W
			
			if srcIsCompound {
				// Find bottommost child
				newV = findBorderNode(g, edge.V, edge.W, true)
			}
			
			if dstIsCompound {
				// Find topmost child
				newW = findBorderNode(g, edge.W, edge.V, false)
			}
			
			if newV != edge.V || newW != edge.W {
				// Update edge
				edge.V = newV
				edge.W = newW
				
				// Mark for removal and re-add with new key
				toRemove = append(toRemove, key)
			}
		}
	}
	
	// Re-key edges that were redirected
	for _, key := range toRemove {
		edge := g.edges[key]
		delete(g.edges, key)
		newKey := g.edgeKey(edge.V, edge.W, edge.Name)
		g.edges[newKey] = edge
	}
	
	return collapsed
}

// findBorderNode finds the appropriate border node for compound edge routing
func findBorderNode(g *Graph, compound, other string, isSource bool) string {
	children := g.children[compound]
	if len(children) == 0 {
		return compound
	}
	
	// For now, return first/last child based on direction
	// In full dagre, this uses more sophisticated logic
	if isSource {
		return children[len(children)-1]
	}
	return children[0]
}

// restoreCollapsedEdges restores original compound edges after layout
func restoreCollapsedEdges(g *Graph, collapsed map[string]*Edge) {
	for key, original := range collapsed {
		if edge, exists := g.edges[key]; exists {
			edge.V = original.V
			edge.W = original.W
		}
	}
}

// adjustContainerDimensions ensures containers are large enough for their children
func adjustContainerDimensions(g *Graph) {
	// Build parent-child hierarchy
	hierarchy := buildHierarchy(g)
	
	// Process from leaves up
	adjustDimensionsRecursive(g, hierarchy, "")
	
	// Also ensure all nodes have their parent set in the graph structure
	for nodeID, parentID := range g.parent {
		if node := g.GetNode(nodeID); node != nil {
			node.Parent = parentID
		}
	}
}

// buildHierarchy creates a tree structure of the compound graph
func buildHierarchy(g *Graph) map[string][]string {
	hierarchy := make(map[string][]string)
	
	// Find all parent-child relationships
	for child, parent := range g.parent {
		if parent == "" {
			// Root level node
			hierarchy[""] = append(hierarchy[""], child)
		} else {
			hierarchy[parent] = append(hierarchy[parent], child)
		}
	}
	
	// Add nodes without parents or children
	for id := range g.nodes {
		if _, hasParent := g.parent[id]; !hasParent {
			if _, hasChildren := hierarchy[id]; !hasChildren {
				hierarchy[""] = append(hierarchy[""], id)
			}
		}
	}
	
	return hierarchy
}

// adjustDimensionsRecursive recursively adjusts container dimensions
func adjustDimensionsRecursive(g *Graph, hierarchy map[string][]string, nodeID string) (minWidth, minHeight float64) {
	children := hierarchy[nodeID]
	
	if nodeID != "" && len(children) == 0 {
		// Leaf node
		node := g.GetNode(nodeID)
		if node != nil {
			return node.Width, node.Height
		}
		return 0, 0
	}
	
	// Process children first
	totalWidth := 0.0
	maxHeight := 0.0
	childCount := 0
	
	for _, childID := range children {
		childWidth, childHeight := adjustDimensionsRecursive(g, hierarchy, childID)
		totalWidth += childWidth
		if childHeight > maxHeight {
			maxHeight = childHeight
		}
		childCount++
	}
	
	if nodeID != "" {
		// This is a container
		node := g.GetNode(nodeID)
		if node != nil {
			// Add padding
			padding := 30.0
			nodeSep := 50.0
			
			// Calculate minimum dimensions
			minWidth = totalWidth + float64(childCount-1)*nodeSep + 2*padding
			minHeight = maxHeight + 2*padding
			
			// Ensure container is at least as large as minimum
			if node.Width < minWidth {
				node.Width = minWidth
			}
			if node.Height < minHeight {
				node.Height = minHeight
			}
			
			return node.Width, node.Height
		}
	}
	
	return totalWidth, maxHeight
}

// postProcessCompoundGraph adjusts positions after layout for compound graphs
func postProcessCompoundGraph(g *Graph) {
	if !g.compound {
		return
	}
	
	// Recalculate container positions based on children
	// This should be done first to ensure containers encompass their children
	recalculateContainerPositions(g)
}

// recalculateContainerPositions updates container positions based on their children
func recalculateContainerPositions(g *Graph) {
	// Build a hierarchy to process containers bottom-up
	hierarchy := buildHierarchy(g)
	
	// Find all containers (nodes that have children)
	containers := make(map[string]bool)
	for parent, children := range hierarchy {
		if parent != "" && len(children) > 0 {
			containers[parent] = true
		}
	}
	
	// Process containers in bottom-up order (deepest first)
	processedContainers := make(map[string]bool)
	
	var processContainer func(containerID string)
	processContainer = func(containerID string) {
		if processedContainers[containerID] {
			return
		}
		
		container := g.GetNode(containerID)
		if container == nil {
			return
		}
		
		// First process any child containers
		for _, childID := range hierarchy[containerID] {
			if containers[childID] {
				processContainer(childID)
			}
		}
		
		// Find bounds of all descendants (not just direct children)
		minX, minY := math.Inf(1), math.Inf(1)
		maxX, maxY := math.Inf(-1), math.Inf(-1)
		hasChildren := false
		
		var collectBounds func(nodeID string)
		collectBounds = func(nodeID string) {
			// Process direct children
			for childID, parentID := range g.parent {
				if parentID == nodeID {
					child := g.GetNode(childID)
					if child != nil {
						hasChildren = true
						childLeft := child.X - child.Width/2
						childRight := child.X + child.Width/2
						childTop := child.Y - child.Height/2
						childBottom := child.Y + child.Height/2
						
						minX = math.Min(minX, childLeft)
						maxX = math.Max(maxX, childRight)
						minY = math.Min(minY, childTop)
						maxY = math.Max(maxY, childBottom)
						
						// Recursively include nested children
						if containers[childID] {
							collectBounds(childID)
						}
					}
				}
			}
		}
		
		collectBounds(containerID)
		
		if hasChildren {
			// Add padding
			padding := 30.0
			minX -= padding
			maxX += padding
			minY -= padding
			maxY += padding
			
			// Update container
			container.X = (minX + maxX) / 2
			container.Y = (minY + maxY) / 2
			container.Width = maxX - minX
			container.Height = maxY - minY
		}
		
		processedContainers[containerID] = true
	}
	
	// Process all root containers
	for containerID := range containers {
		processContainer(containerID)
	}
}