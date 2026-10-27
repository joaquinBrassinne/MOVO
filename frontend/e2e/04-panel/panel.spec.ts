import { expect, test, type Page } from "@playwright/test";

import { BRANCH, BUSINESS, OWNER, signIn } from "../helpers";
import { OnboardingPage } from "../pages/onboarding-page";
import { ServiceSheet } from "../pages/service-sheet";

// Sigue a 03-admin: el negocio está activo, sin piloto, con el plan Starter (2 sedes y 3
// servicios activos como máximo). El orden de los tests es el uso real del panel.
test.describe.configure({ mode: "serial" });

const serviceCard = (page: Page, name: string) =>
  page.getByRole("listitem").filter({ has: page.getByRole("heading", { name, level: 3 }) });
const branchCard = (page: Page, name: string) =>
  page.getByRole("listitem").filter({ has: page.getByRole("heading", { name, level: 3 }) });

async function addService(page: Page, name: string, day = "Sábado") {
  await page.getByRole("button", { name: "Agregar servicio" }).click();
  const sheet = new ServiceSheet(page);
  await sheet.fill({ name, days: [day], ranges: [["10:00", "11:00"]] });
  await sheet.saveAndClose();
  await expect(serviceCard(page, name)).toBeVisible();
}

test.describe("Owner panel", () => {
  test.beforeEach(async ({ page }) => {
    await signIn(page, OWNER);
  });

  test.describe("Frame", () => {
    test(
      "An approved owner lands on the panel and cannot go back to the onboarding",
      { tag: ["@critical", "@e2e", "@panel", "@PANEL-E2E-001"] },
      async ({ page }) => {
        await expect(page).toHaveURL(/\/socios\/panel$/);
        await expect(page.getByText(`Socios · ${BUSINESS.name}`)).toBeVisible();

        await page.goto("/socios/onboarding");
        await expect(page).toHaveURL(/\/socios\/panel$/);
      },
    );

    test(
      "The bottom navigation has four tabs and Mi local leads to the rest of the business",
      { tag: ["@high", "@e2e", "@panel", "@PANEL-E2E-002"] },
      async ({ page }) => {
        const nav = page.getByRole("navigation", { name: "Secciones del panel" });
        const main = page.getByRole("main");
        await expect(nav.getByRole("link")).toHaveText([
          "Inicio",
          "Canjear",
          "Beneficios",
          "Mi local",
        ]);

        await nav.getByRole("link", { name: "Mi local" }).click();
        await expect(page).toHaveURL(/\/socios\/local$/);
        await expect(page.getByRole("heading", { level: 1 })).toHaveText("Mi local");

        await main.getByRole("link", { name: /Servicios/ }).click();
        await expect(page).toHaveURL(/\/socios\/servicios$/);
        await expect(page.getByRole("heading", { level: 1 })).toHaveText("Servicios y horarios");
        // Las pantallas que cuelgan de Mi local mantienen su pestaña marcada.
        await expect(nav.getByRole("link", { name: "Mi local" })).toHaveAttribute(
          "aria-current",
          "page",
        );

        await nav.getByRole("link", { name: "Mi local" }).click();
        await main.getByRole("link", { name: /Sedes y horarios/ }).click();
        await expect(page).toHaveURL(/\/socios\/sedes$/);
        await expect(page.getByRole("heading", { level: 1 })).toHaveText("Mis sedes");

        await nav.getByRole("link", { name: "Mi local" }).click();
        await main.getByRole("link", { name: /Fotos/ }).click();
        await expect(page).toHaveURL(/\/socios\/fotos$/);

        await nav.getByRole("link", { name: "Inicio" }).click();
        await expect(page).toHaveURL(/\/socios\/panel$/);
      },
    );
  });

  test.describe("Services", () => {
    test.beforeEach(async ({ page }) => {
      await page.goto("/socios/servicios");
    });

    test(
      "Shows the plan usage and the services loaded during the onboarding",
      { tag: ["@critical", "@e2e", "@panel", "@PANEL-E2E-003"] },
      async ({ page }) => {
        await expect(page.getByText("Plan Starter activo")).toBeVisible();
        await expect(page.getByText("2 de 3 servicios activos")).toBeVisible();
        await expect(page.getByText("Te queda 1 lugar disponible en tu plan.")).toBeVisible();

        const yoga = serviceCard(page, "Yoga Vinyasa");
        await expect(yoga).toContainText("Lun, Mié y Vie · 09:00 a 10:00");
        await expect(yoga).toContainText("$ 5.500");
        await expect(yoga).toContainText("Activo");
        await expect(serviceCard(page, "Pilates Mat")).toContainText("Mar y Jue · 18:00 a 19:00");
      },
    );

    test(
      "Fills the plan and then blocks adding another service",
      { tag: ["@critical", "@e2e", "@panel", "@PANEL-E2E-004"] },
      async ({ page }) => {
        await addService(page, "Stretching Suave");

        await expect(page.getByText("3 de 3 servicios activos")).toBeVisible();
        await expect(page.getByText("Llegaste al límite de tu plan.")).toBeVisible();
        await expect(page.getByRole("button", { name: "Agregar servicio" })).toBeDisabled();
        await expect(
          page.getByText("Pausá uno de tus servicios para poder agregar otro."),
        ).toBeVisible();
      },
    );

    test(
      "Pausing a service frees a place and the plan limit also guards reactivating",
      { tag: ["@critical", "@e2e", "@panel", "@PANEL-E2E-005"] },
      async ({ page }) => {
        await page.getByRole("switch", { name: "Stretching Suave: activo" }).click();
        await expect(serviceCard(page, "Stretching Suave")).toContainText("Pausado temporalmente");
        await expect(page.getByText("2 de 3 servicios activos")).toBeVisible();
        await expect(page.getByText("2 activos · 1 pausado")).toBeVisible();
        await expect(page.getByRole("button", { name: "Agregar servicio" })).toBeEnabled();

        await addService(page, "Barre", "Domingo");
        await expect(page.getByText("3 de 3 servicios activos")).toBeVisible();

        // Con el plan lleno, volver a activar el pausado lo rechaza la API.
        await page.getByRole("switch", { name: "Stretching Suave: pausado" }).click();
        await expect(
          page.getByText("Llegaste al límite de tu plan", { exact: true }),
        ).toBeVisible();
        await expect(
          page.getByText("Tu plan no tiene lugar para otro servicio activo."),
        ).toBeVisible();
        await expect(serviceCard(page, "Stretching Suave")).toContainText("Pausado temporalmente");
      },
    );

    test(
      "Edits a service: name, price and schedule",
      { tag: ["@high", "@e2e", "@panel", "@PANEL-E2E-006"] },
      async ({ page }) => {
        await page.getByRole("button", { name: "Editar Pilates Mat" }).click();
        const sheet = new ServiceSheet(page);
        await expect(sheet.name).toHaveValue("Pilates Mat");

        await sheet.name.fill("Pilates Reformer");
        await sheet.price.fill("$ 12.000");
        await sheet.dialog
          .getByRole("button", { name: "Quitar la franja de 18:00 a 19:00" })
          .click();
        await sheet.dialog.getByRole("button", { name: "Agregar franja" }).click();
        await sheet.dialog.getByLabel("Desde").fill("19:00");
        await sheet.dialog.getByLabel("Hasta").fill("20:30");
        await sheet.dialog.getByRole("button", { name: "Sumar" }).click();
        await sheet.saveAndClose();

        const card = serviceCard(page, "Pilates Reformer");
        await expect(card).toContainText("Mar y Jue · 19:00 a 20:30");
        await expect(card).toContainText("$ 12.000");
        await expect(page.getByText("Pilates Mat")).toHaveCount(0);
      },
    );

    test(
      "Rejects a service without a name, without days and with a bad price",
      { tag: ["@high", "@e2e", "@panel", "@PANEL-E2E-007"] },
      async ({ page }) => {
        await page.getByRole("switch", { name: "Barre: activo" }).click();
        await page.getByRole("button", { name: "Agregar servicio" }).click();
        const sheet = new ServiceSheet(page);

        await sheet.price.fill("mucho");
        // Una franja sin ningún día elegido no se puede guardar.
        await sheet.dialog.getByRole("button", { name: "Agregar franja" }).click();
        await sheet.dialog.getByLabel("Desde").fill("10:00");
        await sheet.dialog.getByLabel("Hasta").fill("11:00");
        await sheet.dialog.getByRole("button", { name: "Sumar" }).click();
        await sheet.save.click();

        await expect(sheet.dialog.getByText("Ingresá el nombre del servicio")).toBeVisible();
        await expect(sheet.dialog.getByText("Ingresá un precio válido en pesos")).toBeVisible();
        await expect(sheet.dialog.getByText("Elegí al menos un día")).toBeVisible();

        await sheet.dialog.getByRole("button", { name: "Cerrar" }).click();
        await expect(sheet.dialog).toBeHidden();
      },
    );

    test(
      "Deletes a service from its sheet after confirming",
      { tag: ["@high", "@e2e", "@panel", "@PANEL-E2E-008"] },
      async ({ page }) => {
        await page.getByRole("button", { name: "Editar Barre" }).click();
        const sheet = new ServiceSheet(page);

        page.once("dialog", (dialog) => void dialog.accept());
        await sheet.dialog.getByRole("button", { name: "Eliminar servicio" }).click();

        await expect(serviceCard(page, "Barre")).toHaveCount(0);
        await expect(sheet.dialog).toBeHidden();
      },
    );
  });

  test.describe("Branches", () => {
    test.beforeEach(async ({ page }) => {
      await page.goto("/socios/sedes");
    });

    test(
      "Lists the branch created during the onboarding with the plan usage",
      { tag: ["@critical", "@e2e", "@panel", "@PANEL-E2E-009"] },
      async ({ page }) => {
        await expect(page.getByText("Plan Starter activo")).toBeVisible();
        await expect(page.getByText("1 de 2 sedes activas")).toBeVisible();

        const card = branchCard(page, "Sede principal");
        await expect(card).toContainText(`${BRANCH.address}, ${BRANCH.city}`);
        await expect(card).toContainText(BRANCH.reference);
        await expect(card).toContainText("Activa");
        await expect(page.getByRole("link", { name: "Agregar sede" })).toBeVisible();
      },
    );

    test(
      "Asks for every required field of a new branch",
      { tag: ["@high", "@e2e", "@panel", "@PANEL-E2E-010"] },
      async ({ page }) => {
        await page.getByRole("link", { name: "Agregar sede" }).click();
        await expect(page).toHaveURL(/\/socios\/sedes\/nueva$/);

        await page.getByRole("button", { name: "Guardar sede" }).click();

        await expect(page.getByText("Ingresá el nombre de la sede")).toBeVisible();
        await expect(page.getByText("Ingresá la dirección")).toBeVisible();
        await expect(page.getByText("Ingresá la ciudad")).toBeVisible();
        await expect(page.getByText("Marcá la ubicación en el mapa")).toBeVisible();
        await expect(page).toHaveURL(/\/socios\/sedes\/nueva$/);
      },
    );

    test(
      "Rejects opening hours where closing is not after opening",
      { tag: ["@high", "@e2e", "@panel", "@PANEL-E2E-011"] },
      async ({ page }) => {
        await page.goto("/socios/sedes/nueva");
        const form = new OnboardingPage(page);
        await page.getByLabel("Nombre de la sede").fill("Sede Centro");
        await form.fillAddress({ address: "Belgrano 250", city: BRANCH.city });
        await form.searchAndPin();

        await page.getByRole("switch", { name: "Lunes: cerrado" }).click();
        await page.getByLabel("Lunes: abre").fill("20:00");
        await page.getByLabel("Lunes: cierra").fill("08:00");
        await page.getByRole("button", { name: "Guardar sede" }).click();

        await expect(page.getByText("Horarios incorrectos")).toBeVisible();
        await expect(page).toHaveURL(/\/socios\/sedes\/nueva$/);
      },
    );

    test(
      "Creates a branch with opening hours, copying Monday to Friday",
      { tag: ["@critical", "@e2e", "@panel", "@PANEL-E2E-012"] },
      async ({ page }) => {
        await page.goto("/socios/sedes/nueva");
        const form = new OnboardingPage(page);
        await page.getByLabel("Nombre de la sede").fill("Sede Centro");
        await form.fillAddress({ address: "Belgrano 250", city: BRANCH.city });
        await form.searchAndPin();

        await page.getByRole("switch", { name: "Lunes: cerrado" }).click();
        await page.getByLabel("Lunes: abre").fill("08:00");
        await page.getByLabel("Lunes: cierra").fill("20:00");
        await page.getByRole("button", { name: "Copiar Lunes a otros días" }).click();
        for (const name of ["Martes", "Miércoles", "Jueves", "Viernes"]) {
          await page.getByRole("checkbox", { name }).check();
        }
        await page.getByRole("button", { name: "Aplicar" }).click();
        await expect(page.getByRole("switch", { name: "Viernes: abierto" })).toBeVisible();
        await expect(page.getByLabel("Viernes: cierra")).toHaveValue("20:00");
        await expect(page.getByRole("switch", { name: "Sábado: cerrado" })).toBeVisible();

        await page.getByRole("button", { name: "Guardar sede" }).click();

        await expect(page).toHaveURL(/\/socios\/sedes$/);
        await expect(branchCard(page, "Sede Centro")).toContainText("Belgrano 250, Río Tercero");
        await expect(page.getByText("2 de 2 sedes activas")).toBeVisible();
        await expect(page.getByText("Llegaste al límite de sedes de tu plan")).toBeVisible();
        await expect(page.getByRole("link", { name: "Agregar sede" })).toHaveCount(0);
      },
    );

    test(
      "Edits a branch and keeps its saved hours",
      { tag: ["@high", "@e2e", "@panel", "@PANEL-E2E-013"] },
      async ({ page }) => {
        await page.getByRole("link", { name: "Editar Sede Centro" }).click();
        await expect(page.getByRole("heading", { name: "Sede Centro", level: 2 })).toBeVisible();

        await expect(page.getByLabel("Nombre de la sede")).toHaveValue("Sede Centro");
        await expect(page.getByLabel("Lunes: abre")).toHaveValue("08:00");
        await expect(page.getByLabel("Miércoles: cierra")).toHaveValue("20:00");
        await expect(page.getByRole("switch", { name: "Sábado: cerrado" })).toBeVisible();
        await expect(page.getByRole("switch", { name: "Domingo: cerrado" })).toBeVisible();

        await page.getByLabel("Nombre de la sede").fill("Sede Centro Norte");
        await page.getByLabel("Lunes: cierra").fill("21:00");
        await page.getByRole("button", { name: "Guardar sede" }).click();

        await expect(page).toHaveURL(/\/socios\/sedes$/);
        await expect(branchCard(page, "Sede Centro Norte")).toBeVisible();

        await page.getByRole("link", { name: "Editar Sede Centro Norte" }).click();
        await expect(page.getByLabel("Lunes: cierra")).toHaveValue("21:00");
      },
    );

    test(
      "Hides a branch and the plan limit also guards showing it again",
      { tag: ["@critical", "@e2e", "@panel", "@PANEL-E2E-014"] },
      async ({ page }) => {
        const hide = page.getByRole("switch", {
          name: "Sede Centro Norte: visible al público",
        });
        await hide.click();
        await expect(branchCard(page, "Sede Centro Norte")).toContainText("Oculta");
        await expect(page.getByText("1 de 2 sedes activas")).toBeVisible();

        // Con un lugar libre se puede crear otra sede...
        await page.getByRole("link", { name: "Agregar sede" }).click();
        const form = new OnboardingPage(page);
        await page.getByLabel("Nombre de la sede").fill("Sede Extra");
        await form.fillAddress({ address: "Sarmiento 90", city: BRANCH.city });
        await form.searchAndPin();
        await page.getByRole("button", { name: "Guardar sede" }).click();
        await expect(page).toHaveURL(/\/socios\/sedes$/);
        await expect(page.getByText("2 de 2 sedes activas")).toBeVisible();

        // ...y con el plan lleno la API rechaza volver a mostrar la oculta.
        await page.getByRole("switch", { name: "Sede Centro Norte: oculta" }).click();
        await expect(page.getByText("Tu plan no tiene lugar para otra sede.")).toBeVisible();
        await expect(branchCard(page, "Sede Centro Norte")).toContainText("Oculta");
      },
    );

    test(
      "Says so when the branch does not exist",
      { tag: ["@medium", "@e2e", "@panel", "@PANEL-E2E-015"] },
      async ({ page }) => {
        await page.goto("/socios/sedes/99999");

        await expect(page.getByText("No encontramos esa sede")).toBeVisible();
        await page.getByRole("link", { name: "Volver a Mis sedes" }).click();
        await expect(page).toHaveURL(/\/socios\/sedes$/);
      },
    );
  });

  test.describe("Session", () => {
    test(
      "Signing out ends the session for good",
      { tag: ["@critical", "@e2e", "@panel", "@PANEL-E2E-016"] },
      async ({ page }) => {
        await page.getByRole("button", { name: "Cuenta" }).click();
        await expect(page.getByRole("banner").getByText(OWNER.email)).toBeVisible();
        await page.getByRole("banner").getByRole("button", { name: "Cerrar sesión" }).click();

        await expect(page).toHaveURL(/\/socios\/ingresar$/);

        await page.goto("/socios/panel");
        await expect(page).toHaveURL(/\/socios\/ingresar$/);

        // La cookie de refresco se borró: recargar no devuelve la sesión.
        await page.reload();
        await expect(page).toHaveURL(/\/socios\/ingresar$/);
      },
    );
  });
});
