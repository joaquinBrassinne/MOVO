import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

import { BUSINESS, OWNER } from "../helpers";
import { PORTS } from "../support/env.mjs";

// El cliente no tiene cuenta: entra a la ficha pública, genera su cupón y lo guarda en el
// navegador. El negocio ya está activo y tiene beneficios (benefits.spec).
test.describe.configure({ mode: "serial" });
test.use({
  viewport: { width: 390, height: 844 },
  permissions: ["clipboard-read", "clipboard-write"],
});

const API = `http://localhost:${PORTS.api}/api/v1`;
const PROFILE = "/locales/estudio-aire-libre";

async function ownerToken(request: APIRequestContext): Promise<string> {
  const login = await request.post(`${API}/auth/login`, { data: OWNER });
  return ((await login.json()) as { access_token: string }).access_token;
}

async function openSheet(page: Page, benefit = 0) {
  await page.goto(PROFILE);
  await page.getByRole("button", { name: "Obtener cupón" }).nth(benefit).click();
  return page.getByRole("dialog", { name: "Obtené tu cupón" });
}

async function generate(sheet: ReturnType<Page["getByRole"]>, name: string, phone: string) {
  await sheet.getByLabel("Tu nombre").fill(name);
  await sheet.getByLabel("Tu WhatsApp").fill(phone);
  await sheet.getByRole("button", { name: "Generar cupón" }).click();
}

