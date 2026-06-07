import { useState } from "react";
import {
  MapContainer,
  Marker,
  TileLayer,
  Popup,
  useMapEvents
} from "react-leaflet";

function MapEvents({ addMarker }) {
  useMapEvents({
    click(e) {
      addMarker(e.latlng);
    }
  });

  return null;
}

export default function Map() {
  const [markers, setMarkers] = useState([]);

  const addMarker = (latlng) => {
    setMarkers((prev) => [
      ...prev,
      {
        id: crypto.randomUUID(),
        position: latlng
      }
    ]);
  };

  const deleteMarker = (id) => {
    setMarkers((prev) => prev.filter((m) => m.id !== id));
  };

  return (
    <MapContainer
      center={[51.505, -0.09]}
      zoom={13}
      scrollWheelZoom={false}
      style={{ height: "100vh", width: "100%" }}
    >
      <MapEvents addMarker={addMarker} />

      <TileLayer
        attribution='&copy; OpenStreetMap contributors'
        url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
      />

      {markers.map((marker) => (
        <Marker
          key={marker.id}
          position={marker.position}
          eventHandlers={{
            click: () => deleteMarker(marker.id)
          }}
        >
          <Popup>
            Click marker to delete
          </Popup>
        </Marker>
      ))}
    </MapContainer>
  );
}