import { expect, test, type Page } from "@playwright/test";

import { OWNER, signIn } from "../helpers";
import { failOnConsoleErrors } from "../support/console";

// Sigue a 03-admin: el negocio está activo y todavía no tiene beneficios ni cupo. Los
// tests son el uso real de la vista, en orden. Se miran a 390 px: el panel es mobile primero.
test.describe.configure({ mode: "serial" });
test.use({ viewport: { width: 390, height: 844 } });

const card = (page: Page, text: string) =>
  page.getByTestId("benefit-card").filter({ hasText: text });
const sheet = (page: Page, name: string) => page.getByRole("dialog", { name });

async function open(page: Page) {
  await page
    .getByRole("navigation", { name: "Secciones del panel" })
    .getByRole("link", { name: "Beneficios" })
    .click();
  await expect(page).toHaveURL(/\/socios\/beneficios$/);
}

test.describe("Benefits", () => {
  failOnConsoleErrors();

  test.beforeEach(async ({ page }) => {
    await signIn(page, OWNER);
    await open(page);
  });

  test(
    "A new business sees the empty state and a quota without a limit",
    { tag: ["@high", "@e2e", "@benefits", "@BENEFITS-E2E-001"] },
    async ({ page }) => {
      await expect(
        page.getByRole("heading", { name: "Todavía no creaste beneficios" }),
      ).toBeVisible();
      await expect(page.getByRole("region", { name: "Cupo del mes" })).toContainText("Sin límite");
      await page.screenshot({ path: "test-results/shots/benefits-empty.png", fullPage: true });
    },
  );

  test(
    "The owner creates a percent, a fixed amount and a custom benefit",
    { tag: ["@critical", "@e2e", "@benefits", "@BENEFITS-E2E-002"] },
    async ({ page }) => {
      // Porcentaje: un atajo y condiciones.
      await page.getByRole("button", { name: "Crear mi primer beneficio" }).click();
      const form = sheet(page, "Nuevo beneficio");
      await form.getByRole("button", { name: "15%" }).click();
      await form.getByLabel("Condiciones").fill("Alumnos nuevos en Funcional");
      await page.screenshot({ path: "test-results/shots/benefit-sheet.png" });
      await form.getByRole("button", { name: "Crear beneficio" }).click();
      await expect(card(page, "-15%")).toContainText("15% de descuento");
      await expect(card(page, "-15%")).toContainText("Alumnos nuevos en Funcional");
      await expect(card(page, "-15%")).toContainText("0 generados · 0 canjeados este mes");

      // Monto fijo.
      await page.getByRole("button", { name: "Nuevo beneficio" }).click();
      const amount = sheet(page, "Nuevo beneficio");
      await amount.getByRole("button", { name: "Monto fijo" }).click();
      await amount.getByLabel("Monto de descuento").fill("2000");
      await amount.getByRole("button", { name: "Crear beneficio" }).click();
      await expect(card(page, "-$2.000")).toContainText("$2.000 de descuento");

      // Otro beneficio, con título propio.
      await page.getByRole("button", { name: "Nuevo beneficio" }).click();
      const custom = sheet(page, "Nuevo beneficio");
      await custom.getByRole("button", { name: "Otro beneficio" }).click();
      await custom.getByLabel("Describí el beneficio").fill("Una clase de prueba gratis");
      await custom.getByRole("button", { name: "Crear beneficio" }).click();
      await expect(card(page, "Beneficio")).toContainText("Una clase de prueba gratis");
      await page.screenshot({ path: "test-results/shots/benefits-list.png", fullPage: true });
    },
  );

  test(
    "A percent can be any whole number from 1 to 100 and is edited in place",
    { tag: ["@high", "@e2e", "@benefits", "@BENEFITS-E2E-003"] },
    async ({ page }) => {
      await card(page, "-15%")
        .getByRole("button", { name: /^Editar/ })
        .click();
      const form = sheet(page, "Editar beneficio");
      await form.getByLabel("Porcentaje de descuento").fill("37");
      await form.getByRole("button", { name: "Guardar cambios" }).click();
      await expect(card(page, "-37%")).toContainText("37% de descuento");

      await card(page, "-37%")
        .getByRole("button", { name: /^Editar/ })
        .click();
      const invalid = sheet(page, "Editar beneficio");
      await invalid.getByLabel("Porcentaje de descuento").fill("101");
      await invalid.getByRole("button", { name: "Guardar cambios" }).click();
      await expect(invalid.getByRole("alert")).toContainText("1 a 100");
      await invalid.getByRole("button", { name: "Cerrar" }).click();
    },
  );

  test(
    "Links and phone numbers are refused in the free texts",
    { tag: ["@high", "@e2e", "@benefits", "@BENEFITS-E2E-004"] },
    async ({ page }) => {
      await page.getByRole("button", { name: "Nuevo beneficio" }).click();
      const form = sheet(page, "Nuevo beneficio");
      await form.getByRole("button", { name: "Otro beneficio" }).click();
      await form.getByLabel("Describí el beneficio").fill("Mirá www.promo.com");
      await form.getByRole("button", { name: "Crear beneficio" }).click();
      await expect(form.getByRole("alert")).toContainText("No incluyas links");

      await form.getByLabel("Describí el beneficio").fill("Una clase gratis");
      await form.getByLabel("Condiciones").fill("Llamá al 351 123 4567");
      await form.getByRole("button", { name: "Crear beneficio" }).click();
      await expect(form.getByRole("alert")).toContainText("No incluyas teléfonos");
      await form.getByRole("button", { name: "Cerrar" }).click();
      await expect(card(page, "Llamá")).toHaveCount(0);
    },
  );

  test(
    "The owner's switch turns a benefit off and on",
    { tag: ["@high", "@e2e", "@benefits", "@BENEFITS-E2E-005"] },
    async ({ page }) => {
      const toggle = card(page, "-$2.000").getByRole("switch");
      await expect(toggle).toHaveAttribute("aria-checked", "true");
      await toggle.click();
      await expect(toggle).toHaveAttribute("aria-checked", "false");

      await page.reload();
      await expect(card(page, "-$2.000").getByRole("switch")).toHaveAttribute(
        "aria-checked",
        "false",
      );
      await card(page, "-$2.000").getByRole("switch").click();
      await expect(card(page, "-$2.000").getByRole("switch")).toHaveAttribute(
        "aria-checked",
        "true",
      );
    },
  );

  test(
    "The owner sets a monthly quota for the whole business and then removes it",
    { tag: ["@critical", "@e2e", "@benefits", "@BENEFITS-E2E-006"] },
    async ({ page }) => {
      const region = page.getByRole("region", { name: "Cupo del mes" });
      await region.getByRole("button", { name: "Editar cupo" }).click();
      const quota = sheet(page, "Cupo mensual");
      await quota.getByRole("switch", { name: "Sin límite de cupones" }).click();
      await quota.getByRole("button", { name: "20", exact: true }).click();
      await page.screenshot({ path: "test-results/shots/quota-sheet.png" });
      await quota.getByRole("button", { name: "Guardar" }).click();

      await expect(region).toContainText("de 20");
      await expect(region).toContainText("0% consumido");
      await expect(region).toContainText("20 restantes");
      await page.screenshot({ path: "test-results/shots/benefits-quota.png", fullPage: true });

      await region.getByRole("button", { name: "Editar cupo" }).click();
      await sheet(page, "Cupo mensual")
        .getByRole("switch", { name: "Sin límite de cupones" })
        .click();
      await sheet(page, "Cupo mensual").getByRole("button", { name: "Guardar" }).click();
      await expect(region).toContainText("Sin límite");
    },
  );
});
