package graph

type UserGraph struct {
	Nodes map[int64]*Node
}

func NewUserGraph(nodes map[int64]*Node) *UserGraph {
	return &UserGraph{
		Nodes: nodes,
	}
}
