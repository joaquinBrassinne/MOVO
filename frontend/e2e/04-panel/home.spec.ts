import { expect, test, type APIRequestContext } from "@playwright/test";

import { BUSINESS, OWNER, signIn } from "../helpers";
import { failOnConsoleErrors } from "../support/console";
import { PORTS } from "../support/env.mjs";

// Sigue a benefits.spec: el negocio ya tiene beneficios y no tiene cupo. Los cupones los
// genera el "cliente" por la API pública, como lo haría la web de clientes.
test.describe.configure({ mode: "serial" });
test.use({ viewport: { width: 390, height: 844 } });

const API = `http://localhost:${PORTS.api}/api/v1`;

interface Owner {
  token: string;
  benefitId: number;
  branchId: number;
}

async function loadOwner(request: APIRequestContext): Promise<Owner> {
  const login = await request.post(`${API}/auth/login`, { data: OWNER });
  const { access_token: token } = (await login.json()) as { access_token: string };
  const headers = { Authorization: `Bearer ${token}` };

  const benefits = await request.get(`${API}/me/business/benefits`, { headers });
  const { benefits: list } = (await benefits.json()) as { benefits: { id: number }[] };
  const business = await request.get(`${API}/me/business`, { headers });
  const { branches } = (await business.json()) as { branches: { id: number }[] };
  return { token, benefitId: list[0].id, branchId: branches[0].id };
}

async function generate(request: APIRequestContext, owner: Owner, name: string, phone: string) {
  const response = await request.post(`${API}/benefits/${owner.benefitId}/codes`, {
    data: { name, phone },
  });
  expect(response.status()).toBe(201);
  return ((await response.json()) as { code: string }).code;
}

async function redeem(request: APIRequestContext, owner: Owner, code: string) {
  const response = await request.post(`${API}/me/business/codes/redeem`, {
    headers: { Authorization: `Bearer ${owner.token}` },
    data: { code, branch_id: owner.branchId },
  });
  expect(response.status()).toBe(200);
}

test.describe("Owner home", () => {
  failOnConsoleErrors();

  test(
    "The home shows the redeem button, the counts, the quota and an empty activity",
    { tag: ["@critical", "@e2e", "@home", "@HOME-E2E-001"] },
    async ({ page }) => {
      await signIn(page, OWNER);

      await expect(page.getByRole("heading", { name: `Hola, ${BUSINESS.name}` })).toBeVisible();
      await expect(page.getByRole("link", { name: "Canjear cupón" })).toHaveAttribute(
        "href",
        "/socios/canjear",
      );
      for (const label of [
        "Cupones activos",
        "Canjes de hoy",
        "Vistas de tu perfil",
        "Clics a WhatsApp",
      ]) {
        await expect(page.getByText(label, { exact: true })).toBeVisible();
      }
      await expect(page.getByRole("region", { name: "Cupo del mes" })).toBeVisible();
      await expect(page.getByText("Todavía no hay actividad")).toBeVisible();
      await page.screenshot({ path: "test-results/shots/home-empty.png", fullPage: true });
    },
  );

  test(
    "Generated and redeemed coupons reach the activity, the counts and the bell",
    { tag: ["@critical", "@e2e", "@home", "@HOME-E2E-002"] },
    async ({ page, request }) => {
      await signIn(page, OWNER);
      await expect(page.getByText("Todavía no hay actividad")).toBeVisible();

      const owner = await loadOwner(request);
      const first = await generate(request, owner, "Lucía Martínez", "5493571000011");
      await generate(request, owner, "Tomás Rojas", "5493571000012");
      await redeem(request, owner, first);

      await page.reload();
      const activity = page.getByRole("region", { name: "Actividad reciente" });
      await expect(activity).toContainText("Lucía M. canjeó su cupón de");
      await expect(activity).toContainText("Tomás R. generó un cupón de");
      await expect(activity).toContainText("Lucía M. generó un cupón de");

      const card = (label: string) => page.getByText(label, { exact: true }).locator("xpath=..");
      await expect(card("Cupones activos")).toContainText("1");
      await expect(card("Canjes de hoy")).toContainText("1");
      // Nada de teléfonos en la pantalla.
      await expect(page.getByText("5493571000011")).toHaveCount(0);

      // La campana cuenta lo posterior a la primera visita y se apaga al mirarla.
      const bell = page.getByRole("button", { name: /^Novedades/ });
      await expect(bell).toHaveAccessibleName("Novedades: 3");
      await page.screenshot({ path: "test-results/shots/home-activity.png", fullPage: true });
      await bell.click();
      await expect(bell).toHaveAccessibleName("Novedades");
    },
  );
});
