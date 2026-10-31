import { expect, test } from "@playwright/test";

import { OWNER, photo, signIn } from "../helpers";

// Sigue a panel.spec.ts: el mismo negocio, activo. El servidor de e2e permite 6 fotos
// (MAX_BUSINESS_IMAGES en e2e/support/env.mjs). El orden de los tests es el uso real.
test.describe.configure({ mode: "serial" });

test.describe("Owner photo gallery", () => {
  test.beforeEach(async ({ page }) => {
    await signIn(page, OWNER);
    await page.goto("/owners/photos");
  });

  test(
    "Deletes the photos left by the onboarding and shows the empty state with the limit",
    { tag: ["@high", "@e2e", "@photos", "@PHOTOS-E2E-001"] },
    async ({ page }) => {
      // El alta dejó fotos cargadas: se eliminan todas, como lo haría el dueño. Primero se espera
      // a que la pantalla termine de cargar, para no contar fotos de un esqueleto.
      await expect(page.getByText(/\d+ de 6 fotos/)).toBeVisible();
      const photos = page.getByRole("img", { name: /del local$/ });
      const menu = page.getByRole("button", { name: "Opciones de la foto 1" });
      for (let left = await photos.count(); left > 0; left--) {
        await menu.click();
        await page.getByRole("menuitem", { name: "Eliminar" }).click();
        await page.getByRole("button", { name: "Eliminar", exact: true }).click();
        // Hasta que la lista se actualiza el menú de la primera foto es otro: no se sigue antes.
        await expect(photos).toHaveCount(left - 1);
      }

      await expect(
        page.getByText("Todavía no subiste fotos. Una buena portada ayuda a que te elijan."),
      ).toBeVisible();
      await expect(page.getByText("0 de 6 fotos")).toBeVisible();
    },
  );

  test(
    "Rejects a file that is not an image before uploading it",
    { tag: ["@high", "@e2e", "@photos", "@PHOTOS-E2E-002"] },
    async ({ page }) => {
      await page.locator('input[type="file"]').setInputFiles({
        name: "notas.txt",
        mimeType: "text/plain",
        buffer: Buffer.from("no soy una foto"),
      });

      await expect(page.getByText("Ese archivo no es una foto válida")).toBeVisible();
      await expect(page.getByText("0 de 6 fotos")).toBeVisible();
    },
  );

  test(
    "Uploads photos and the first one is the cover",
    { tag: ["@critical", "@e2e", "@photos", "@PHOTOS-E2E-003"] },
    async ({ page }) => {
      await page
        .locator('input[type="file"]')
        .setInputFiles([
          photo("a.png", 200, 30, 30),
          photo("b.png", 30, 200, 30),
          photo("c.png", 30, 30, 200),
        ]);

      await expect(page.getByText("3 de 6 fotos")).toBeVisible();
      await expect(page.getByRole("img", { name: "Foto de portada del local" })).toBeVisible();
      await expect(page.getByText("Portada", { exact: true })).toHaveCount(1);
    },
  );

  test(
    "Moves a photo up and chooses another cover from its menu",
    { tag: ["@high", "@e2e", "@photos", "@PHOTOS-E2E-004"] },
    async ({ page }) => {
      await expect(page.getByRole("button", { name: "Subir la foto 1" })).toBeDisabled();
      await expect(page.getByRole("button", { name: "Bajar la foto 3" })).toBeDisabled();

      // Las URLs firmadas cambian en cada carga: la foto se reconoce por la ruta del archivo.
      const fileOf = async (name: string) =>
        new URL((await page.getByRole("img", { name }).getAttribute("src")) ?? "").pathname;

      const third = await fileOf("Foto 3 del local");
      await page.getByRole("button", { name: "Subir la foto 3" }).click();
      await expect.poll(() => fileOf("Foto 2 del local")).toBe(third);

      // Elegir portada desde el menú de la foto 3 la lleva al frente.
      const last = await fileOf("Foto 3 del local");
      await page.getByRole("button", { name: "Opciones de la foto 3" }).click();
      await page.getByRole("menuitem", { name: "Usar como portada" }).click();
      await expect.poll(() => fileOf("Foto de portada del local")).toBe(last);
      await expect(page.getByText("Portada", { exact: true })).toHaveCount(1);

      // La portada no ofrece elegirse a sí misma.
      await page.getByRole("button", { name: "Opciones de la foto 1" }).click();
      await expect(page.getByRole("menuitem", { name: "Usar como portada" })).toHaveCount(0);
    },
  );

  test(
    "Asks before deleting and keeps the photo when the owner cancels",
    { tag: ["@high", "@e2e", "@photos", "@PHOTOS-E2E-005"] },
    async ({ page }) => {
      await page.getByRole("button", { name: "Opciones de la foto 2" }).click();
      await page.getByRole("menuitem", { name: "Eliminar" }).click();
      await expect(page.getByText("¿Eliminar esta foto?")).toBeVisible();

      await page.getByRole("button", { name: "Cancelar" }).click();
      await expect(page.getByText("3 de 6 fotos")).toBeVisible();

      await page.getByRole("button", { name: "Opciones de la foto 2" }).click();
      await page.getByRole("menuitem", { name: "Eliminar" }).click();
      await page.getByRole("button", { name: "Eliminar", exact: true }).click();
      await expect(page.getByText("2 de 6 fotos")).toBeVisible();
    },
  );

  test(
    "Disables the upload tile at the limit and says so",
    { tag: ["@high", "@e2e", "@photos", "@PHOTOS-E2E-006"] },
    async ({ page }) => {
      await page
        .locator('input[type="file"]')
        .setInputFiles([
          photo("d.png", 90, 90, 10),
          photo("e.png", 10, 90, 90),
          photo("f.png", 90, 10, 90),
          photo("g.png", 120, 60, 30),
        ]);

      await expect(page.getByText("6 de 6 fotos")).toBeVisible();
      await expect(page.getByText("Llegaste al máximo de fotos")).toBeVisible();
      await expect(page.locator('input[type="file"]')).toBeDisabled();
    },
  );
});
