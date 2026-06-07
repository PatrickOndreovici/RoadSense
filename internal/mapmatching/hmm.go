package mapmatching

import (
	"RoadSense/internal/graph"
	rtree2 "RoadSense/internal/rtree"
	"fmt"
	"math"
	"sort"

	"github.com/tidwall/rtree"
)

type MapMatcher struct {
	graph             *graph.Graph
	rtree             *rtree.RTreeGN[float64, *rtree2.RoadSegment]
	maxRadius         float64
	standardDeviation float64
	beta              float64
}

type MatchResult struct {
	GpsPoint         GpsPoint
	MatchedRoadPoint CandidatePoint
	RoadSegmentID    string
	Confidence       float64
	RouteToNext      []CandidatePoint // NEW: Route geometry to next point
}

type GpsPoint struct {
	Lat float64
	Lng float64
}

type CandidatePoint struct {
	Lat float64
	Lng float64
}

type Candidate struct {
	NodeA      *graph.Node
	NodeB      *graph.Node
	Proj       graph.Projection
	EdgeLength float64
}

func NewMapMatcher(g *graph.Graph, rt *rtree.RTreeGN[float64, *rtree2.RoadSegment]) *MapMatcher {
	return &MapMatcher{
		graph:             g,
		rtree:             rt,
		maxRadius:         50.0,
		standardDeviation: 12,
		beta:              40,
	}
}

func (mm *MapMatcher) FindCandidates(gps GpsPoint) []Candidate {
	latScaled := gps.Lat * graph.CoordMultiplier
	lngScaled := gps.Lng * graph.CoordMultiplier

	dLat := graph.MetersToDegreesLat(mm.maxRadius)
	dLng := graph.MetersToDegreesLon(mm.maxRadius, gps.Lat)

	dLat *= graph.CoordMultiplier
	dLng *= graph.CoordMultiplier

	minBounds := [2]float64{latScaled - dLat, lngScaled - dLng}
	maxBounds := [2]float64{latScaled + dLat, lngScaled + dLng}

	candidates := make([]Candidate, 0)

	mm.rtree.Search(minBounds, maxBounds, func(min, max [2]float64, segment *rtree2.RoadSegment) bool {
		nodeA := segment.A
		nodeB := segment.B

		proj := graph.ProjectPointToEdge(gps.Lat, gps.Lng, nodeA, nodeB)

		if proj.Distance <= mm.maxRadius {
			edgeLength := graph.DistanceBetweenGpsPoints(
				float64(nodeA.Lat)/graph.CoordMultiplier,
				float64(nodeA.Lon)/graph.CoordMultiplier,
				float64(nodeB.Lat)/graph.CoordMultiplier,
				float64(nodeB.Lon)/graph.CoordMultiplier,
			)

			candidates = append(candidates, Candidate{
				NodeA:      nodeA,
				NodeB:      nodeB,
				Proj:       proj,
				EdgeLength: edgeLength,
			})
		}

		return true
	})

	if len(candidates) > 15 {
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].Proj.Distance < candidates[j].Proj.Distance
		})

		maxCandidates := 15
		threshold := 3.0 * mm.standardDeviation

		pruned := make([]Candidate, 0, maxCandidates)
		for i := 0; i < len(candidates) && i < maxCandidates; i++ {
			if candidates[i].Proj.Distance <= threshold {
				pruned = append(pruned, candidates[i])
			}
		}

		if len(pruned) < 8 && len(candidates) >= 8 {
			candidates = candidates[:8]
		} else {
			candidates = pruned
		}
	}

	return candidates
}

