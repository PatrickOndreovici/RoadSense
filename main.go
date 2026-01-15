package main

import (
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

func main() {
	// Start profiling server
	go func() {
		fmt.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	// Read CSV file
	records := readCsvFile("./driving.csv")
	gpsPoints := parseGpsPoints(records)

	fmt.Printf("Loaded %d GPS points\n", len(gpsPoints))

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

	// You can optionally tune these parameters based on your GPS device accuracy
	// Uncomment and adjust if needed:
	// mapMatcher.SetStandardDeviation(5.0)  // GPS accuracy in meters
	// mapMatcher.SetBeta(15.0)              // Route deviation tolerance
	// mapMatcher.SetMaxRadius(150.0)        // Search radius in meters

	fmt.Println("Starting map matching...")
	results := mapMatcher.Match(gpsPoints)

	if results == nil {
		log.Fatal("Map matching failed - no results returned")
	}

	// Print results
	fmt.Printf("\n=== Map Matching Results ===\n")
	fmt.Printf("Matched %d points\n\n", len(results))

	for i, result := range results {
		fmt.Printf("Point %d:\n", i+1)
		fmt.Printf("  Original GPS: (%.6f, %.6f)\n",
			result.GpsPoint.Lat, result.GpsPoint.Lng)
		fmt.Printf("  Matched to:   (%.6f, %.6f)\n",
			result.MatchedRoadPoint.Lat, result.MatchedRoadPoint.Lng)
		fmt.Printf("  Road Segment: %s\n", result.RoadSegmentID)
		fmt.Printf("  Confidence:   %.6f\n", result.Confidence)

		// Calculate distance from original to matched point
		distance := graph.DistanceBetweenGpsPoints(
			result.GpsPoint.Lat, result.GpsPoint.Lng,
			result.MatchedRoadPoint.Lat, result.MatchedRoadPoint.Lng,
		)
		fmt.Printf("  Distance:     %.2f meters\n\n", distance)
	}

	// Calculate statistics
	totalDistance := 0.0
	for i := 0; i < len(results); i++ {
		distance := graph.DistanceBetweenGpsPoints(
			results[i].GpsPoint.Lat, results[i].GpsPoint.Lng,
			results[i].MatchedRoadPoint.Lat, results[i].MatchedRoadPoint.Lng,
		)
		totalDistance += distance
	}

	avgDistance := totalDistance / float64(len(results))
	fmt.Printf("=== Statistics ===\n")
	fmt.Printf("Average matching distance: %.2f meters\n", avgDistance)

	// Optional: Save results to CSV
	saveResultsToCSV("./matched_results.csv", results)
}

func saveResultsToCSV(filePath string, results []mapmatching.MatchResult) {
	f, err := os.Create(filePath)
	if err != nil {
		log.Printf("Error creating output file: %v", err)
		return
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// Write header
	header := []string{
		"original_lat", "original_lng",
		"matched_lat", "matched_lng",
		"road_segment_id", "confidence", "distance_meters",
	}
	writer.Write(header)

	// Write data
	for _, result := range results {
		distance := graph.DistanceBetweenGpsPoints(
			result.GpsPoint.Lat, result.GpsPoint.Lng,
			result.MatchedRoadPoint.Lat, result.MatchedRoadPoint.Lng,
		)

		row := []string{
			fmt.Sprintf("%.6f", result.GpsPoint.Lat),
			fmt.Sprintf("%.6f", result.GpsPoint.Lng),
			fmt.Sprintf("%.6f", result.MatchedRoadPoint.Lat),
			fmt.Sprintf("%.6f", result.MatchedRoadPoint.Lng),
			result.RoadSegmentID,
			fmt.Sprintf("%.6f", result.Confidence),
			fmt.Sprintf("%.2f", distance),
		}
		writer.Write(row)
	}

	fmt.Printf("Results saved to %s\n", filePath)
}
