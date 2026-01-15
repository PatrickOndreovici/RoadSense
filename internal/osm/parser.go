package osm

import (
	"RoadSense/internal/graph"
	"io"
	"log"
	"os"
	"runtime"

	"github.com/qedus/osmpbf"
)

// Allowed car-drivable road types
var drivable = map[string]bool{
	"motorway":       true,
	"motorway_link":  true,
	"trunk":          true,
	"trunk_link":     true,
	"primary":        true,
	"primary_link":   true,
	"secondary":      true,
	"secondary_link": true,
	"tertiary":       true,
	"tertiary_link":  true,
	"unclassified":   true,
	"residential":    true,
	"living_street":  true,
	"service":        true,
}

func ParseOSMFile(path string) (map[int64]*graph.Node, []*graph.Way) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	d := osmpbf.NewDecoder(f)
	d.SetBufferSize(osmpbf.MaxBlobSize)

	if err := d.Start(runtime.GOMAXPROCS(-1)); err != nil {
		log.Fatal(err)
	}

	nodeMap := make(map[int64]*graph.Node, 5_000_000)
	ways := make([]*graph.Way, 0, 1_000_000)

	for {
		v, err := d.Decode()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		switch obj := v.(type) {

		case *osmpbf.Node:
			// store only if referenced later (but we don't know yet)
			nodeMap[obj.ID] = &graph.Node{
				ID:  obj.ID,
				Lat: int32(obj.Lat * graph.CoordMultiplier),
				Lon: int32(obj.Lon * graph.CoordMultiplier),
			}

		case *osmpbf.Way:
			highway := obj.Tags["highway"]
			if !drivable[highway] {
				continue // skip non-drivable ways
			}

			nodes := make([]int64, len(obj.NodeIDs))
			copy(nodes, obj.NodeIDs)

			oneway, reversed := normalizeOneway(obj.Tags["oneway"])
			w := &graph.Way{
				Id:       obj.ID,
				Nodes:    nodes,
				OneWay:   oneway,
				Reversed: reversed,
			}
			ways = append(ways, w)

		case *osmpbf.Relation:
			// ignore; not needed for car graph

		default:
			// ignore unknown types
		}
	}

	return nodeMap, ways
}

func normalizeOneway(tag string) (oneway bool, reversed bool) {
	switch tag {
	case "yes", "1", "true", "T", "Yes", "YES", "True", "TRUE":
		return true, false
	case "-1":
		return true, true
	default:
		return false, false
	}
}