func (mm *MapMatcher) Match(gpsPoints []GpsPoint) []MatchResult {
	if len(gpsPoints) == 0 {
		return nil
	}

	filteredPoints := gpsPoints

	if len(filteredPoints) == 0 {
		return nil
	}

	allCandidates := make([][]Candidate, len(filteredPoints))
	for i, gps := range filteredPoints {
		allCandidates[i] = mm.FindCandidates(gps)
		if len(allCandidates[i]) == 0 {
			fmt.Printf("No candidates found for GPS point %d (%.5f, %.5f)\n", i, gps.Lat, gps.Lng)
			return nil
		}
	}

	logViterbi := make([][]float64, len(filteredPoints))
	backpointer := make([][]int, len(filteredPoints))

	logViterbi[0] = make([]float64, len(allCandidates[0]))
	for i, candidate := range allCandidates[0] {
		emission := mm.emissionProbability(filteredPoints[0], candidate)
		if emission > 0 {
			logViterbi[0][i] = math.Log(emission)
		} else {
			logViterbi[0][i] = math.Inf(-1)
		}
	}

	for t := 1; t < len(filteredPoints); t++ {
		logViterbi[t] = make([]float64, len(allCandidates[t]))
		backpointer[t] = make([]int, len(allCandidates[t]))

		greatCircleDistance := graph.DistanceBetweenGpsPoints(
			filteredPoints[t-1].Lat, filteredPoints[t-1].Lng,
			filteredPoints[t].Lat, filteredPoints[t].Lng,
		)

		routeDistances := mm.computeAllRouteDistances(
			filteredPoints[t-1], filteredPoints[t],
			allCandidates[t-1], allCandidates[t],
			greatCircleDistance,
		)

		for i, candidate := range allCandidates[t] {
			maxLogProb := math.Inf(-1)
			maxPrevIdx := 0

			emission := mm.emissionProbability(filteredPoints[t], candidate)
			logEmission := math.Inf(-1)
			if emission > 0 {
				logEmission = math.Log(emission)
			}

			for j := range allCandidates[t-1] {
				routeDistance := routeDistances[j][i]

				var logTransition float64
				if math.IsInf(routeDistance, 1) {
					logTransition = -20.0
				} else {
					dt := math.Abs(greatCircleDistance - routeDistance)
					transition := (1.0 / mm.beta) * math.Exp(-dt/mm.beta)
					logTransition = math.Log(transition)
				}

				logProb := logViterbi[t-1][j] + logTransition + logEmission

				if logProb > maxLogProb {
					maxLogProb = logProb
					maxPrevIdx = j
				}
			}

			logViterbi[t][i] = maxLogProb
			backpointer[t][i] = maxPrevIdx
		}
	}

	bestPath := make([]int, len(filteredPoints))

	maxLogProb := math.Inf(-1)
	maxIdx := 0
	for i, logProb := range logViterbi[len(filteredPoints)-1] {
		if logProb > maxLogProb {
			maxLogProb = logProb
			maxIdx = i
		}
	}
	bestPath[len(filteredPoints)-1] = maxIdx

	for t := len(filteredPoints) - 2; t >= 0; t-- {
		bestPath[t] = backpointer[t+1][bestPath[t+1]]
	}

	// Build flattened results with ALL route points
	results := make([]MatchResult, 0)

	for t, candidateIdx := range bestPath {
		candidate := allCandidates[t][candidateIdx]

		var segmentID string
		if candidate.NodeA.ID < candidate.NodeB.ID {
			segmentID = formatSegmentID(candidate.NodeA.ID, candidate.NodeB.ID)
		} else {
			segmentID = formatSegmentID(candidate.NodeB.ID, candidate.NodeA.ID)
		}

		totalProb := 0.0
		for i := range allCandidates[t] {
			if !math.IsInf(logViterbi[t][i], -1) {
				totalProb += math.Exp(logViterbi[t][i])
			}
		}

		confidence := 0.0
		if totalProb > 0 && !math.IsInf(logViterbi[t][candidateIdx], -1) {
			confidence = math.Exp(logViterbi[t][candidateIdx]) / totalProb
		}

		// Add matched point
		results = append(results, MatchResult{
			GpsPoint: filteredPoints[t],
			MatchedRoadPoint: CandidatePoint{
				Lat: candidate.Proj.Lat,
				Lng: candidate.Proj.Lon,
			},
			RoadSegmentID: segmentID,
			Confidence:    confidence,
			RouteToNext:   nil,
		})

		// Add all intermediate route points to next matched point
		if t < len(filteredPoints)-1 {
			nextCandidate := allCandidates[t+1][bestPath[t+1]]
			routePoints := mm.reconstructRoute(candidate, nextCandidate)

			// Skip first point (already added above) and last point (will be added in next iteration)
			for i := 1; i < len(routePoints)-1; i++ {
				results = append(results, MatchResult{
					GpsPoint:         GpsPoint{Lat: 0, Lng: 0}, // No original GPS point for route nodes
					MatchedRoadPoint: routePoints[i],
					RoadSegmentID:    "",         // Intermediate point
					Confidence:       confidence, // Inherit confidence from matched point
					RouteToNext:      nil,
				})
			}
		}
	}

	return results
}

// NEW: Reconstruct the actual route between two matched candidates
func (mm *MapMatcher) reconstructRoute(from, to Candidate) []CandidatePoint {
	route := []CandidatePoint{}

	// Same edge case
	if mm.onSameEdge(from, to) {
		// Just return the two projection points
		route = append(route, CandidatePoint{Lat: from.Proj.Lat, Lng: from.Proj.Lon})
		route = append(route, CandidatePoint{Lat: to.Proj.Lat, Lng: to.Proj.Lon})
		return route
	}

	// Different edges - need to find shortest path
	userGraph := mm.CreateUserGraph(from)
	startID := int64(-1)

	// Determine which node of 'to' edge to route to
	distA := to.EdgeLength * to.Proj.T
	distB := to.EdgeLength * (1.0 - to.Proj.T)

	// Try both routes and pick the shorter one
	distToA, pathToA := mm.graph.Dijkstra(startID, to.NodeA.ID, userGraph)
	distToB, pathToB := mm.graph.Dijkstra(startID, to.NodeB.ID, userGraph)

	var finalPath []int64
	var _ *graph.Node

	if distToA+distA < distToB+distB {
		finalPath = pathToA
		_ = to.NodeA
	} else {
		finalPath = pathToB
		_ = to.NodeB
	}

	// Convert path to coordinates
	if len(finalPath) > 0 {
		for _, nodeID := range finalPath {
			var node *graph.Node
			if nodeID < 0 {
				node = userGraph.Nodes[nodeID]
			} else {
				node = mm.graph.Nodes[nodeID]
			}

			if node != nil {
				route = append(route, CandidatePoint{
					Lat: float64(node.Lat) / graph.CoordMultiplier,
					Lng: float64(node.Lon) / graph.CoordMultiplier,
				})
			}
		}
	}

	// Add final projection point
	route = append(route, CandidatePoint{Lat: to.Proj.Lat, Lng: to.Proj.Lon})

	return route
}

