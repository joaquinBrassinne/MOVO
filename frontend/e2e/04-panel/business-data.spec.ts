import { expect, test } from "@playwright/test";

import { ADMIN, BUSINESS, OWNER, signIn } from "../helpers";
import { failOnConsoleErrors } from "../support/console";
import { PORTS } from "../support/env.mjs";

// Sigue a 03-admin: el negocio está activo. Se cambia solo la descripción para no alterar lo
// que otros specs leen (nombre, categoría y teléfono).
test.describe.configure({ mode: "serial" });
test.use({ viewport: { width: 390, height: 844 } });

const API = `http://localhost:${PORTS.api}/api/v1`;
const NEW_DESCRIPTION = "Yoga y pilates en un espacio luminoso, con clases para todos los niveles.";

async function openData(page: import("@playwright/test").Page) {
  await signIn(page, OWNER);
  await page
    .getByRole("navigation", { name: "Secciones del panel" })
    .getByRole("link", { name: "Mi local" })
    .click();
  await page
    .getByRole("main")
    .getByRole("link", { name: /Datos del negocio/ })
    .click();
  await expect(page).toHaveURL(/\/owners\/business\/details$/);
}

test.describe("Business data", () => {
  failOnConsoleErrors();

  test(
    "The owner sees their data, edits the description and the change survives a reload",
    { tag: ["@critical", "@e2e", "@business-data", "@DATA-E2E-001"] },
    async ({ page }) => {
      await openData(page);
      await expect(page.getByRole("heading", { level: 1 })).toHaveText("Datos del negocio");
      await expect(page.getByLabel("Nombre del negocio")).toHaveValue(BUSINESS.name);
      await expect(page.getByRole("radio", { name: BUSINESS.category })).toBeChecked();
      await page.screenshot({ path: "test-results/shots/business-data.png", fullPage: true });

      await page.getByLabel("Descripción").fill(NEW_DESCRIPTION);
      await page.getByRole("button", { name: "Guardar cambios" }).click();
      await expect(page.getByRole("status")).toContainText("Guardamos los cambios");

      await page.reload();
      await expect(page.getByLabel("Descripción")).toHaveValue(NEW_DESCRIPTION);
    },
  );

  test(
    "A blank name is refused before anything is sent",
    { tag: ["@high", "@e2e", "@business-data", "@DATA-E2E-002"] },
    async ({ page }) => {
      await openData(page);

      await page.getByLabel("Nombre del negocio").fill("   ");
      await page.getByRole("button", { name: "Guardar cambios" }).click();
      await expect(page.getByRole("alert").filter({ hasText: "Ingresá el nombre" })).toBeVisible();
    },
  );

  test(
    "The change is recorded in the audit log the admin sees",
    { tag: ["@high", "@e2e", "@business-data", "@DATA-E2E-003"] },
    async ({ request }) => {
      const login = await request.post(`${API}/auth/login`, { data: ADMIN });
      const { access_token: token } = (await login.json()) as { access_token: string };
      const headers = { Authorization: `Bearer ${token}` };

      const list = await request.get(`${API}/admin/businesses`, { headers });
      const { businesses } = (await list.json()) as { businesses: { id: number; name: string }[] };
      const mine = businesses.find((item) => item.name === BUSINESS.name);
      expect(mine).toBeDefined();

      const detail = await request.get(`${API}/admin/businesses/${mine?.id}`, { headers });
      const { history } = (await detail.json()) as {
        history: {
          action: string;
          actor_email: string;
          metadata: { changes: Record<string, unknown> };
        }[];
      };
      const entry = history.find((item) => item.action === "business.profile_update");
      expect(entry?.actor_email).toBe(OWNER.email);
      expect(Object.keys(entry?.metadata.changes ?? {})).toEqual(["description"]);
    },
  );
});
