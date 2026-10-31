import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

import { OWNER, signIn } from "../helpers";
import { freshClient } from "../support/client";
import { failOnConsoleErrors } from "../support/console";
import { PORTS } from "../support/env.mjs";

// Los cupones los genera el "cliente" por la API pública. El negocio puede tener una o dos
// sedes según lo que hayan hecho los specs anteriores: el test elige sede si se la piden.
test.describe.configure({ mode: "serial" });
test.use({ viewport: { width: 390, height: 844 } });

const API = `http://localhost:${PORTS.api}/api/v1`;

async function benefitId(request: APIRequestContext): Promise<{ token: string; id: number }> {
  const login = await request.post(`${API}/auth/login`, { data: OWNER });
  const { access_token: token } = (await login.json()) as { access_token: string };
  const response = await request.get(`${API}/me/business/benefits`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const { benefits } = (await response.json()) as { benefits: { id: number }[] };
  return { token, id: benefits[0].id };
}

async function generate(request: APIRequestContext, id: number, name: string, phone: string) {
  const response = await request.post(`${API}/benefits/${id}/codes`, {
    data: { name, phone },
    headers: freshClient(),
  });
  expect(response.status()).toBe(201);
  return ((await response.json()) as { code: string }).code;
}

async function open(page: Page) {
  await signIn(page, OWNER);
  await page
    .getByRole("navigation", { name: "Secciones del panel" })
    .getByRole("link", { name: "Canjear" })
    .click();
  await expect(page).toHaveURL(/\/owners\/redeem$/);
}

async function look(page: Page, code: string) {
  await page.getByLabel("Código del cupón").fill(code);
  await page.getByRole("button", { name: "Ver cupón" }).click();
}

async function confirm(page: Page) {
  await expect(page.getByRole("region", { name: "Cupón" })).toBeVisible();
  const branch = page.getByLabel("Sede donde se canjea");
  if (await branch.isVisible()) await branch.selectOption({ index: 1 });
  await page.getByRole("button", { name: "Confirmar canje" }).click();
}

test.describe("Redeem", () => {
  failOnConsoleErrors();

  test(
    "Looking at a coupon shows who it is for and what it is worth without redeeming it",
    { tag: ["@critical", "@e2e", "@redeem", "@REDEEM-E2E-001"] },
    async ({ page, request }) => {
      const { token, id } = await benefitId(request);
      // Un cliente que ya canjeó uno y genera otro: el dueño ve que es recurrente.
      const first = await generate(request, id, "Mateo Silva", "5493571000021");
      const redeemed = await request.post(`${API}/me/business/codes/redeem`, {
        headers: { Authorization: `Bearer ${token}` },
        data: { code: first, ...(await firstBranch(request, token)) },
      });
      expect(redeemed.status()).toBe(200);
      const code = await generate(request, id, "Mateo Silva", "5493571000021");

      await open(page);
      // Se escribe como lo escribe una persona: minúsculas y con espacio.
      const field = page.getByLabel("Código del cupón");
      await field.fill(`${code.slice(0, 4)} ${code.slice(4)}`.toLowerCase());
      await expect(field).toHaveValue(`${code.slice(0, 4)}-${code.slice(4)}`);
      await page.getByRole("button", { name: "Ver cupón" }).click();

      const card = page.getByRole("region", { name: "Cupón" });
      await expect(card).toContainText("Mateo Silva");
      await expect(card).toContainText("% de descuento");
      await expect(card).toContainText("Este cliente ya canjeó 1 cupón en tu negocio.");
      await expect(card).toContainText("Vence en");
      await expect(page.getByText("5493571000021")).toHaveCount(0);
      await page.screenshot({ path: "test-results/shots/redeem-preview.png", fullPage: true });

      // Mirar no canjeó nada: el cupón sigue pendiente.
      const preview = await request.get(`${API}/me/business/codes/${code}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      expect(((await preview.json()) as { status: string }).status).toBe("pending");
    },
  );

  test(
    "Confirming redeems the coupon once and a second look says it was already redeemed",
    { tag: ["@critical", "@e2e", "@redeem", "@REDEEM-E2E-002"] },
    async ({ page, request }) => {
      const { id } = await benefitId(request);
      const code = await generate(request, id, "Valeria Ruiz", "5493571000022");

      await open(page);
      await look(page, code);
      await confirm(page);
      await expect(page.getByRole("heading", { name: "Canje confirmado" })).toBeVisible();
      await expect(page.getByRole("status")).toContainText("Valeria Ruiz");
      await page.screenshot({ path: "test-results/shots/redeem-done.png" });

      await page.getByRole("button", { name: "Canjear otro cupón" }).click();
      await look(page, code);
      await expect(page.getByRole("main").getByRole("alert")).toContainText("Ya fue canjeado");
      await expect(page.getByRole("button", { name: "Confirmar canje" })).toHaveCount(0);
    },
  );

  test(
    "A code that does not exist or is badly written is explained",
    { tag: ["@high", "@e2e", "@redeem", "@REDEEM-E2E-003"] },
    async ({ page }) => {
      await open(page);

      await look(page, "ZZZZ-ZZZZ");
      await expect(page.getByRole("main").getByRole("alert")).toContainText(
        "No encontramos ese cupón",
      );

      await look(page, "AB1");
      await expect(page.getByRole("main").getByRole("alert")).toContainText("Revisá el código");
      await page.screenshot({ path: "test-results/shots/redeem-error.png" });
    },
  );

  test(
    "With several branches the last one chosen on this device is remembered",
    { tag: ["@high", "@e2e", "@redeem", "@REDEEM-E2E-004"] },
    async ({ page, request }) => {
      const { id } = await benefitId(request);
      const first = await generate(request, id, "Camila Soto", "5493571000023");

      await open(page);
      await look(page, first);
      const branch = page.getByLabel("Sede donde se canjea");
      await expect(page.getByRole("region", { name: "Cupón" })).toBeVisible();
      test.skip((await branch.count()) === 0, "El negocio tiene una sola sede activa");

      // Se elige la segunda sede y se canjea.
      const options = await branch
        .locator("option")
        .evaluateAll((all) =>
          all.map((option) => (option as HTMLOptionElement).value).filter(Boolean),
        );
      await branch.selectOption(options[1]);
      await page.getByRole("button", { name: "Confirmar canje" }).click();
      await expect(page.getByRole("heading", { name: "Canje confirmado" })).toBeVisible();

      // Otro cupón, otra visita: la sede ya viene elegida.
      const second = await generate(request, id, "Camila Soto", "5493571000023");
      await page.reload();
      await look(page, second);
      await expect(page.getByLabel("Sede donde se canjea")).toHaveValue(options[1]);
    },
  );
});

async function firstBranch(request: APIRequestContext, token: string) {
  const response = await request.get(`${API}/me/business`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const { branches } = (await response.json()) as { branches: { id: number }[] };
  return { branch_id: branches[0].id };
}
