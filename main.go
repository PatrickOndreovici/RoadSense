package main

import (
	"RoadSense/api/handlers"
	graph "RoadSense/internal/graph"
	"RoadSense/internal/mapmatching"
	"RoadSense/internal/osm"
	rtree2 "RoadSense/internal/rtree"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

func readCsvFile(filePath string) [][]string {
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatal("Unable to read input file "+filePath, err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		log.Fatal("Unable to parse file as CSV for "+filePath, err)
	}

	return records
}

func parseGpsPoints(records [][]string) []mapmatching.GpsPoint {
	// Skip header row (index 0)
	gpsPoints := make([]mapmatching.GpsPoint, 0, len(records)-1)

	for i := 1; i < len(records); i++ {
		if len(records[i]) < 3 {
			log.Printf("Skipping invalid row %d: %v", i, records[i])
			continue
		}

		lat, err := strconv.ParseFloat(records[i][1], 64)
		if err != nil {
			log.Printf("Error parsing latitude at row %d: %v", i, err)
			continue
		}

		lng, err := strconv.ParseFloat(records[i][2], 64)
		if err != nil {
			log.Printf("Error parsing longitude at row %d: %v", i, err)
			continue
		}

		gpsPoints = append(gpsPoints, mapmatching.GpsPoint{
			Lat: lat,
			Lng: lng,
		})
	}

	return gpsPoints
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {

	// Parse OSM data and build graph
	fmt.Println("Parsing OSM file...")
	nodes, ways := osm.ParseOSMFile("./bucharest.osm.pbf")

	fmt.Println("Building graph...")
	g := graph.NewGraph(nodes)
	g.Build(ways)

	fmt.Println("Building R-tree...")
	r := rtree2.BuildRtree(g.Nodes)

	// Create map matcher with tuned parameters
	mapMatcher := mapmatching.NewMapMatcher(g, &r)

	mux := http.NewServeMux()

	handler := handlers.NewRouteHandler(mapMatcher)

	mux.HandleFunc("/api/route", handler.HandleCalculateRoute)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: corsMiddleware(mux),
	}
	fmt.Println("Server running on :8080")
	log.Fatal(srv.ListenAndServe())

	// You can optionally tune these parameters based on your GPS device accuracy
	// Uncomment and adjust if needed:
	// mapMatcher.SetStandardDeviation(5.0)  // GPS accuracy in meters
	// mapMatcher.SetBeta(15.0)              // Route deviation tolerance
	// mapMatcher.SetMaxRadius(150.0)        // Search radius in meters

}
