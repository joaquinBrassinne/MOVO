import { expect, test, type Page } from "@playwright/test";

import { ADMIN, BRANCH, BUSINESS, OWNER, signIn } from "../helpers";

// Sigue al alta: el negocio de 02-onboarding está en revisión. El orden de los tests es el
// ciclo de vida del local: se aprueba, se verifica, se marca piloto y se pausa.
test.describe.configure({ mode: "serial" });

const row = (page: Page) => page.getByRole("button", { name: new RegExp(BUSINESS.name) });
const tableRow = (page: Page) => page.getByRole("row").filter({ hasText: BUSINESS.name });
const detail = (page: Page) =>
  page.getByRole("complementary", { name: `Detalle de ${BUSINESS.name}` });

async function openDetail(page: Page) {
  await row(page).click();
  await expect(detail(page)).toBeVisible();
  return detail(page);
}

test.describe("Admin locales", () => {
  test.beforeEach(async ({ page }) => {
    await signIn(page, ADMIN);
    await expect(page).toHaveURL(/\/admin\/locales$/);
  });

  test(
    "Lists the business that is waiting for review with the counts per status",
    { tag: ["@critical", "@e2e", "@admin", "@ADMIN-E2E-001"] },
    async ({ page }) => {
      await expect(page.getByText("1 locales registrados")).toBeVisible();
      await expect(row(page)).toContainText(OWNER.email);
      await expect(row(page)).toContainText("#1");

      await expect(page.getByRole("tab", { name: "Todos (1)" })).toBeVisible();
      await expect(page.getByRole("tab", { name: "En revisión (1)" })).toBeVisible();
      await expect(page.getByRole("tab", { name: "Activos (0)" })).toBeVisible();
      await expect(page.getByRole("tab", { name: "Pausados (0)" })).toBeVisible();
      await expect(page.getByText("Mostrando 1-1 de 1 locales")).toBeVisible();
    },
  );

  test(
    "Searches by name, owner email and shows an empty state when nothing matches",
    { tag: ["@high", "@e2e", "@admin", "@ADMIN-E2E-002"] },
    async ({ page }) => {
      const search = page.getByRole("searchbox", { name: "Buscar locales" });

      await search.fill("aire libre");
      await expect(row(page)).toBeVisible();

      await search.fill(OWNER.email);
      await expect(row(page)).toBeVisible();

      await search.fill("zzzz-no-existe");
      await expect(page.getByText("No hay locales que coincidan con los filtros.")).toBeVisible();
      await expect(row(page)).toHaveCount(0);

      await search.fill("");
      await expect(row(page)).toBeVisible();
    },
  );

  test(
    "Filters by status tab and by category",
    { tag: ["@high", "@e2e", "@admin", "@ADMIN-E2E-003"] },
    async ({ page }) => {
      await page.getByRole("tab", { name: "Activos (0)" }).click();
      await expect(page.getByText("No hay locales que coincidan con los filtros.")).toBeVisible();

      await page.getByRole("tab", { name: "En revisión (1)" }).click();
      await expect(row(page)).toBeVisible();

      await page.getByRole("tab", { name: "Todos (1)" }).click();
      const category = page.getByLabel("Categoría");
      await category.selectOption({ label: "Pilates" });
      await expect(page.getByText("No hay locales que coincidan con los filtros.")).toBeVisible();

      await category.selectOption({ label: BUSINESS.category });
      await expect(row(page)).toBeVisible();
    },
  );

  test(
    "The detail shows contact, branch, services, payment and an empty history",
    { tag: ["@critical", "@e2e", "@admin", "@ADMIN-E2E-004"] },
    async ({ page }) => {
      const panel = await openDetail(page);

      await expect(panel).toContainText("ID #1");
      await expect(panel).toContainText("En revisión");
      await expect(panel.getByRole("link", { name: OWNER.email })).toHaveAttribute(
        "href",
        `mailto:${OWNER.email}`,
      );
      await expect(panel.getByRole("link", { name: /Abrir WhatsApp \(\+549/ })).toHaveAttribute(
        "href",
        /^https:\/\/wa\.me\/549\d+$/,
      );
      await expect(panel).toContainText(`${BRANCH.address}, ${BRANCH.city} (${BRANCH.reference})`);
      await expect(panel.getByRole("link", { name: "Reservas" })).toHaveAttribute(
        "href",
        `${BRANCH.booking}`,
      );
      await expect(panel).toContainText(`${BUSINESS.category} · 2 de 2 servicios activos`);
      await expect(panel).toContainText("Email en Mercado Pago: ana@example.com");
      await expect(panel).toContainText("Nombre del titular: Ana Pérez");
      for (const check of [
        "Mail de Mercado Pago con el mismo email",
        "Plan y monto correctos",
        "Perfil revisado",
      ]) {
        await panel.getByLabel(check).check();
      }
      await expect(panel).toContainText("Todavía no hay acciones registradas.");
      await expect(
        panel.getByRole("button", { name: "Aprobar comercio y publicar" }),
      ).toBeVisible();

      await panel.getByRole("button", { name: "Cerrar el detalle" }).click();
      await expect(detail(page)).toHaveCount(0);
    },
  );

  test(
    "Approving publishes the business and records it in the history",
    { tag: ["@critical", "@e2e", "@admin", "@ADMIN-E2E-005"] },
    async ({ page }) => {
      const panel = await openDetail(page);

      await panel.getByRole("button", { name: "Aprobar comercio y publicar" }).click();

      await expect(panel.getByRole("button", { name: "Pausar el local" })).toBeVisible();
      await expect(panel.getByRole("button", { name: "Aprobar comercio y publicar" })).toHaveCount(
        0,
      );
      await expect(panel.getByText("Aprobado y publicado")).toBeVisible();
      await expect(panel.getByText(new RegExp(ADMIN.email))).toBeVisible();
      await expect(page.getByRole("tab", { name: "Activos (1)" })).toBeVisible();
      await expect(page.getByRole("tab", { name: "En revisión (0)" })).toBeVisible();
    },
  );

  test(
    "Marks the business as verified and as pilot, and undoes both",
    { tag: ["@high", "@e2e", "@admin", "@ADMIN-E2E-006"] },
    async ({ page }) => {
      const panel = await openDetail(page);
      const verified = panel.getByRole("switch", { name: "Insignia verificado" });
      const pilot = panel.getByRole("switch", { name: "Acceso piloto" });
      await expect(verified).toHaveAttribute("aria-checked", "false");

      await verified.click();
      await expect(verified).toHaveAttribute("aria-checked", "true");
      await expect(panel.getByText("Marcado como verificado")).toBeVisible();

      await pilot.click();
      await expect(pilot).toHaveAttribute("aria-checked", "true");
      await expect(panel.getByText("Marcado como piloto")).toBeVisible();
      await expect(tableRow(page)).toContainText("Piloto");

      await pilot.click();
      await expect(pilot).toHaveAttribute("aria-checked", "false");
      await expect(panel.getByText("Piloto quitado")).toBeVisible();
      await expect(tableRow(page)).not.toContainText("Piloto");

      await verified.click();
      await expect(verified).toHaveAttribute("aria-checked", "false");
      await expect(panel.getByText("Verificación quitada")).toBeVisible();
    },
  );

  test(
    "Pauses and reactivates the business",
    { tag: ["@critical", "@e2e", "@admin", "@ADMIN-E2E-007"] },
    async ({ page }) => {
      const panel = await openDetail(page);

      await panel.getByRole("button", { name: "Pausar el local" }).click();
      await expect(panel.getByRole("button", { name: "Reactivar el local" })).toBeVisible();
      await expect(panel.getByText("Pausado", { exact: true }).first()).toBeVisible();
      await expect(page.getByRole("tab", { name: "Pausados (1)" })).toBeVisible();

      await panel.getByRole("button", { name: "Reactivar el local" }).click();
      await expect(panel.getByRole("button", { name: "Pausar el local" })).toBeVisible();
      await expect(panel.getByText("Reactivado")).toBeVisible();
      await expect(page.getByRole("tab", { name: "Activos (1)" })).toBeVisible();
    },
  );

  test(
    "Keeps the full history, newest first",
    { tag: ["@medium", "@e2e", "@admin", "@ADMIN-E2E-008"] },
    async ({ page }) => {
      const panel = await openDetail(page);
      const entries = panel.getByRole("list").last().getByRole("listitem");

      await expect(entries).toHaveCount(7);
      const actions = (await entries.locator("p:first-child").allTextContents()).map((t) =>
        t.trim(),
      );
      expect(actions).toEqual([
        "Reactivado",
        "Pausado",
        "Verificación quitada",
        "Piloto quitado",
        "Marcado como piloto",
        "Marcado como verificado",
        "Aprobado y publicado",
      ]);
    },
  );
});
