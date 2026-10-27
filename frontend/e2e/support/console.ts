import { expect, test } from "@playwright/test";

/**
 * Hace fallar los tests del `describe` donde se llama si la página deja errores en la
 * consola (hidratación, valores inestables de Next, excepciones). Una respuesta 404 de la API
 * es esperable en algún test y no es un error de la pantalla.
 */
export function failOnConsoleErrors() {
  let problems: string[] = [];

  test.beforeEach(({ page }) => {
    problems = [];
    page.on("console", (message) => {
      if (message.type() === "error" && !message.text().includes("Failed to load resource")) {
        problems.push(message.text());
      }
    });
    page.on("pageerror", (error) => problems.push(error.message));
  });

  test.afterEach(() => {
    expect(problems).toEqual([]);
  });
}
