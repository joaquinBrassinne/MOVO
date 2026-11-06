"use client";

import { useEffect, useRef } from "react";

// Los servicios que ya avisaron en esta carga de la página: en desarrollo React monta dos
// veces cada efecto, y una vista no puede contar doble.
const sent = new Set<string>();

/**
 * Avisa a la API que un servicio estuvo a la vista, una sola vez por carga y solo cuando el
 * visitante lo ve en pantalla. El render del servidor (rastreadores, vistas previas de enlaces)
 * no pasa por acá, así que no cuenta. Dibuja un marcador invisible.
 */
export function ServiceViewBeacon({
  branchId,
  serviceId,
}: {
  branchId: number;
  serviceId: number;
}) {
  const marker = useRef<HTMLSpanElement>(null);

  useEffect(() => {
    const node = marker.current;
    if (!node) return;
    const key = `${branchId}/${serviceId}`;

    const send = () => {
      if (sent.has(key)) return;
      sent.add(key);
      const url = `/api/v1/branches/${branchId}/services/${serviceId}/view`;
      if (navigator.sendBeacon?.(url)) return;
      void fetch(url, { method: "POST", keepalive: true }).catch(() => {
        // Contar una vista nunca es motivo para molestar a quien mira la ficha.
      });
    };

    if (typeof IntersectionObserver === "undefined") {
      send();
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) {
          send();
          observer.disconnect();
        }
      },
      { threshold: 0.5 },
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, [branchId, serviceId]);

  // Cubre la tarjeta del servicio (que debe ser `relative`) para medir cuánto de ella se ve.
  return <span ref={marker} aria-hidden="true" className="pointer-events-none absolute inset-0" />;
}
