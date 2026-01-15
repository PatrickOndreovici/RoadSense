package graph

type Way struct {
	Id       int64
	Nodes    []int64
	OneWay   bool
	Reversed bool
}
