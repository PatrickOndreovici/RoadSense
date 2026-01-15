package rtree

import (
	"RoadSense/internal/graph"
	"fmt"

	"github.com/tidwall/rtree"
)

// EdgeRect stored in RTree with bounding box and edge metadata
type EdgeRect struct {
	MinX float64
	MinY float64
	MaxX float64
	MaxY float64
}

type RoadSegment struct {
	A *graph.Node
	B *graph.Node
}

func BuildRtree(nodes map[int64]*graph.Node) rtree.RTreeGN[float64, *RoadSegment] {
	var rt rtree.RTreeGN[float64, *RoadSegment]

	const epsilon = 1e-3 // tiny fix for zero-width/height

	cnt := 0
	for _, node := range nodes {
		for _, edge := range node.Edges {

			x1 := float64(node.Lat)
			y1 := float64(node.Lon)
			x2 := float64(nodes[edge.ToNodeID].Lat)
			y2 := float64(nodes[edge.ToNodeID].Lon)

			minX := min(x1, x2)
			minY := min(y1, y2)
			maxX := max(x1, x2)
			maxY := max(y1, y2)

			// Fix zero-size boxes
			if maxX == minX {
				maxX = minX + epsilon
			}
			if maxY == minY {
				maxY = minY + epsilon
			}

			er := &EdgeRect{
				MinX: minX,
				MinY: minY,
				MaxX: maxX,
				MaxY: maxY,
			}

			roadSegment := &RoadSegment{
				A: node,
				B: nodes[edge.ToNodeID],
			}

			rt.Insert(
				[2]float64{er.MinX, er.MinY},
				[2]float64{er.MaxX, er.MaxY},
				roadSegment,
			)

			cnt++
		}
	}
	fmt.Println(cnt)
	return rt
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func FindCandidateSegments(
	rt *rtree.RTreeGN[float64, *RoadSegment],
	gpsLat, gpsLon float64,
	maxDistMeters float64,
) []*RoadSegment {

	// Convert GPS point to your stored coordinate scale
	latScaled := gpsLat * graph.CoordMultiplier
	lonScaled := gpsLon * graph.CoordMultiplier

	// Convert meters → degrees
	dLat := graph.MetersToDegreesLat(maxDistMeters)
	dLon := graph.MetersToDegreesLon(maxDistMeters, gpsLat)

	// Scale to match R-tree
	dLat *= graph.CoordMultiplier
	dLon *= graph.CoordMultiplier

	min := [2]float64{
		latScaled - dLat,
		lonScaled - dLon,
	}
	max := [2]float64{
		latScaled + dLat,
		lonScaled + dLon,
	}

	var candidates []*RoadSegment

	rt.Search(min, max, func(_, _ [2]float64, seg *RoadSegment) bool {
		candidates = append(candidates, seg)
		return true
	})

	return candidates
}
