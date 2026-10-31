"use client";

// Capa de tiles vectoriales (MapLibre GL) dentro de un mapa de Leaflet. react-leaflet no
// trae un wrapper para esto, así que la capa se agrega y se quita con el mapa del contexto.
import "maplibre-gl/dist/maplibre-gl.css";
import { maplibreGL } from "@maplibre/maplibre-gl-leaflet";
import { useEffect } from "react";
import { useMap } from "react-leaflet";

import { MAP_ATTRIBUTION, MAP_STYLE_URL } from "@/lib/map";

export function VectorTileLayer() {
  const map = useMap();

  useEffect(() => {
    const layer = maplibreGL({ style: MAP_STYLE_URL });
    layer.addTo(map);
    map.attributionControl.addAttribution(MAP_ATTRIBUTION);
    return () => {
      map.attributionControl.removeAttribution(MAP_ATTRIBUTION);
      layer.remove();
    };
  }, [map]);

  return null;
}
