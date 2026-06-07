package handlers

import (
	"RoadSense/internal/mapmatching"
	"encoding/json"
	"net/http"
)

type RouteHandler struct {
	mapMatcher *mapmatching.MapMatcher
}

func NewRouteHandler(mapMatcher *mapmatching.MapMatcher) *RouteHandler {
	return &RouteHandler{
		mapMatcher: mapMatcher,
	}
}

type routeRequest struct {
	Coordinates []struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"coordinates"`
}

func routeRequestToGpsPoint(request routeRequest) []mapmatching.GpsPoint {
	var gpsPoints []mapmatching.GpsPoint
	for i := 0; i < len(request.Coordinates); i++ {
		gpsPoints = append(gpsPoints, mapmatching.GpsPoint{
			Lat: request.Coordinates[i].Lat,
			Lng: request.Coordinates[i].Lng,
		})
	}
	return gpsPoints
}

func (h *RouteHandler) HandleCalculateRoute(w http.ResponseWriter, r *http.Request) {
	var req routeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	gpsPoints := routeRequestToGpsPoint(req)

	w.Header().Set("Content-Type", "application/json")
	results := h.mapMatcher.Match(gpsPoints)
	json.NewEncoder(w).Encode(results)
	return
}
