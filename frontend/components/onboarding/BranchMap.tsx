"use client";

// Mapa con el pin de la sede. Se carga solo en el navegador (ver StepBranch): Leaflet
// necesita `window`. Mapas: OpenStreetMap + React Leaflet (CLAUDE.md, SDD §12.3).

import "leaflet/dist/leaflet.css";

import L from "leaflet";
import { useEffect } from "react";
import { MapContainer, Marker, TileLayer, useMap, useMapEvents } from "react-leaflet";

import { PlusIcon } from "@/components/socios/icons";
import type { MapView, Pin } from "@/lib/business/useBranchForm";

// Los tiles públicos de OSM son para el piloto; en producción la URL cambia por variable.
const TILES_URL =
  process.env.NEXT_PUBLIC_MAP_TILES_URL ?? "https://tile.openstreetmap.org/{z}/{x}/{y}.png";
const ATTRIBUTION = '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>';

const CITY_ZOOM = 12;
const STREET_ZOOM = 17;

// El marcador se dibuja en SVG: así no depende de las imágenes que Leaflet busca por URL.
const PIN_ICON = L.divIcon({
  className: "",
  iconSize: [44, 52],
  iconAnchor: [22, 52],
  html: `<svg width="44" height="52" viewBox="0 0 44 52" aria-hidden="true">
    <path d="M22 51C22 51 3 33 3 20a19 19 0 0 1 38 0c0 13-19 31-19 31Z" fill="#fcd34d" stroke="#23213f" stroke-width="3"/>
    <circle cx="22" cy="20" r="7" fill="#23213f"/>
  </svg>`,
});

interface Props {
  pin: Pin | null;
  view: MapView;
  onMove: (pin: Pin) => void;
}

export default function BranchMap({ pin, view, onMove }: Props) {
  return (
    <MapContainer
      center={[view.latitude, view.longitude]}
      zoom={pin ? STREET_ZOOM : CITY_ZOOM}
      zoomControl={false}
      scrollWheelZoom={false}
      className="size-full"
    >
      <TileLayer url={TILES_URL} attribution={ATTRIBUTION} />
      <FlyTo view={view} zoomIn={pin !== null} />
      <PlaceOnClick onPick={onMove} />
      {pin ? (
        <Marker
          position={[pin.latitude, pin.longitude]}
          icon={PIN_ICON}
          draggable
          eventHandlers={{
            dragend: (event) => {
              const { lat, lng } = (event.target as L.Marker).getLatLng();
              onMove({ latitude: lat, longitude: lng });
            },
          }}
        />
      ) : null}
      <ZoomButtons />
    </MapContainer>
  );
}

/** Lleva el mapa a donde pide la vista cada vez que `nonce` cambia (buscar, ubicarme, recentrar). */
function FlyTo({ view, zoomIn }: { view: MapView; zoomIn: boolean }) {
  const map = useMap();

  useEffect(() => {
    if (view.nonce === 0) return;
    map.flyTo([view.latitude, view.longitude], zoomIn ? STREET_ZOOM : map.getZoom());
    // Solo importa el pedido nuevo: las coordenadas llegan siempre junto con el nonce.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [view.nonce]);

  return null;
}

/** Tocar el mapa mueve el pin ahí, para quien no quiere arrastrarlo. */
function PlaceOnClick({ onPick }: { onPick: (pin: Pin) => void }) {
  useMapEvents({
    click: (event) => onPick({ latitude: event.latlng.lat, longitude: event.latlng.lng }),
  });
  return null;
}

function ZoomButtons() {
  const map = useMap();
  const button =
    "grid size-11 place-items-center rounded-full bg-white text-movo-ink shadow-md outline-none focus-visible:ring-2 focus-visible:ring-movo-accent";

  return (
    <div className="absolute right-3 bottom-8 z-[1000] flex flex-col gap-2">
      <button type="button" aria-label="Acercar" onClick={() => map.zoomIn()} className={button}>
        <PlusIcon />
      </button>
      <button type="button" aria-label="Alejar" onClick={() => map.zoomOut()} className={button}>
        <span aria-hidden="true" className="h-0.5 w-4 rounded bg-movo-ink" />
      </button>
    </div>
  );
}
