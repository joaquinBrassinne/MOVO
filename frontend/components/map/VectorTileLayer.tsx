"use client";

// Capa de tiles vectoriales (MapLibre GL) dentro de un mapa de Leaflet. react-leaflet no
// trae un wrapper para esto, así que la capa se agrega y se quita con el mapa del contexto.
import "maplibre-gl/dist/maplibre-gl.css";
import { maplibreGL } from "@maplibre/maplibre-gl-leaflet";
import { addProtocol, setWorkerUrl } from "maplibre-gl";
import { Protocol } from "pmtiles";
import { useEffect } from "react";
import { useMap } from "react-leaflet";

import { MAP_ATTRIBUTION, mapStyle } from "@/lib/map";

// MapLibre deduce la URL de su worker desde su propio archivo, y el bundler lo mueve:
// "Worker failed to load". Se emite el worker como asset y se le dice dónde está.
setWorkerUrl(new URL("maplibre-gl/dist/maplibre-gl-worker.mjs", import.meta.url).toString());

// El protocolo pmtiles:// se registra una sola vez por página: lo comparten todos los mapas.
let pmtilesRegistered = false;
function registerPmtiles() {
  if (pmtilesRegistered) return;
  addProtocol("pmtiles", new Protocol().tile);
  pmtilesRegistered = true;
}

export function VectorTileLayer() {
  const map = useMap();

  useEffect(() => {
    registerPmtiles();
    const layer = maplibreGL({ style: mapStyle(window.location.origin) });
    layer.addTo(map);
    map.attributionControl.addAttribution(MAP_ATTRIBUTION);
    return () => {
      map.attributionControl.removeAttribution(MAP_ATTRIBUTION);
      layer.remove();
    };
  }, [map]);

  return null;
}
