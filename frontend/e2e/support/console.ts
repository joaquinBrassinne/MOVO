import { expect, test } from "@playwright/test";

/** Lo que el servidor de desarrollo de Next informa de todas las pantallas del panel desde antes de los cupones. */
const KNOWN_DEV_NOISE = [
  // Un aviso de `next dev` sobre la navegación instantánea; no es un error de la pantalla.
  "Could not validate that a segment in your UI has instant navigation",
  // Una respuesta 404 de la API es esperable en algún test.
  "Failed to load resource",
];

/**
 * Hace fallar los tests del `describe` donde se llama si la página deja errores en la
 * consola (hidratación, valores inestables de Next, excepciones).
 */
export function failOnConsoleErrors() {
  let problems: string[] = [];

  test.beforeEach(({ page }) => {
    problems = [];
    page.on("console", (message) => {
      const text = message.text();
      if (message.type() === "error" && !KNOWN_DEV_NOISE.some((noise) => text.includes(noise))) {
        problems.push(text.slice(0, 600));
      }
    });
    page.on("pageerror", (error) => problems.push(error.message.slice(0, 600)));
  });

  test.afterEach(() => {
    expect(problems).toEqual([]);
  });
}
