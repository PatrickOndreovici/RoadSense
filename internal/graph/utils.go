package graph

import (
	"math"
)

func Distance(a, b *Node) float64 {
	lat1 := float64(a.Lat) / CoordMultiplier
	lon1 := float64(a.Lon) / CoordMultiplier
	lat2 := float64(b.Lat) / CoordMultiplier
	lon2 := float64(b.Lon) / CoordMultiplier

	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	dLat := lat2Rad - lat1Rad
	dLon := lon2Rad - lon1Rad

	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	return 2 * earthRadiusMeters * math.Asin(math.Sqrt(h))
}

func DistanceBetweenGpsPoints(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	dLat := lat2Rad - lat1Rad
	dLon := lon2Rad - lon1Rad

	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	return 2 * earthRadiusMeters * math.Asin(math.Sqrt(h))
}

func latLonToXY(lat, lon float64, refLat float64) (x, y float64) {
	x = lon * degToRad * earthRadiusMeters * math.Cos(refLat*degToRad)
	y = lat * degToRad * earthRadiusMeters
	return
}

type Projection struct {
	Lat      float64 // projected latitude
	Lon      float64 // projected longitude
	Distance float64 // distance from GPS point to edge (meters)
	T        float64 // position along edge [0..1]
}

func ProjectPointToEdge(
	gpsLat, gpsLon float64,
	a, b *Node,
) Projection {

	// Convert stored coords
	aLat := float64(a.Lat) / CoordMultiplier
	aLon := float64(a.Lon) / CoordMultiplier
	bLat := float64(b.Lat) / CoordMultiplier
	bLon := float64(b.Lon) / CoordMultiplier

	// Local Cartesian coordinates
	ax, ay := latLonToXY(aLat, aLon, aLat)
	bx, by := latLonToXY(bLat, bLon, aLat)
	px, py := latLonToXY(gpsLat, gpsLon, aLat)

	// Vector AB and AP
	abx := bx - ax
	aby := by - ay
	apx := px - ax
	apy := py - ay

	// Project AP onto AB
	abLen2 := abx*abx + aby*aby
	var t float64
	if abLen2 > 0 {
		t = (apx*abx + apy*aby) / abLen2
	}

	// Clamp to segment
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}

	// Closest point
	cx := ax + t*abx
	cy := ay + t*aby

	// Distance
	dx := px - cx
	dy := py - cy
	dist := math.Hypot(dx, dy)

	// Convert back to lat/lon
	projLat := (cy / earthRadiusMeters) * radToDeg
	projLon := (cx / (earthRadiusMeters * math.Cos(aLat*degToRad))) * radToDeg

	return Projection{
		Lat:      projLat,
		Lon:      projLon,
		Distance: dist,
		T:        t,
	}
}

func MetersToDegreesLat(m float64) float64 {
	return (m / earthRadiusMeters) * radToDeg
}

func MetersToDegreesLon(m float64, lat float64) float64 {
	return (m / (earthRadiusMeters * math.Cos(lat*degToRad))) * radToDeg
}
