import { PORTS } from "./env.mjs";

// `next dev` compila cada ruta la primera vez que la piden. Se las pide acá una vez para
// que esa demora no cuente contra el tiempo de ningún test.
const ROUTES = [
  "/socios/ingresar",
  "/socios/crear-cuenta",
  "/socios/onboarding",
  "/socios/panel",
  "/socios/servicios",
  "/socios/sedes",
  "/socios/sedes/nueva",
  "/socios/sedes/1",
  "/admin/locales",
];

export default async function warmUp() {
  for (const route of ROUTES) {
    await fetch(`http://localhost:${PORTS.web}${route}`).catch(() => undefined);
  }
}
