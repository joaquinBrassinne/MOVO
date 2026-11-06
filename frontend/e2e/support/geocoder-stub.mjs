// Nominatim de mentira: los E2E nunca llaman al servicio público (SDD §12.3).
// Responde con una coordenada fija, salvo para direcciones que contienen "inexistente".

import { createServer } from "node:http";

import { PORTS } from "./env.mjs";

const PLACE = { lat: "-32.1925", lon: "-64.4198" };

const server = createServer((request, response) => {
  const url = new URL(request.url ?? "/", "http://localhost");
  response.setHeader("Content-Type", "application/json");

  if (url.pathname !== "/search") {
    response.end("{}");
    return;
  }

  const query = url.searchParams.get("q") ?? "";
  const found = !query.toLowerCase().includes("inexistente");
  response.end(JSON.stringify(found ? [{ ...PLACE, display_name: query }] : []));
});

server.listen(PORTS.geocoder, () => console.log(`geocoder stub on :${PORTS.geocoder}`));
