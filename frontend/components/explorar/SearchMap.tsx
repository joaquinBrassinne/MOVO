"use client";

// Mapa de búsqueda. Se carga solo en el navegador (ver MapScreen): Leaflet necesita
// `window`. Mapas: OpenStreetMap + React Leaflet (CLAUDE.md, SDD §12.3).

import "leaflet/dist/leaflet.css";

import L from "leaflet";
import { useEffect, useRef } from "react";
import { CircleMarker, MapContainer, Marker, useMap, useMapEvents } from "react-leaflet";

import { MountedOnly } from "@/components/MountedOnly";
import { PlusIcon } from "@/components/socios/icons";
import { formatDistance } from "@/lib/search/format";
import { MountWhileVisible } from "@/components/map/MountWhileVisible";
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
}

// El marcador se dibuja en HTML/SVG: así no depende de las imágenes que Leaflet busca por URL.
function pinIcon(result: SearchResult, selected: boolean): L.DivIcon {
  const tone = selected
    ? "z-10 border-movo-accent bg-movo-accent text-movo-ink"
    : "border-movo-muted/35 bg-movo-page text-white";
  // Un ancla de 0×0 en el punto del local; la píldora se alza sobre él con la punta abajo.
  return L.divIcon({
    className: "",
    iconSize: [0, 0],
    iconAnchor: [0, 0],
    html: `<span class="absolute bottom-2 left-0 flex -translate-x-1/2 items-center gap-1.5 whitespace-nowrap rounded-xl border-[1.5px] px-2.5 py-1.5 text-xs font-bold shadow-lg ${tone}">
      <svg viewBox="0 0 24 24" class="size-3.5" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${iconMarkup(categoryIconName(result.category_slug))}</svg>
      ${formatDistance(result.distance_m)}
      <span class="absolute top-full left-1/2 -mt-1.5 size-2.5 -translate-x-1/2 rotate-45 border-r-[1.5px] border-b-[1.5px] ${tone}"></span>
    </span>`,
  });
}

export default function SearchMap(props: Props) {
  return (
    <MountedOnly>
      <SearchMapView {...props} />
    </MountedOnly>
  );
}

function SearchMapView({
  center,
  recenterKey,
  results,
  selectedId,
  user,
  onSelect,
  onMoved,
}: Props) {
  return (
    <MountWhileVisible>
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
            pathOptions={{ color: "#181733", weight: 3, fillColor: "#fcd34d", fillOpacity: 1 }}
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
        <Controls />
      </MapContainer>
    </MountWhileVisible>
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

function Controls() {
  const map = useMap();
  const button =
    "grid size-11 place-items-center bg-movo-page text-white shadow-md outline-none focus-visible:ring-2 focus-visible:ring-movo-accent";

  return (
    <div className="absolute right-4 bottom-[18.5rem] z-[1000] flex flex-col overflow-hidden rounded-2xl shadow-md">
      <button type="button" aria-label="Acercar" onClick={() => map.zoomIn()} className={button}>
        <PlusIcon />
      </button>
      <button type="button" aria-label="Alejar" onClick={() => map.zoomOut()} className={button}>
        <span aria-hidden="true" className="h-0.5 w-4 rounded bg-white" />
      </button>
    </div>
  );
}
