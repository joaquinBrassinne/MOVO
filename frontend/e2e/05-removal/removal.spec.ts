import { expect, test, type Page } from "@playwright/test";

import { ADMIN, BUSINESS, OWNER, signIn } from "../helpers";

// Cierra el ciclo de vida del local de los specs anteriores: la baja es definitiva.
test.describe.configure({ mode: "serial" });

const detail = (page: Page) =>
  page.getByRole("complementary", { name: `Detalle de ${BUSINESS.name}` });

async function openDetail(page: Page) {
  await signIn(page, ADMIN);
  await page.getByRole("button", { name: new RegExp(BUSINESS.name) }).click();
  await expect(detail(page)).toBeVisible();
  return detail(page);
}

test.describe("Business removal", () => {
  test(
    "Cancelling the confirmation keeps the business active",
    { tag: ["@high", "@e2e", "@removal", "@REMOVAL-E2E-001"] },
    async ({ page }) => {
      const panel = await openDetail(page);

      page.once("dialog", (dialog) => void dialog.dismiss());
      await panel.getByRole("button", { name: "Dar de baja" }).click();

      await expect(panel.getByRole("button", { name: "Pausar el local" })).toBeVisible();
      await expect(panel.getByText("Dado de baja")).toHaveCount(0);
    },
  );

  test(
    "The super admin removes the business and the action is final",
    { tag: ["@critical", "@e2e", "@removal", "@REMOVAL-E2E-002"] },
    async ({ page }) => {
      const panel = await openDetail(page);

      page.once("dialog", (dialog) => void dialog.accept());
      await panel.getByRole("button", { name: "Dar de baja" }).click();

      await expect(panel.getByText("Este local está dado de baja.")).toBeVisible();
      await expect(panel.getByText("Dado de baja", { exact: true })).toBeVisible();
      await expect(panel.getByRole("button", { name: "Pausar el local" })).toHaveCount(0);
      await expect(panel.getByRole("button", { name: "Reactivar el local" })).toHaveCount(0);
      await expect(panel.getByRole("switch", { name: "Insignia verificado" })).toHaveCount(0);
      await expect(page.getByText("0 locales registrados")).toBeVisible();
    },
  );

  test(
    "The owner of a removed business is told so and cannot continue",
    { tag: ["@critical", "@e2e", "@removal", "@REMOVAL-E2E-003"] },
    async ({ page }) => {
      await signIn(page, OWNER);
      await page.goto("/socios/onboarding");

      await expect(page.getByText("Este negocio fue dado de baja")).toBeVisible();
      await expect(page.getByRole("button", { name: "Siguiente" })).toHaveCount(0);
    },
  );
});
