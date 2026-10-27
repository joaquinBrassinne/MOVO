import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

import { OWNER, signIn } from "../helpers";
import { PORTS } from "../support/env.mjs";
import { writeQrVideo } from "../support/fake-camera.mjs";

// La cámara es un video con un QR que cada test escribe antes de abrir el escáner (ver
// support/fake-camera.mjs). Los cupones los genera el "cliente" por la API pública.
test.describe.configure({ mode: "serial" });
test.use({ viewport: { width: 390, height: 844 } });

const API = `http://localhost:${PORTS.api}/api/v1`;
const ORIGIN = `http://localhost:${PORTS.web}`;

async function newCoupon(request: APIRequestContext, name: string, phone: string) {
  const login = await request.post(`${API}/auth/login`, { data: OWNER });
  const { access_token: token } = (await login.json()) as { access_token: string };
  const benefits = await request.get(`${API}/me/business/benefits`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const { benefits: list } = (await benefits.json()) as { benefits: { id: number }[] };
  const response = await request.post(`${API}/benefits/${list[0].id}/codes`, {
    data: { name, phone },
  });
  expect(response.status()).toBe(201);
  return { token, code: ((await response.json()) as { code: string }).code };
}

async function openRedeem(page: Page) {
  await signIn(page, OWNER);
  await page
    .getByRole("navigation", { name: "Secciones del panel" })
    .getByRole("link", { name: "Canjear" })
    .click();
  await expect(page).toHaveURL(/\/socios\/canjear$/);
}

test.describe("QR scanner", () => {
  test(
    "Scanning the QR of a coupon shows its preview, redeems nothing and loads only from our origin",
    { tag: ["@critical", "@e2e", "@scanner", "@SCANNER-E2E-001"] },
    async ({ page, request }) => {
      const { token, code } = await newCoupon(request, "Sofía Ledesma", "5493571000031");
      writeQrVideo(code);

      const requested: string[] = [];
      page.on("request", (item) => requested.push(item.url()));
      await openRedeem(page);
      // Antes de tocar el botón no se baja ni la librería ni el WASM.
      expect(requested.some((url) => url.includes("zxing"))).toBe(false);

      await page.getByRole("button", { name: "Escanear QR" }).click();
      await expect(page.getByRole("region", { name: "Escáner de QR" })).toBeVisible();
      await page.screenshot({ path: "test-results/shots/scanner-open.png" });

      const card = page.getByRole("region", { name: "Cupón" });
      await expect(card).toContainText("Sofía Ledesma", { timeout: 30_000 });
      await expect(page.getByRole("region", { name: "Escáner de QR" })).toHaveCount(0);
      await expect(page.getByLabel("Código del cupón")).toHaveValue(
        `${code.slice(0, 4)}-${code.slice(4)}`,
      );
      await page.screenshot({ path: "test-results/shots/scanner-found.png", fullPage: true });

      // El WASM salió de nuestro origen y nada se pidió a un servidor externo.
      expect(requested.some((url) => url === `${ORIGIN}/zxing/zxing_reader.wasm`)).toBe(true);
      const external = requested.filter(
        (url) => !url.startsWith(ORIGIN) && !/^(data|blob):/.test(url),
      );
      expect(external).toEqual([]);

      // Leer el QR no canjeó nada: el cupón sigue pendiente.
      const preview = await request.get(`${API}/me/business/codes/${code}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      expect(((await preview.json()) as { status: string }).status).toBe("pending");
    },
  );

  test(
    "A QR that is not a MOVO coupon is ignored with a hint and manual entry stays available",
    { tag: ["@high", "@e2e", "@scanner", "@SCANNER-E2E-002"] },
    async ({ page }) => {
      writeQrVideo("https://example.com/promo");
      await openRedeem(page);

      await page.getByRole("button", { name: "Escanear QR" }).click();
      await expect(
        page.getByRole("status").filter({ hasText: "no es de un cupón de MOVO" }),
      ).toBeVisible({
        timeout: 30_000,
      });
      await expect(page.getByLabel("Código del cupón")).toBeEnabled();

      await page.getByRole("button", { name: "Cancelar" }).click();
      await expect(page.getByRole("region", { name: "Escáner de QR" })).toHaveCount(0);
    },
  );

  test(
    "Without permission or without a camera the scanner says so and manual entry still works",
    { tag: ["@critical", "@e2e", "@scanner", "@SCANNER-E2E-003"] },
    async ({ page, request }) => {
      const { code } = await newCoupon(request, "Julián Paz", "5493571000032");
      await page.addInitScript(() => {
        navigator.mediaDevices.getUserMedia = () =>
          Promise.reject(new DOMException("denied", "NotAllowedError"));
      });
      await openRedeem(page);

      await page.getByRole("button", { name: "Escanear QR" }).click();
      const scanner = page.getByRole("region", { name: "Escáner de QR" });
      await expect(scanner).toContainText("No tenemos permiso para usar la cámara");
      await expect(scanner).toContainText("ingresar el código a mano");
      await page.screenshot({ path: "test-results/shots/scanner-denied.png", fullPage: true });

      // El ingreso manual sigue funcionando.
      await page.getByLabel("Código del cupón").fill(code);
      await page.getByRole("button", { name: "Ver cupón" }).click();
      await expect(page.getByRole("region", { name: "Cupón" })).toContainText("Julián Paz");
    },
  );

  test(
    "A browser with no camera support gets the same notice",
    { tag: ["@high", "@e2e", "@scanner", "@SCANNER-E2E-004"] },
    async ({ page }) => {
      await page.addInitScript(() => {
        Object.defineProperty(navigator, "mediaDevices", { value: undefined });
      });
      await openRedeem(page);

      await page.getByRole("button", { name: "Escanear QR" }).click();
      await expect(page.getByRole("region", { name: "Escáner de QR" })).toContainText(
        "No encontramos una cámara",
      );
      await expect(page.getByLabel("Código del cupón")).toBeEnabled();
    },
  );
});
