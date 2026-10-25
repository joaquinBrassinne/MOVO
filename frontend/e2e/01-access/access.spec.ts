import { expect, test } from "@playwright/test";

import { ADMIN, OWNER, signIn } from "../helpers";
import { AccessPage } from "../pages/access-page";

test.describe("Access", () => {
  test(
    "Anonymous visitors of private views are sent to sign in",
    { tag: ["@critical", "@e2e", "@access", "@ACCESS-E2E-001"] },
    async ({ page }) => {
      for (const path of ["/socios/panel", "/socios/onboarding", "/admin/locales"]) {
        await page.goto(path);
        await expect(page).toHaveURL(/\/socios\/ingresar$/);
      }
    },
  );

  test(
    "Sign up rejects mismatched passwords, short passwords and invalid emails",
    { tag: ["@high", "@e2e", "@access", "@ACCESS-E2E-002"] },
    async ({ page }) => {
      const access = new AccessPage(page);
      await access.gotoRegister();

      await access.register("nueva@example.com", "ClaveSegura#2026", "OtraClave#2026");
      await expect(page.getByText("Las contraseñas no coinciden")).toBeVisible();

      await access.register("nueva@example.com", "corta", "corta");
      await expect(access.registerPassword).toHaveAttribute("aria-invalid", "true");

      await access.register("no-es-un-email", "ClaveSegura#2026");
      await expect(page.getByText("Ingresá un email válido")).toBeVisible();

      await expect(page).toHaveURL(/\/socios\/crear-cuenta$/);
    },
  );

  test(
    "An owner can create an account and lands on the business onboarding",
    { tag: ["@critical", "@e2e", "@access", "@ACCESS-E2E-003"] },
    async ({ page }) => {
      const access = new AccessPage(page);
      await access.gotoRegister();
      await access.register(OWNER.email, OWNER.password);

      await expect(page).toHaveURL(/\/socios\/onboarding$/);
    },
  );

  test(
    "The same email cannot be registered twice",
    { tag: ["@high", "@e2e", "@access", "@ACCESS-E2E-004"] },
    async ({ page }) => {
      const access = new AccessPage(page);
      await access.gotoRegister();
      await access.register(OWNER.email, OWNER.password);

      await expect(page.getByText("Ese email ya está registrado")).toBeVisible();
      await expect(page).toHaveURL(/\/socios\/crear-cuenta$/);
    },
  );

  test(
    "Sign in fails with a wrong password and shows no session",
    { tag: ["@critical", "@e2e", "@access", "@ACCESS-E2E-005"] },
    async ({ page }) => {
      const access = new AccessPage(page);
      await access.gotoLogin();
      await access.login(OWNER.email, "ContraseñaIncorrecta#1");

      await expect(page.getByText("Email o contraseña incorrectos")).toBeVisible();
      await expect(page).toHaveURL(/\/socios\/ingresar$/);
    },
  );

  test(
    "An owner without a business signs in and is taken to the onboarding, and the session survives a reload",
    { tag: ["@critical", "@e2e", "@access", "@ACCESS-E2E-006"] },
    async ({ page }) => {
      await signIn(page, OWNER);
      await expect(page).toHaveURL(/\/socios\/onboarding$/);

      await page.reload();
      await expect(page).toHaveURL(/\/socios\/onboarding$/);
    },
  );

  test(
    "Signed-in users do not see the access views again",
    { tag: ["@medium", "@e2e", "@access", "@ACCESS-E2E-007"] },
    async ({ page }) => {
      await signIn(page, OWNER);
      await page.goto("/socios/ingresar");
      await expect(page).toHaveURL(/\/socios\/onboarding$/);
    },
  );

  test(
    "The super admin signs in to the locales directory and owners cannot open it",
    { tag: ["@critical", "@e2e", "@access", "@ACCESS-E2E-008"] },
    async ({ page }) => {
      await signIn(page, ADMIN);
      await expect(page).toHaveURL(/\/admin\/locales$/);
      await expect(page.getByRole("heading", { name: /Gestión de locales/ })).toBeVisible();
    },
  );

  test(
    "An owner who opens the admin directory is sent back to the owner area",
    { tag: ["@critical", "@e2e", "@access", "@ACCESS-E2E-009"] },
    async ({ page }) => {
      await signIn(page, OWNER);
      await page.goto("/admin/locales");
      await expect(page).toHaveURL(/\/socios\/onboarding$/);
      await expect(page.getByRole("heading", { name: /Gestión de locales/ })).toHaveCount(0);
    },
  );

  test(
    "The browser proxy only lets the closed list of routes through",
    { tag: ["@critical", "@e2e", "@access", "@ACCESS-E2E-010"] },
    async ({ request }) => {
      // `auth/*` nunca pasa por el proxy: el refresh token solo lo ve /api/session.
      const auth = await request.post("/api/v1/auth/login", {
        data: { email: OWNER.email, password: OWNER.password },
      });
      expect(auth.status()).toBe(404);

      const unknown = await request.get("/api/v1/healthz");
      expect(unknown.status()).toBe(404);

      // Lo privado exige token; las categorías son públicas.
      const mine = await request.get("/api/v1/me/business");
      expect(mine.status()).toBe(401);

      const adminList = await request.get("/api/v1/admin/businesses");
      expect(adminList.status()).toBe(401);

      const categories = await request.get("/api/v1/categories");
      expect(categories.ok()).toBe(true);
    },
  );
});
