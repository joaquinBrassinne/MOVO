// Carga de la búsqueda pública (GET /search), SDD 7.5.
//
// Ejecutar con `make load-search` (o `k6 run backend/loadtest/search.js`). Variables:
//   BASE_URL   raíz de la API, por defecto http://localhost:8080 (sin /api/v1)
//   PEAK_VUS   usuarios virtuales en el pico, por defecto 50
//
// Los orígenes se reparten alrededor del centro de la zona densa que siembra
// `make seed-dev` (Río Tercero, cmd/seed-dev/seed.sql), en una grilla fija para que
// las corridas sean comparables entre sí. Son ~100 puntos distintos: cada uno cae en
// una celda de caché distinta de SEARCH_CACHE_CELL_DEG, así que la tasa de aciertos
// depende de cuántas personas buscan desde la misma zona, como en producción.
import http from 'k6/http';
import { check } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const PEAK_VUS = Number(__ENV.PEAK_VUS || 50);

// Centro de la zona densa (seed-dev): mismo que usa la medición de la Fase 3.
const CENTER_LAT = -32.1738;
const CENTER_LNG = -64.1146;
const STEP_DEG = 0.02; // ~2,2 km entre orígenes contiguos
const GRID = 10; // 10 x 10 orígenes, de -0,09° a +0,09° alrededor del centro

const ORIGINS = [];
for (let i = 0; i < GRID; i++) {
  for (let j = 0; j < GRID; j++) {
    ORIGINS.push({
      lat: CENTER_LAT + (i - GRID / 2) * STEP_DEG,
      lng: CENTER_LNG + (j - GRID / 2) * STEP_DEG,
    });
  }
}

export const options = {
  stages: [
    { duration: '30s', target: Math.round(PEAK_VUS / 5) },
    { duration: '1m', target: PEAK_VUS },
    { duration: '2m', target: PEAK_VUS },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    // Presupuesto de la Fase 3: p95 < 300 ms en la búsqueda.
    'http_req_duration{name:search}': ['p(95)<300'],
    http_req_failed: ['rate<0.01'],
  },
  // Cuenta p50/p95 también en el resumen final, no solo los del por defecto de k6.
  summaryTrendStats: ['avg', 'min', 'med', 'p(90)', 'p(95)', 'max'],
};

// Mayoría a 5 km (lo que la app pide al abrir el mapa) y una parte a 50 km (el peor caso
// medido en la Fase 3). El reparto es determinista por iteración.
function radiusFor(iter) {
  return iter % 5 === 0 ? 50000 : 5000;
}

export default function () {
  const origin = ORIGINS[Math.floor(Math.random() * ORIGINS.length)];
  const radius = radiusFor(__ITER);
  const url =
    `${BASE_URL}/api/v1/search?lat=${origin.lat.toFixed(4)}&lng=${origin.lng.toFixed(4)}` +
    `&radius_m=${radius}&limit=21&offset=0`;

  const res = http.get(url, { tags: { name: 'search' } });

  check(res, {
    'status is 200': (r) => r.status === 200,
    'body has results': (r) => r.status === 200 && Array.isArray(r.json('results')),
  });
}
