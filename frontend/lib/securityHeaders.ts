/** Una regla de cabeceras en el formato de `headers()` de next.config. */
export type HeaderRule = { source: string; headers: { key: string; value: string }[] };

// Sin `script-src`: los scripts de arranque de Next van en línea y todavía sin
// nonce, así que restringirlos rompería la app. Queda como tarea aparte.
const contentSecurityPolicy = [
  "frame-ancestors 'none'",
  "base-uri 'self'",
  "form-action 'self'",
  "object-src 'none'",
].join("; ");

// El escáner de QR del panel usa la cámara y el mapa pide la ubicación: solo
// para este origen. Todo lo demás queda cerrado.
const permissionsPolicy = "camera=(self), geolocation=(self), microphone=(), payment=(), usb=()";

/**
 * Cabeceras de seguridad de todas las rutas. HSTS solo en producción: sobre
 * http local el navegador lo ignora, y activarlo por error en un entorno sin
 * HTTPS dejaría el sitio inaccesible.
 */
export function securityHeaders(production: boolean): HeaderRule[] {
  const headers = [
    { key: "Content-Security-Policy", value: contentSecurityPolicy },
    { key: "X-Frame-Options", value: "DENY" },
    { key: "X-Content-Type-Options", value: "nosniff" },
    // La URL del cupón lleva su token: a otros sitios solo sale el origen.
    { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
    { key: "Permissions-Policy", value: permissionsPolicy },
  ];
  if (production) {
    headers.push({ key: "Strict-Transport-Security", value: "max-age=31536000" });
  }
  return [{ source: "/:path*", headers }];
}
