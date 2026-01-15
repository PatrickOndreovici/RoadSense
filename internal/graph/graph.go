package graph

import (
	"container/heap"
	"math"
)

type Graph struct {
	Nodes map[int64]*Node
}

func NewGraph(nodes map[int64]*Node) *Graph {
	return &Graph{
		Nodes: nodes,
	}
}

func (graph *Graph) Build(ways []*Way) {
	for _, way := range ways {

		if way.Reversed {
			for i := len(way.Nodes)/2 - 1; i >= 0; i-- {
				opp := len(way.Nodes) - 1 - i
				way.Nodes[i], way.Nodes[opp] = way.Nodes[opp], way.Nodes[i]
			}
		}

		for i := 1; i < len(way.Nodes); i++ {
			fromID := way.Nodes[i-1]
			toID := way.Nodes[i]

			fromNode := graph.Nodes[fromID]
			toNode := graph.Nodes[toID]
			if fromNode == nil || toNode == nil {
				continue
			}

			fromNode.InsertEdge(toNode, way.Id)

			if !way.OneWay {
				toNode.InsertEdge(fromNode, way.Id)
			}
		}
	}
}

// ---------- Priority Queue ----------

type Item struct {
	NodeID int64
	Cost   float64
	Index  int
}

type PriorityQueue []*Item

func (pq PriorityQueue) Len() int           { return len(pq) }
func (pq PriorityQueue) Less(i, j int) bool { return pq[i].Cost < pq[j].Cost }
func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	item := x.(*Item)
	item.Index = len(*pq)
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.Index = -1
	*pq = old[:n-1]
	return item
}

// ---------- Dijkstra ----------

func (g *Graph) Dijkstra(startID, endID int64, userGraph *UserGraph) (float64, []int64) {
	// If start or end missing
	if g.Nodes[startID] == nil || g.Nodes[endID] == nil {
		return math.Inf(1), nil
	}

	dist := make(map[int64]float64, len(g.Nodes))
	prev := make(map[int64]int64, len(g.Nodes))

	for id := range g.Nodes {
		dist[id] = math.Inf(1)
	}

	dist[startID] = 0

	pq := &PriorityQueue{}
	heap.Push(pq, &Item{NodeID: startID, Cost: 0})

	visited := make(map[int64]bool)

	for pq.Len() > 0 {
		item := heap.Pop(pq).(*Item)
		currentID := item.NodeID

		if visited[currentID] {
			continue
		}
		visited[currentID] = true

		// Early exit
		if currentID == endID {
			break
		}

		currentNode := g.Nodes[currentID]
		// for negative ids, take the node from user graph
		if currentID < 0 {
			currentNode = userGraph.Nodes[currentID]
		}

		for _, edge := range currentNode.Edges {
			nextID := edge.ToNodeID
			if visited[nextID] {
				continue
			}

			newCost := dist[currentID] + edge.Distance
			if newCost < dist[nextID] {
				dist[nextID] = newCost
				prev[nextID] = currentID
				heap.Push(pq, &Item{NodeID: nextID, Cost: newCost})
			}
		}
	}

	// No path found
	if dist[endID] == math.Inf(1) {
		return math.Inf(1), nil
	}

	// Reconstruct path
	path := []int64{}
	for cur := endID; cur != startID; cur = prev[cur] {
		path = append(path, cur)
	}
	path = append(path, startID)

	// Reverse
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	return dist[endID], path
}

func (g *Graph) DijkstraWithinDistance(
	startID, endID int64,
	maxDistance float64,
	userGraph *UserGraph,
) float64 {
	dist := make(map[int64]float64, len(g.Nodes))
	visited := make(map[int64]bool, len(g.Nodes))

	for id := range g.Nodes {
		dist[id] = math.Inf(1)
	}
	dist[startID] = 0

	pq := &PriorityQueue{}
	heap.Push(pq, &Item{NodeID: startID, Cost: 0})

	for pq.Len() > 0 {
		item := heap.Pop(pq).(*Item)
		currentID := item.NodeID
		currentCost := item.Cost

		// If the smallest cost already exceeds maxDistance, stop
		if currentCost > maxDistance {
			break
		}

		if visited[currentID] {
			continue
		}
		visited[currentID] = true

		// Early exit if we reached destination
		if currentID == endID {
			return currentCost
		}

		currentNode := g.Nodes[currentID]
		if currentID < 0 {
			currentNode = userGraph.Nodes[currentID]
		}

		for _, edge := range currentNode.Edges {
			nextID := edge.ToNodeID
			if visited[nextID] {
				continue
			}

			newCost := currentCost + edge.Distance

			// Ignore paths beyond maxDistance
			if newCost > maxDistance {
				continue
			}

			if newCost < dist[nextID] {
				dist[nextID] = newCost
				heap.Push(pq, &Item{
					NodeID: nextID,
					Cost:   newCost,
				})
			}
		}
	}

	// No valid path within maxDistance
	return math.Inf(1)
}

// DijkstraMultiTarget computes shortest distances from startID to all nodes in targetNodes
// Returns: map[targetNodeID]distance
// This is MORE EFFICIENT than calling Dijkstra multiple times when you have multiple targets
func (g *Graph) DijkstraMultiTarget(
	startID int64,
	targetNodes map[int64]bool,
	maxDistance float64,
	userGraph *UserGraph,
) map[int64]float64 {

	dist := make(map[int64]float64)
	visited := make(map[int64]bool)

	// Initialize distances for all target nodes
	for id := range targetNodes {
		dist[id] = math.Inf(1)
	}
	dist[startID] = 0

	pq := &PriorityQueue{}
	heap.Push(pq, &Item{NodeID: startID, Cost: 0})

	foundCount := 0
	targetCount := len(targetNodes)

	for pq.Len() > 0 {
		item := heap.Pop(pq).(*Item)
		currentID := item.NodeID
		currentCost := item.Cost

		// If cost exceeds maxDistance, stop
		if currentCost > maxDistance {
			break
		}

		if visited[currentID] {
			continue
		}
		visited[currentID] = true

		// Check if we found a target
		if targetNodes[currentID] {
			dist[currentID] = currentCost
			foundCount++

			// Early exit if all targets found
			if foundCount == targetCount {
				break
			}
		}

		currentNode := g.Nodes[currentID]
		if currentID < 0 {
			currentNode = userGraph.Nodes[currentID]
		}

		for _, edge := range currentNode.Edges {
			nextID := edge.ToNodeID
			if visited[nextID] {
				continue
			}

			newCost := currentCost + edge.Distance
			if newCost > maxDistance {
				continue
			}

			// CRITICAL FIX: Check if node has been seen before
			// If not in map, distance is implicitly infinite
			oldCost, exists := dist[nextID]
			if !exists {
				oldCost = math.Inf(1)
			}

			if newCost < oldCost {
				dist[nextID] = newCost
				heap.Push(pq, &Item{
					NodeID: nextID,
					Cost:   newCost,
				})
			}
		}
	}

	return dist
}
