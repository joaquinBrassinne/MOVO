import { expect, test } from "@playwright/test";

import { BRANCH, BUSINESS, OWNER, photo, signIn } from "../helpers";
import { OnboardingPage } from "../pages/onboarding-page";
import { ServiceSheet } from "../pages/service-sheet";

// Cada test ingresa de nuevo: el alta retoma donde el dueño quedó según lo guardado en la
// API, así que el orden de los tests es el avance real del alta (ver playwright.config.ts).
test.describe.configure({ mode: "serial" });

test.describe("Onboarding", () => {
  test.beforeEach(async ({ page }) => {
    await signIn(page, OWNER);
  });

  test.describe("Step 1: business", () => {
    test(
      "Asks for the required data before moving on",
      { tag: ["@critical", "@e2e", "@onboarding", "@ONBOARD-E2E-001"] },
      async ({ page }) => {
        const onboarding = new OnboardingPage(page);
        await onboarding.expectStep(1);

        await onboarding.next.click();

        await expect(page.getByText("Ingresá el nombre de tu negocio")).toBeVisible();
        await expect(page.getByText("Elegí una categoría")).toBeVisible();
        await expect(
          page.getByText("Ingresá tu número con característica, sin 0 ni 15"),
        ).toBeVisible();
        await onboarding.expectStep(1);
      },
    );

    test(
      "Offers every category and counts the description",
      { tag: ["@medium", "@e2e", "@onboarding", "@ONBOARD-E2E-002"] },
      async ({ page }) => {
        const onboarding = new OnboardingPage(page);

        const radios = page.getByRole("radio");
        await expect(radios).toHaveCount(7);
        await expect(page.getByRole("radio", { name: "Pádel" })).toBeVisible();

        await onboarding.businessDescription.fill("hola");
        await expect(page.getByText("4/500")).toBeVisible();
      },
    );

    test(
      "Rejects a WhatsApp number that is not valid",
      { tag: ["@high", "@e2e", "@onboarding", "@ONBOARD-E2E-003"] },
      async ({ page }) => {
        const onboarding = new OnboardingPage(page);
        await onboarding.fillBusiness({ ...BUSINESS, phone: "123" });
        await onboarding.next.click();

        await expect(
          page.getByText("Ingresá tu número con característica, sin 0 ni 15"),
        ).toBeVisible();
        await onboarding.expectStep(1);
      },
    );

    test(
      "Saves the business and moves on to the branch",
      { tag: ["@critical", "@e2e", "@onboarding", "@ONBOARD-E2E-004"] },
      async ({ page }) => {
        const onboarding = new OnboardingPage(page);
        await onboarding.fillBusiness(BUSINESS);
        await onboarding.next.click();

        await onboarding.expectStep(2);
      },
    );
  });

  test.describe("Step 2: branch", () => {
    test(
      "Resumes at the branch and keeps the business data when going back",
      { tag: ["@high", "@e2e", "@onboarding", "@ONBOARD-E2E-005"] },
      async ({ page }) => {
        const onboarding = new OnboardingPage(page);
        await onboarding.expectStep(2);

        await onboarding.back.click();

        await onboarding.expectStep(1);
        await expect(onboarding.businessName).toHaveValue(BUSINESS.name);
        await expect(onboarding.businessPhone).toHaveValue(BUSINESS.phone);
        await expect(page.getByRole("radio", { name: BUSINESS.category })).toHaveAttribute(
          "aria-checked",
          "true",
        );
      },
    );

    test(
      "Asks for the address and the pin before moving on",
      { tag: ["@critical", "@e2e", "@onboarding", "@ONBOARD-E2E-006"] },
      async ({ page }) => {
        const onboarding = new OnboardingPage(page);
        await onboarding.next.click();

        await expect(page.getByText("Ingresá la dirección")).toBeVisible();
        await expect(page.getByText("Ingresá la ciudad")).toBeVisible();
        await expect(page.getByText("Marcá la ubicación en el mapa")).toBeVisible();
        await onboarding.expectStep(2);
      },
    );

    test(
      "Reports an address the geocoder does not find",
      { tag: ["@high", "@e2e", "@onboarding", "@ONBOARD-E2E-007"] },
      async ({ page }) => {
        const onboarding = new OnboardingPage(page);
        await onboarding.fillAddress({ address: "Calle Inexistente 123", city: BRANCH.city });
        await onboarding.searchOnMap.click();

        await expect(
          page.getByText("No encontramos esa dirección. Revisala o ubicá el pin a mano."),
        ).toBeVisible();
        await expect(onboarding.coordinates).toHaveCount(0);
      },
    );

    test(
      "Searching the address puts the pin on the map",
      { tag: ["@critical", "@e2e", "@onboarding", "@ONBOARD-E2E-008"] },
      async ({ page }) => {
        const onboarding = new OnboardingPage(page);
        await onboarding.fillAddress(BRANCH);
        await onboarding.searchAndPin();

        await expect(onboarding.coordinates).toHaveText("Lat -32.1925 · Lng -64.4198");
        await expect(page.locator(".leaflet-marker-icon")).toBeVisible();
        await expect(page.getByRole("button", { name: "Recentrar" })).toBeEnabled();
      },
    );

    test(
      "Tapping the map moves the pin",
      { tag: ["@high", "@e2e", "@onboarding", "@ONBOARD-E2E-009"] },
      async ({ page }) => {
        const onboarding = new OnboardingPage(page);
        await onboarding.fillAddress(BRANCH);
        await onboarding.searchAndPin();
        const before = await onboarding.coordinates.textContent();

        await onboarding.tapMap(60, 80);

        await expect(onboarding.coordinates).not.toHaveText(before ?? "");
      },
    );

    test(
      "The map never covers the fixed footer buttons",
      { tag: ["@high", "@e2e", "@onboarding", "@ONBOARD-E2E-010"] },
      async ({ page }) => {
        const onboarding = new OnboardingPage(page);
        await onboarding.fillAddress(BRANCH);
        await onboarding.searchAndPin();

        // Deja el mapa parcialmente detrás de la barra inferior, como en el bug original.
        const box = await onboarding.map.boundingBox();
        const viewport = page.viewportSize();
        if (!box || !viewport) throw new Error("the map is not on screen");
        await page.evaluate(
          (delta) => window.scrollBy(0, delta),
          box.y + box.height - (viewport.height - 30),
        );

        for (const button of [onboarding.next, onboarding.back]) {
          const reachable = await button.evaluate((element) => {
            const rect = element.getBoundingClientRect();
            const top = document.elementFromPoint(
              rect.x + rect.width / 2,
              rect.y + rect.height / 2,
            );
            return top !== null && element.contains(top);
          });
          expect(reachable).toBe(true);
        }
      },
    );

    test(
      "Rejects a booking link without a scheme",
      { tag: ["@high", "@e2e", "@onboarding", "@ONBOARD-E2E-011"] },
      async ({ page }) => {
        const onboarding = new OnboardingPage(page);
        await onboarding.fillAddress(BRANCH);
        await onboarding.searchAndPin();
        await onboarding.booking.fill("turnos.example.com");

        await onboarding.next.click();

        await expect(page.getByText("Ingresá un link que empiece con https://")).toBeVisible();
        await onboarding.expectStep(2);
      },
    );

    test(
      "Saves the branch and moves on to the services",
      { tag: ["@critical", "@e2e", "@onboarding", "@ONBOARD-E2E-012"] },
      async ({ page }) => {
        const onboarding = new OnboardingPage(page);
        await onboarding.fillAddress(BRANCH);
        await onboarding.searchAndPin();
        await onboarding.booking.fill(BRANCH.booking);

        await onboarding.next.click();

        await onboarding.expectStep(3);
      },
    );
  });

  test.describe("Step 3: services", () => {
    test(
      "Starts empty and asks for a name before saving a service",
      { tag: ["@high", "@e2e", "@onboarding", "@ONBOARD-E2E-013"] },
      async ({ page }) => {
        const onboarding = new OnboardingPage(page);
        await onboarding.expectStep(3);
        await expect(page.getByText("Todavía no cargaste ningún servicio.")).toBeVisible();

        await page.getByRole("button", { name: "Agregar servicio" }).click();
        const sheet = new ServiceSheet(page);
        await sheet.save.click();

        await expect(sheet.dialog.getByText("Ingresá el nombre del servicio")).toBeVisible();
        await expect(sheet.dialog).toBeVisible();
      },
    );

    test(
      "Adds a service with days, hours and price",
      { tag: ["@critical", "@e2e", "@onboarding", "@ONBOARD-E2E-014"] },
      async ({ page }) => {
        await page.getByRole("button", { name: "Agregar servicio" }).click();
        const sheet = new ServiceSheet(page);
        await sheet.fill({
          name: "Yoga Vinyasa Flow",
          description: "Clase guiada para todos los niveles.",
          price: "5500",
          days: ["Lunes", "Miércoles", "Viernes"],
          ranges: [["09:00", "10:00"]],
        });
        await sheet.saveAndClose();

        const item = page.getByRole("listitem").filter({ hasText: "Yoga Vinyasa Flow" });
        await expect(item).toBeVisible();
        await expect(item).toContainText("Lun, Mié y Vie · 09:00 a 10:00");
      },
    );

    test(
      "Edits and removes services",
      { tag: ["@high", "@e2e", "@onboarding", "@ONBOARD-E2E-015"] },
      async ({ page }) => {
        // Un segundo servicio que queda, y uno de paso que se quita.
        for (const name of ["Pilates Mat", "Stretching"]) {
          await page.getByRole("button", { name: "Agregar servicio" }).click();
          const sheet = new ServiceSheet(page);
          await sheet.fill({ name, days: ["Martes", "Jueves"], ranges: [["18:00", "19:00"]] });
          await sheet.saveAndClose();
          await expect(page.getByRole("listitem").filter({ hasText: name })).toBeVisible();
        }

        await page.getByRole("button", { name: "Quitar Stretching" }).click();
        await expect(page.getByText("Stretching")).toHaveCount(0);

        await page.getByRole("button", { name: "Editar Yoga Vinyasa Flow" }).click();
        const sheet = new ServiceSheet(page);
        await expect(sheet.name).toHaveValue("Yoga Vinyasa Flow");
        await expect(sheet.price).toHaveValue("5500");
        await sheet.name.fill("Yoga Vinyasa");
        await sheet.saveAndClose();

        await expect(page.getByRole("listitem").filter({ hasText: "Yoga Vinyasa" })).toBeVisible();
        await expect(page.getByText("Yoga Vinyasa Flow")).toHaveCount(0);
        await expect(page.locator("#step-form li")).toHaveCount(2);
      },
    );

    test(
      "Resumes at the services and keeps the branch when going back",
      { tag: ["@high", "@e2e", "@onboarding", "@ONBOARD-E2E-016"] },
      async ({ page }) => {
        const onboarding = new OnboardingPage(page);
        await onboarding.expectStep(3);
        await expect(page.locator("#step-form li")).toHaveCount(2);

        await onboarding.back.click();

        await onboarding.expectStep(2);
        await expect(onboarding.address).toHaveValue(BRANCH.address);
        await expect(onboarding.reference).toHaveValue(BRANCH.reference);
        await expect(onboarding.booking).toHaveValue(BRANCH.booking);
        await expect(onboarding.coordinates).toHaveText("Lat -32.1925 · Lng -64.4198");
      },
    );
  });

  test.describe("Step 4: photos and review", () => {
    test(
      "Rejects a file that is not an image",
      { tag: ["@high", "@e2e", "@onboarding", "@ONBOARD-E2E-017"] },
      async ({ page }) => {
        const onboarding = new OnboardingPage(page);
        await onboarding.next.click();
        await onboarding.expectStep(4);

        await onboarding.fileInput.setInputFiles({
          name: "notas.txt",
          mimeType: "text/plain",
          buffer: Buffer.from("no soy una foto"),
        });

        await expect(page.getByText(/Ese archivo no es una foto/)).toBeVisible();
        await expect(page.getByText("0 de 6 subidas")).toBeVisible();
      },
    );

    test(
      "Uploads photos, the first one becomes the cover and the image loads from storage",
      { tag: ["@critical", "@e2e", "@onboarding", "@ONBOARD-E2E-018"] },
      async ({ page }) => {
        const onboarding = new OnboardingPage(page);
        await onboarding.next.click();
        await onboarding.expectStep(4);

        await onboarding.fileInput.setInputFiles([
          photo("fachada.png", 250, 10, 10),
          photo("sala.png", 10, 250, 10),
          photo("recepcion.png", 10, 10, 250),
        ]);

        await expect(page.getByText("3 de 6 subidas")).toBeVisible();
        await expect(page.getByText("PORTADA", { exact: true })).toBeVisible();

        const cover = page.getByAltText("Foto de portada del local");
        await expect(cover).toBeVisible();
        await expect
          .poll(() => cover.evaluate((image: HTMLImageElement) => image.naturalWidth))
          .toBeGreaterThan(0);
      },
    );

    test(
      "Changes the cover and removes a photo",
      { tag: ["@high", "@e2e", "@onboarding", "@ONBOARD-E2E-019"] },
      async ({ page }) => {
        const onboarding = new OnboardingPage(page);
        await onboarding.next.click();
        await onboarding.expectStep(4);
        await expect(page.getByText("3 de 6 subidas")).toBeVisible();

        const cover = page.getByAltText("Foto de portada del local");
        const keyOf = async () => (await cover.getAttribute("src"))?.split("?")[0];
        const before = await keyOf();

        await page.getByRole("button", { name: "Hacer portada" }).first().click();
        await expect.poll(keyOf).not.toBe(before);
        await expect(page.getByText("3 de 6 subidas")).toBeVisible();

        await page.getByRole("button", { name: "Quitar la foto 2" }).click();
        await expect(page.getByText("2 de 6 subidas")).toBeVisible();
      },
    );

    test(
      "Stops offering uploads at six photos and brings them back after removing one",
      { tag: ["@medium", "@e2e", "@onboarding", "@ONBOARD-E2E-020"] },
      async ({ page }) => {
        const onboarding = new OnboardingPage(page);
        await onboarding.next.click();
        await onboarding.expectStep(4);
        await expect(page.getByText("2 de 6 subidas")).toBeVisible();

        await onboarding.fileInput.setInputFiles([
          photo("terraza.png", 200, 200, 10),
          photo("vestuario.png", 200, 10, 200),
          photo("patio.png", 10, 200, 200),
          photo("entrada.png", 120, 120, 120),
        ]);

        await expect(page.getByText("6 de 6 subidas")).toBeVisible();
        await expect(page.getByText("+ Agregar más fotos")).toHaveCount(0);

        await page.getByRole("button", { name: "Quitar la foto 6" }).click();
        await expect(page.getByText("5 de 6 subidas")).toBeVisible();
        await expect(page.getByText("+ Agregar más fotos")).toBeVisible();
      },
    );

    test(
      "Sends the business to review with the Mercado Pago payer",
      { tag: ["@critical", "@e2e", "@onboarding", "@ONBOARD-E2E-021"] },
      async ({ page }) => {
        const onboarding = new OnboardingPage(page);
        await onboarding.next.click();
        await onboarding.expectStep(4);

        await onboarding.submitForReview.click();
        await expect(page.getByText("Ingresá el email con el que pagaste")).toBeVisible();
        await expect(page.getByText("Ingresá tu nombre completo")).toBeVisible();

        await onboarding.payerEmail.fill("ana@example.com");
        await onboarding.payerName.fill("Ana Pérez");
        await onboarding.submitForReview.click();

        await expect(
          page.getByRole("heading", { name: "Tu negocio está en revisión" }),
        ).toBeVisible();
        await expect(page.getByText(BUSINESS.name)).toBeVisible();
        await expect(page.getByText("2 servicios")).toBeVisible();
        await expect(
          page.getByText(`+54 ${BUSINESS.phone.replace(/^(\d{2}) (\d{4}) (\d{4})$/, "$1 $2 $3")}`),
        ).toBeVisible();
        await expect(page.getByText(`${BRANCH.address}, ${BRANCH.city}`)).toBeVisible();
      },
    );
  });

  test.describe("After sending", () => {
    test(
      "Keeps showing the review screen and lets the owner reach the panel",
      { tag: ["@critical", "@e2e", "@onboarding", "@ONBOARD-E2E-022"] },
      async ({ page }) => {
        const onboarding = new OnboardingPage(page);
        await onboarding.gotoStart();
        await expect(
          page.getByRole("heading", { name: "Tu negocio está en revisión" }),
        ).toBeVisible();

        await page.getByRole("link", { name: "Ver mi panel" }).click();

        await expect(page).toHaveURL(/\/socios\/panel$/);
        await expect(page.getByText(`Socios · ${BUSINESS.name}`)).toBeVisible();
      },
    );
  });
});