test.describe("Customer coupons", () => {
  test(
    "The profile lists the benefits on offer, best first, with a button to get each coupon",
    { tag: ["@critical", "@e2e", "@coupons", "@COUPONS-E2E-001"] },
    async ({ page }) => {
      await page.goto(PROFILE);

      const section = page.getByRole("region", { name: "Beneficios" });
      await expect(section).toBeVisible();
      const cards = section.getByRole("listitem");
      await expect(cards).toHaveCount(3);
      // Primero el porcentaje (37 %), después el monto y por último el beneficio propio.
      await expect(cards.nth(0)).toContainText("-37%");
      await expect(cards.nth(1)).toContainText("-$2.000");
      await expect(cards.nth(2)).toContainText("Una clase de prueba gratis");
      await page.screenshot({ path: "test-results/shots/profile-benefits.png", fullPage: true });
    },
  );

  test(
    "Getting a coupon needs a name and a WhatsApp, and ends on the coupon with its code and QR",
    { tag: ["@critical", "@e2e", "@coupons", "@COUPONS-E2E-002"] },
    async ({ page }) => {
      const sheet = await openSheet(page);
      await sheet.getByRole("button", { name: "Generar cupón" }).click();
      await expect(sheet.getByText("Ingresá tu nombre")).toBeVisible();
      await expect(sheet.getByText("Ingresá tu número con característica")).toBeVisible();
      await page.screenshot({ path: "test-results/shots/get-coupon.png" });

      await generate(sheet, "Camila Gómez", "11 5555 0001");
      await expect(page).toHaveURL(/\/cupon\/[A-Za-z0-9_-]{43}\?nuevo=1$/);
      await expect(page.getByRole("heading", { name: "¡Cupón generado!" })).toBeVisible();

      const qr = page.getByRole("img", { name: /^Código QR del cupón / });
      await expect(qr).toBeVisible();
      await expect(page.getByText(/^Vence en \d\d:\d\d hs$/)).toBeVisible();
      await expect(page.getByText("5491155550001")).toHaveCount(0);
      await page.screenshot({ path: "test-results/shots/coupon-generated.png", fullPage: true });

      await page.getByRole("button", { name: "Guardar enlace del cupón" }).click();
      await expect(page.getByRole("status")).toContainText("Enlace copiado");
      const copied = await page.evaluate(() => navigator.clipboard.readText());
      expect(copied).toMatch(/\/cupon\/[A-Za-z0-9_-]{43}$/);
    },
  );

  test(
    "Mis cupones shows the coupons saved on this device and opens each with its link",
    { tag: ["@critical", "@e2e", "@coupons", "@COUPONS-E2E-003"] },
    async ({ page }) => {
      // Un navegador limpio no tiene nada guardado.
      await page.goto("/cupones");
      await expect(page.getByText("Todavía no tenés cupones guardados")).toBeVisible();

      // Se genera uno y se vuelve a la lista.
      const sheet = await openSheet(page, 1);
      await generate(sheet, "Lucía Torres", "11 5555 0002");
      await expect(page.getByRole("heading", { name: "¡Cupón generado!" })).toBeVisible();
      await page.goto("/cupones");

      const cards = page.getByRole("list", { name: "Cupones" }).getByRole("listitem");
      await expect(cards).toHaveCount(1);
      await expect(cards.first()).toContainText(BUSINESS.name);
      await expect(cards.first()).toContainText("-$2.000");
      await expect(cards.first()).toContainText(/Vence en \d+ hs/);
      await expect(page.getByRole("tab", { name: /Activos \(1\)/ })).toBeVisible();
      await expect(page.getByRole("tab", { name: /Historial \(0\)/ })).toBeVisible();
      await page.screenshot({ path: "test-results/shots/my-coupons.png", fullPage: true });

      await cards
        .first()
        .getByRole("link", { name: /Abrir el cupón/ })
        .click();
      await expect(page).toHaveURL(/\/cupon\/[A-Za-z0-9_-]{43}$/);
      await expect(page.getByRole("heading", { name: "Tu cupón" })).toBeVisible();
      await expect(page.getByRole("img", { name: /^Código QR del cupón / })).toBeVisible();
    },
  );

  test(
    "A second coupon at the same business is refused and points to the active one",
    { tag: ["@high", "@e2e", "@coupons", "@COUPONS-E2E-004"] },
    async ({ page }) => {
      const first = await openSheet(page);
      await generate(first, "Sofía Peña", "11 5555 0003");
      await expect(page.getByRole("heading", { name: "¡Cupón generado!" })).toBeVisible();

      const second = await openSheet(page, 1);
      await generate(second, "Sofía Peña", "11 5555 0003");
      const alert = second.getByRole("alert");
      await expect(alert).toContainText("Ya tenés un cupón activo en este local");
      await alert.getByRole("link", { name: "Ver mi cupón activo" }).click();
      await expect(page).toHaveURL(/\/cupones$/);
      await expect(page.getByRole("list", { name: "Cupones" }).getByRole("listitem")).toHaveCount(
        1,
      );
    },
  );

  test(
    "Once the business redeems the coupon, it moves to the history of the customer",
    { tag: ["@high", "@e2e", "@coupons", "@COUPONS-E2E-005"] },
    async ({ page, request }) => {
      const sheet = await openSheet(page);
      await generate(sheet, "Julián Paz", "11 5555 0004");
      await expect(page.getByRole("heading", { name: "¡Cupón generado!" })).toBeVisible();
      const code = (await page.getByLabel(/^Código del cupón /).getAttribute("aria-label"))
        ?.replace("Código del cupón ", "")
        .replace("-", "");

      const token = await ownerToken(request);
      const business = await request.get(`${API}/me/business`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const { branches } = (await business.json()) as { branches: { id: number }[] };
      const redeem = await request.post(`${API}/me/business/codes/redeem`, {
        headers: { Authorization: `Bearer ${token}` },
        data: { code, branch_id: branches[0].id },
      });
      expect(redeem.status()).toBe(200);

      await page.goto("/cupones");
      await expect(page.getByRole("tab", { name: /Activos \(0\)/ })).toBeVisible();
      await page.getByRole("tab", { name: /Historial \(1\)/ }).click();
      await expect(page.getByRole("list", { name: "Cupones" }).getByRole("listitem")).toContainText(
        "Canjeado el",
      );
      await expect(page.getByRole("list", { name: "Cupones" }).getByRole("listitem")).toContainText(
        "Cupón ya utilizado",
      );
    },
  );

  test(
    "A link that is not a coupon says so, and a business at its monthly limit says when it reopens",
    { tag: ["@high", "@e2e", "@coupons", "@COUPONS-E2E-006"] },
    async ({ page, request }) => {
      await page.goto(`/cupon/${"x".repeat(43)}`);
      await expect(page.getByRole("main").getByRole("alert")).toContainText(
        "No encontramos este cupón",
      );

      // El cupo del local se agota: el límite pasa a ser lo ya generado este mes.
      const token = await ownerToken(request);
      const headers = { Authorization: `Bearer ${token}` };
      const quota = await request.get(`${API}/me/business/coupon-quota`, { headers });
      const { generated_this_month: generated } = (await quota.json()) as {
        generated_this_month: number;
      };
      const limit = await request.put(`${API}/me/business/coupon-quota`, {
        headers,
        data: { limit: generated },
      });
      expect(limit.status()).toBe(200);

      try {
        const sheet = await openSheet(page);
        await generate(sheet, "Tomás Ibarra", "11 5555 0005");
        await expect(sheet.getByRole("alert")).toContainText("Cupos mensuales alcanzados");
        await expect(sheet.getByRole("alert")).toContainText(
          /Nuevos lugares disponibles el \d+ de/,
        );
        await page.screenshot({ path: "test-results/shots/quota-reached.png" });
      } finally {
        await request.put(`${API}/me/business/coupon-quota`, { headers, data: { limit: null } });
      }
    },
  );
});