func (mm *MapMatcher) computeAllRouteDistances(
	gpsPoint1, gpsPoint2 GpsPoint,
	prevCandidates, currCandidates []Candidate,
	greatCircleDistance float64,
) [][]float64 {

	routeDistances := make([][]float64, len(prevCandidates))
	for i := range routeDistances {
		routeDistances[i] = make([]float64, len(currCandidates))
	}

	maxRouteDistance := greatCircleDistance + 5.0*mm.beta

	for prevIdx, prevCandidate := range prevCandidates {

		targetNodes := make(map[int64]bool)
		for _, currCandidate := range currCandidates {
			if !mm.onSameEdge(prevCandidate, currCandidate) {
				targetNodes[currCandidate.NodeA.ID] = true
				targetNodes[currCandidate.NodeB.ID] = true
			}
		}

		var distances map[int64]float64
		if len(targetNodes) > 0 {
			userGraph := mm.CreateUserGraph(prevCandidate)
			startID := int64(-1)

			distances = mm.graph.DijkstraMultiTarget(
				startID,
				targetNodes,
				maxRouteDistance,
				userGraph,
			)
		}

		for currIdx, currCandidate := range currCandidates {

			if mm.onSameEdge(prevCandidate, currCandidate) {
				routeDistances[prevIdx][currIdx] = mm.computeSameEdgeDistance(
					prevCandidate, currCandidate,
				)
			} else {
				distA2ToProj2 := currCandidate.EdgeLength * currCandidate.Proj.T
				distB2ToProj2 := currCandidate.EdgeLength * (1.0 - currCandidate.Proj.T)

				distToA, okA := distances[currCandidate.NodeA.ID]
				routeViaA := math.Inf(1)
				if okA && !math.IsInf(distToA, 1) {
					routeViaA = distToA + distA2ToProj2
				}

				distToB, okB := distances[currCandidate.NodeB.ID]
				routeViaB := math.Inf(1)
				if okB && !math.IsInf(distToB, 1) {
					routeViaB = distToB + distB2ToProj2
				}

				routeDistances[prevIdx][currIdx] = math.Min(routeViaA, routeViaB)
			}
		}
	}

	return routeDistances
}

func (mm *MapMatcher) onSameEdge(c1, c2 Candidate) bool {
	return (c1.NodeA.ID == c2.NodeA.ID && c1.NodeB.ID == c2.NodeB.ID) ||
		(c1.NodeA.ID == c2.NodeB.ID && c1.NodeB.ID == c2.NodeA.ID)
}

func (mm *MapMatcher) computeSameEdgeDistance(c1, c2 Candidate) float64 {
	if c1.NodeA.ID == c2.NodeA.ID {
		return math.Abs(c2.Proj.T-c1.Proj.T) * c1.EdgeLength
	} else {
		t1 := c1.Proj.T
		t2 := 1.0 - c2.Proj.T
		return math.Abs(t2-t1) * c1.EdgeLength
	}
}

func formatSegmentID(id1, id2 int64) string {
	return fmt.Sprintf("%d-%d", id1, id2)
}

func (mm *MapMatcher) emissionProbability(gpsPoint GpsPoint, candidate Candidate) float64 {
	distance := candidate.Proj.Distance

	sigma := mm.standardDeviation
	coefficient := 1.0 / (math.Sqrt(2*math.Pi) * sigma)
	exponent := -0.5 * distance * distance / (sigma * sigma)

	return coefficient * math.Exp(exponent)
}

func (mm *MapMatcher) CreateUserGraph(candidate Candidate) *graph.UserGraph {
	userNodes := make(map[int64]*graph.Node)

	projNode := &graph.Node{
		ID:  -1,
		Lat: int32(candidate.Proj.Lat * graph.CoordMultiplier),
		Lon: int32(candidate.Proj.Lon * graph.CoordMultiplier),
	}

	distToB := candidate.EdgeLength * (1.0 - candidate.Proj.T)
	distToA := candidate.EdgeLength * candidate.Proj.T
	projNode.InsertEdgeWithDistance(candidate.NodeB, distToB)
	projNode.InsertEdgeWithDistance(candidate.NodeA, distToA)

	userNodes[-1] = projNode
	return graph.NewUserGraph(userNodes)
}
