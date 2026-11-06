import { PORTS } from "./env.mjs";
import { writeQrVideo } from "./fake-camera.mjs";

// `next dev` compila cada ruta la primera vez que la piden. Se las pide acá una vez para
// que esa demora no cuente contra el tiempo de ningún test.
const ROUTES = [
  "/owners/sign-in",
  "/owners/sign-up",
  "/owners/onboarding",
  "/owners/dashboard",
  "/owners/redeem",
  "/owners/benefits",
  "/owners/business",
  "/owners/services",
  "/owners/branches",
  "/owners/branches/new",
  "/owners/branches/1",
  "/admin/businesses",
];

export default async function warmUp() {
  // Chromium abre el archivo de la cámara falsa al arrancar: tiene que existir.
  writeQrVideo("PLACEHOLDER");
  for (const route of ROUTES) {
    await fetch(`http://localhost:${PORTS.web}${route}`).catch(() => undefined);
  }
}
