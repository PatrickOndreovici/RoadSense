package graph

type Node struct {
	ID    int64
	Lat   int32
	Lon   int32
	Edges []*Edge
}

func (node *Node) InsertEdge(toNode *Node, wayId int64) {
	dist := Distance(node, toNode)

	node.Edges = append(node.Edges, &Edge{
		ToNodeID: toNode.ID,
		Distance: dist,
		WayId:    wayId,
	})
}

func (node *Node) InsertEdgeWithDistance(toNode *Node, distance float64) {
	node.Edges = append(node.Edges, &Edge{
		ToNodeID: toNode.ID,
		Distance: distance,
		WayId:    0,
	})
}
