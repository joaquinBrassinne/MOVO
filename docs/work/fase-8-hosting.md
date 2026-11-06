# Fase 8 — Hosting para el piloto (8.7)

Investigado en octubre de 2026 con precios publicados por cada proveedor (sin IVA salvo que se indique).
Volumen del piloto: 5 a 10 locales. Mapa de Córdoba 66 MB + 12 MB de tipografías (el país entero, ~1,1 GB); imágenes de locales: pocos GB como mucho.

## Decisión

Un solo VPS **Hetzner CX23** (2 vCPU, 4 GB de RAM, 40 GB NVMe, €5,49/mes) con todo adentro: Postgres + PostGIS, Redis, API, worker, Next y Caddy.
**Cloudflare** delante: DNS, CDN y R2 (imágenes y archivo PMTiles). Volcados de Postgres fuera del servidor en R2 o B2.

## Costo mensual estimado

| Pieza | Opción elegida | Costo |
|---|---|---|
| VPS | Hetzner CX23, 20 TB de tráfico incluidos | €5,49 (USD 6,49) + IPv4 (precio sin confirmar) |
| Bucket (imágenes + mapa) | Cloudflare R2: 10 GB gratis, salida siempre gratis | $0 (después $0,015/GB-mes) |
| CDN y DNS | Cloudflare (plan gratuito, sin confirmar en esta ronda) | $0 |
| Respaldos fuera del servidor | Volcados diarios a R2 o B2 (10 GB gratis en ambos) | $0 |
| Dominio | `.com.ar` en NIC Argentina, ARS 8.500/año, requiere CUIT/CUIL con Clave Fiscal nivel 2+ | < USD 1/mes |
| **Total piloto** | | **≈ USD 7 a 8** |

## Alternativas consideradas

| Opción | USD/mes | Por qué no |
|---|---|---|
| OVHcloud VPS-1 (2 vCPU / 4 GB, IPv4 incluida) | 4,54 | Plan B más barato; se prefirió Hetzner por trayectoria y tráfico incluido |
| Contabo Cloud VPS 4 (4 vCPU / 8 GB) | 6,19 (€5,50 con IVA) | Recursos compartidos y tráfico de uso justo |
| Vultr regular 2/4 (tiene São Paulo, ~21 ms) | 20 | Mejor latencia, 3 veces el costo |
| DigitalOcean / Linode / Lightsail 2/4 | 24 | Precio |
| Backblaze B2 como bucket | $0 hasta 10 GB, luego $6,95/TB; salida gratis hasta 3× lo almacenado | R2 no cobra salida |
| Hetzner Storage Box BX11 (1 TB) | €3,20 | Los volcados caben gratis en R2/B2 |
| Cloudflare Workers/Containers + Postgres en alwaysData | ≥ 11 | Ver abajo |

### Descartado: todo en Cloudflare + Postgres en alwaysData
- El plan gratuito de alwaysData prohíbe el uso con fines de lucro (1 GB de disco, 256 MB de RAM, 1/4 de CPU, 3 días de backups). El plan Small cuesta €5/mes (1 GB de RAM, 50 GB, 7 días de backups); PostGIS se activa con un clic.
- La API en Go no corre en Workers: haría falta Cloudflare Containers (cobro por uso sobre el plan Workers Paid de $5/mes), y el worker con lock consultivo necesita un proceso vivo.
- Next 16 con `cacheComponents` en Workers requeriría OpenNext, sin verificar.
- La base en Francia y la API en el borde sumarían ~200 ms por consulta desde Sudamérica.

## Qué queda a cargo del operador

Copias fuera del servidor y prueba de restauración (8.3), parches del sistema y de las imágenes, alerta externa sobre `/readyz` (8.2), firewall con solo 80/443/SSH (8.1, 8.4) y los topes ya medidos en la Fase 7: `DB_MAX_CONNS` ≤ 10, 1–2 GB de swap, nada se compila en el VPS.

## Criterio para pasar a algo mayor

- Subir a **CX33** (€8,49, reescalado) con RAM sostenida sobre ~70 %, swap en uso o p95 de búsqueda degradándose.
- Evaluar Postgres gestionado en la misma región que la API solo cuando una caída de unos minutos ya cueste plata a un local.
- Si la latencia desde Argentina molesta en lo dinámico, mover la API a un VPS en São Paulo (Vultr) y mantener R2/CDN.

## Pendiente de confirmar al contratar

Precio del IPv4 de Hetzner, plan gratuito de Cloudflare (DNS/CDN), cotización en pesos del dominio, y que el pgBouncer de alwaysData no aplique (solo si se reconsidera esa opción).
