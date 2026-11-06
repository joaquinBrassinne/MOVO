"use client";

// Con `cacheComponents` Next deja la página anterior oculta (<Activity>) y la vuelve a
// mostrar al volver atrás. Eso suelta y reasigna el nodo del mapa, y Leaflet falla con
// "Map container is being reused by another instance". Los efectos sí se limpian al
// ocultar y se repiten al mostrar, así que el mapa solo existe mientras la página se ve.
import { useEffect, useState, type ReactNode } from "react";

export function MountWhileVisible({ children }: { children: ReactNode }) {
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    // Sincroniza con la visibilidad de <Activity>: no hay otro evento al que suscribirse.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setVisible(true);
    return () => setVisible(false);
  }, []);

  return visible ? children : <div className="size-full animate-pulse bg-movo-card" />;
}
