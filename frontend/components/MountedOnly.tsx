"use client";

import { useEffect, useState, type ReactNode } from "react";

/**
 * Renderiza a los hijos solo mientras los efectos de la página están activos.
 *
 * `MapContainer` de react-leaflet 5 destruye el mapa al limpiar su efecto pero conserva la
 * instancia en su estado. Cuando React vuelve a activar la página (navegar y volver con
 * `cacheComponents`, Fast Refresh) reutiliza un mapa ya destruido y Leaflet falla con
 * "Map container is being reused". Desmontar los hijos en la limpieza fuerza un mapa nuevo.
 */
export function MountedOnly({ children }: { children: ReactNode }) {
  const [active, setActive] = useState(false);

  useEffect(() => {
    // Activar en el efecto es el punto: así los hijos solo existen mientras el efecto vive.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setActive(true);
    return () => setActive(false);
  }, []);

  return active ? children : null;
}
