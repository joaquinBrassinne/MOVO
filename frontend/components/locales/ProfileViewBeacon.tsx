"use client";

import { useEffect } from "react";

// Las fichas que ya avisaron en esta carga de la página: en desarrollo React monta dos
// veces cada efecto, y una visita no puede contar doble.
const sent = new Set<string>();

/**
 * Avisa a la API que alguien vio la ficha, una sola vez por carga y después de montar. El
 * render del servidor (rastreadores, vistas previas de enlaces) no pasa por acá, así que
 * no cuenta. No dibuja nada.
 */
export function ProfileViewBeacon({ slug }: { slug: string }) {
  useEffect(() => {
    if (sent.has(slug)) return;
    sent.add(slug);

    const url = `/api/v1/businesses/${slug}/view`;
    if (navigator.sendBeacon?.(url)) return;
    void fetch(url, { method: "POST", keepalive: true }).catch(() => {
      // Contar una vista nunca es motivo para molestar a quien mira la ficha.
    });
  }, [slug]);

  return null;
}
