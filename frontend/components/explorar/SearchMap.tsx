"use client";

// Mapa de búsqueda. Se carga solo en el navegador (ver MapScreen): Leaflet necesita
// `window`. Mapas: OpenStreetMap + React Leaflet (CLAUDE.md, SDD §12.3).

import "leaflet/dist/leaflet.css";

import L from "leaflet";
import { useEffect, useRef } from "react";
import { CircleMarker, MapContainer, Marker, useMap, useMapEvents } from "react-leaflet";

import { PlusIcon } from "@/components/socios/icons";
import { VectorTileLayer } from "@/components/map/VectorTileLayer";
import type { Point, SearchResult } from "@/lib/search/types";

import { categoryIconName, iconMarkup } from "./icons";

const ZOOM = 14;

interface Props {
  center: Point;
  /** Cambia cuando hay que volver a llevar el mapa a `center` aunque no se haya movido. */
  recenterKey: string;
  results: SearchResult[];
  selectedId: number | null;
  user: Point | null;
  onSelect: (branchId: number) => void;
  onMoved: (center: Point) => void;
  onLocate: () => void;
}

function escapeHtml(text: string): string {
  return text.replace(/[&<>"']/g, (c) => `&#${c.charCodeAt(0)};`);
}

// El marcador se dibuja en HTML/SVG: así no depende de las imágenes que Leaflet busca por URL.
function pinIcon(result: SearchResult, selected: boolean): L.DivIcon {
  const size = selected ? 56 : 44;
  const label = selected
    ? `<span class="absolute bottom-full left-1/2 mb-2 flex -translate-x-1/2 items-center gap-2 whitespace-nowrap rounded-full bg-white px-3 py-1.5 text-sm font-bold text-movo-ink shadow-md"><span class="size-2 rounded-full bg-movo-accent"></span>${escapeHtml(result.business_name)}</span>`
    : "";
  const star = result.featured
    ? `<span class="absolute -top-1 -right-1 grid size-5 place-items-center rounded-full bg-movo-ink text-movo-accent"><svg viewBox="0 0 24 24" class="size-3" fill="currentColor"><path d="m12 2 3.100 6.300 6.900 1-5 4.900 1.200 6.800L12 17.700 5.800 21l1.200-6.800-5-4.900 6.900-1L12 2Z"/></svg></span>`
    : "";
  const ring = selected ? "ring-8 ring-movo-accent/30" : "";

  return L.divIcon({
    className: "",
    iconSize: [size, size],
    iconAnchor: [size / 2, size / 2],
    html: `<span class="relative grid place-items-center rounded-full bg-movo-accent text-movo-ink shadow-lg ${ring}" style="width:${size}px;height:${size}px">
      ${label}
      <svg viewBox="0 0 24 24" class="${selected ? "size-7" : "size-6"}" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${iconMarkup(categoryIconName(result.category_slug))}</svg>
      ${star}
    </span>`,
  });
}

export default function SearchMap({
  center,
  recenterKey,
  results,
  selectedId,
  user,
  onSelect,
  onMoved,
  onLocate,
}: Props) {
  return (
    <MapContainer
      center={[center.lat, center.lng]}
      zoom={ZOOM}
      zoomControl={false}
      className="size-full"
    >
      <VectorTileLayer />
      <FlyTo center={center} recenterKey={recenterKey} />
      <MoveWatcher onMoved={onMoved} />
      {user ? (
        <CircleMarker
          center={[user.lat, user.lng]}
          radius={8}
          pathOptions={{ color: "#ffffff", weight: 3, fillColor: "#3b82f6", fillOpacity: 1 }}
        />
      ) : null}
      {results.map((result) => (
        <Marker
          key={result.branch_id}
          position={[result.lat, result.lng]}
          icon={pinIcon(result, result.branch_id === selectedId)}
          zIndexOffset={result.branch_id === selectedId ? 1000 : 0}
          title={result.business_name}
          eventHandlers={{ click: () => onSelect(result.branch_id) }}
        />
      ))}
      <Controls onLocate={onLocate} />
    </MapContainer>
  );
}

/** Lleva el mapa a `center` cada vez que cambia la clave (menos al montar, que ya arranca ahí). */
function FlyTo({ center, recenterKey }: { center: Point; recenterKey: string }) {
  const map = useMap();
  const first = useRef(true);

  useEffect(() => {
    if (first.current) {
      first.current = false;
      return;
    }
    map.flyTo([center.lat, center.lng], Math.max(map.getZoom(), ZOOM));
    // Solo importa el pedido nuevo: el centro llega siempre junto con la clave.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [recenterKey]);

  return null;
}

function MoveWatcher({ onMoved }: { onMoved: (center: Point) => void }) {
  const map = useMapEvents({
    moveend: () => {
      const { lat, lng } = map.getCenter();
      onMoved({ lat, lng });
    },
  });
  return null;
}

function Controls({ onLocate }: { onLocate: () => void }) {
  const map = useMap();
  const button =
    "grid size-12 place-items-center bg-white text-movo-ink shadow-md outline-none focus-visible:ring-2 focus-visible:ring-movo-accent";

  return (
    <div className="absolute right-4 bottom-52 z-[1000] flex flex-col items-center gap-3">
      <button
        type="button"
        aria-label="Usar mi ubicación"
        onClick={onLocate}
        className={`${button} rounded-full`}
      >
        <svg
          viewBox="0 0 24 24"
          className="size-6"
          fill="none"
          stroke="currentColor"
          strokeWidth={2}
          strokeLinecap="round"
          aria-hidden="true"
        >
          <circle cx="12" cy="12" r="3" />
          <circle cx="12" cy="12" r="8" />
          <path d="M12 2v3M12 19v3M2 12h3M19 12h3" />
        </svg>
      </button>
      <div className="flex flex-col overflow-hidden rounded-3xl shadow-md">
        <button type="button" aria-label="Acercar" onClick={() => map.zoomIn()} className={button}>
          <PlusIcon />
        </button>
        <button type="button" aria-label="Alejar" onClick={() => map.zoomOut()} className={button}>
          <span aria-hidden="true" className="h-0.5 w-4 rounded bg-movo-ink" />
        </button>
      </div>
    </div>
  );
}
