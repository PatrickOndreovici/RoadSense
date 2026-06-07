import { useState, useEffect, useRef } from "react";
import L from "leaflet";
import {
  MapContainer,
  Marker,
  TileLayer,
  Popup,
  Polyline,
  useMapEvents,
  useMap,
} from "react-leaflet";

function MapEvents({ addMarker }) {
  useMapEvents({
    click(e) {
      addMarker(e.latlng);
    },
  });
  return null;
}

function MarkerList({ markers }) {
  const map = useMap();
  const containerRef = useRef(null);


  useEffect(() => {
    if (containerRef.current) {
      L.DomEvent.disableClickPropagation(containerRef.current);
    }
    return () => clearTimeout(containerRef.current);
  }, []);

  const focusMarker = (position) => {
    map.flyTo(position, 16, { duration: 1 });
  };

  return (
    <div
        ref={containerRef}
      style={{
        position: "absolute",
        top: 10,
        right: 10,
        zIndex: 1000,
        width: 280,
        maxHeight: 300,
        overflowY: "auto",
        background: "white",
        borderRadius: 8,
        boxShadow: "0 2px 10px rgba(0,0,0,0.15)",
        padding: 12,
      }}
    >
      <h3 style={{ margin: "0 0 10px", fontSize: 16, color: "#333" }}>
        Markers ({markers.length})
      </h3>

      {markers.length === 0 && (
        <div style={{ color: "#666" }}>Click on the map to add markers</div>
      )}

      {markers.map((marker, index) => (
        <div
          key={marker.id}
          onClick={() => focusMarker(marker.position)}
          style={{
            padding: 10,
            marginBottom: 8,
            border: "1px solid #eee",
            borderRadius: 6,
            cursor: "pointer",
            transition: "0.2s",
          }}
          onMouseEnter={(e) => {
            e.currentTarget.style.background = "#f5f5f5";
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.background = "white";
          }}
        >
          <div style={{ fontWeight: 600, marginBottom: 4, color: "#333" }}>
            Marker #{index + 1}
          </div>
          <div style={{ fontSize: 13, color: "#555" }}>
            📍 {marker.position.lat.toFixed(5)}, {marker.position.lng.toFixed(5)}
          </div>
        </div>
      ))}
    </div>
  );
}

function CalculateRouteButton({ markers, onRouteCalculated, onClearAll }) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const errorTimerRef = useRef(null);
  const containerRef = useRef(null);

  useEffect(() => {
    if (containerRef.current) {
      L.DomEvent.disableClickPropagation(containerRef.current);
    }
    return () => clearTimeout(errorTimerRef.current);
  }, []);

  const showError = (msg) => {
    clearTimeout(errorTimerRef.current);
    setError(msg);
    errorTimerRef.current = setTimeout(() => setError(null), 3000);
  };

  const calculateRoute = async () => {
    if (markers.length < 2) {
      showError("Add at least 2 markers to calculate a route.");
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const coordinates = markers.map((m) => ({
        lat: m.position.lat,
        lng: m.position.lng,
      }));

      const response = await fetch("http://localhost:8080/api/route", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ coordinates }),
      });

      if (!response.ok) throw new Error(`Server error: ${response.status}`);

      const data = await response.json();
      console.log("Route response:", data);

      const routeCoords = data
        .filter((point) => point.MatchedRoadPoint !== null)
        .map((point) => [point.MatchedRoadPoint.Lat, point.MatchedRoadPoint.Lng]);

      onRouteCalculated(routeCoords);
    } catch (err) {
      showError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div
      ref={containerRef}
      style={{
        position: "absolute",
        bottom: 24,
        right: 16,
        zIndex: 1000,
        display: "flex",
        flexDirection: "column",
        alignItems: "flex-end",
        gap: 8,
      }}
    >
      {error && (
        <div
          style={{
            background: "#fff0f0",
            color: "#c0392b",
            border: "1px solid #f5c6c6",
            borderRadius: 6,
            padding: "8px 12px",
            fontSize: 13,
            maxWidth: 240,
            textAlign: "right",
          }}
        >
          {error}
        </div>
      )}

      <div style={{ display: "flex", gap: 8 }}>
        <button
          onClick={onClearAll}
          style={{
            padding: "12px 20px",
            background: "#7f8c8d",
            color: "white",
            border: "none",
            borderRadius: 8,
            fontSize: 15,
            fontWeight: 600,
            cursor: "pointer",
            boxShadow: "0 2px 10px rgba(0,0,0,0.2)",
            whiteSpace: "nowrap",
          }}
          onMouseEnter={(e) => {
            e.currentTarget.style.background = "#636e72";
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.background = "#7f8c8d";
          }}
        >
          Clear All
        </button>

        <button
          onClick={calculateRoute}
          disabled={loading}
          style={{
            padding: "12px 20px",
            background: loading ? "#7f8c8d" : "#2c3e50",
            color: "white",
            border: "none",
            borderRadius: 8,
            fontSize: 15,
            fontWeight: 600,
            cursor: loading ? "not-allowed" : "pointer",
            boxShadow: "0 2px 10px rgba(0,0,0,0.2)",
            transition: "background 0.2s",
            whiteSpace: "nowrap",
          }}
          onMouseEnter={(e) => {
            if (!loading) e.currentTarget.style.background = "#1a252f";
          }}
          onMouseLeave={(e) => {
            if (!loading) e.currentTarget.style.background = "#2c3e50";
          }}
        >
          {loading ? "Calculating…" : "Calculate Route"}
        </button>
      </div>
    </div>
  );
}

export default function Map() {
  const [markers, setMarkers] = useState([]);
  const [routeCoords, setRouteCoords] = useState([]);

  const addMarker = (latlng) => {
    setMarkers((prev) => [
      ...prev,
      { id: crypto.randomUUID(), position: latlng },
    ]);
  };

  const deleteMarker = (id) => {
    setMarkers((prev) => prev.filter((m) => m.id !== id));
  };

  const clearAll = () => {
    setMarkers([]);
    setRouteCoords([]);
  };

  return (
    <MapContainer
      center={[44.4268, 26.1025]}
      zoom={13}
      scrollWheelZoom={true}
      style={{ height: "100vh", width: "100%" }}
    >
      <MapEvents addMarker={addMarker} />

      <TileLayer
        attribution="&copy; OpenStreetMap contributors"
        url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
      />

      <MarkerList markers={markers} />
      <CalculateRouteButton
        markers={markers}
        onRouteCalculated={setRouteCoords}
        onClearAll={clearAll}
      />

      {routeCoords.length > 0 && (
        <Polyline positions={routeCoords} color="#e74c3c" weight={4} opacity={0.8} />
      )}

      {markers.map((marker) => (
        <Marker
          key={marker.id}
          position={marker.position}
          eventHandlers={{ click: () => deleteMarker(marker.id) }}
        >
          <Popup>Click marker to delete</Popup>
        </Marker>
      ))}
    </MapContainer>
  );
}