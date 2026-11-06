import { expect, type Locator, type Page } from "@playwright/test";

import { BasePage } from "../base-page";

/** El alta en cuatro pasos (`/owners/onboarding`) y su pantalla de espera. */
export class OnboardingPage extends BasePage {
  readonly next: Locator;
  readonly back: Locator;

  // Paso 1
  readonly businessName: Locator;
  readonly businessDescription: Locator;
  readonly businessPhone: Locator;

  // Paso 2
  readonly address: Locator;
  readonly city: Locator;
  readonly reference: Locator;
  readonly booking: Locator;
  readonly searchOnMap: Locator;
  readonly map: Locator;
  readonly coordinates: Locator;

  // Paso 4
  readonly fileInput: Locator;
  readonly payerEmail: Locator;
  readonly payerName: Locator;
  readonly submitForReview: Locator;

  constructor(page: Page) {
    super(page);
    this.next = page.getByRole("button", { name: "Siguiente" });
    this.back = page.getByRole("button", { name: "Atrás" });

    this.businessName = page.getByLabel("Nombre del negocio");
    this.businessDescription = page.getByLabel("Descripción");
    this.businessPhone = page.getByLabel("WhatsApp oficial");

    this.address = page.getByLabel("Dirección del local");
    this.city = page.getByLabel("Ciudad o localidad");
    this.reference = page.getByLabel("Piso / Depto / Timbre");
    this.booking = page.getByLabel("Link de reservas online");
    this.searchOnMap = page.getByRole("button", { name: "Buscar en el mapa" });
    this.map = page.getByRole("application", { name: "Mapa para ubicar tu sede" });
    this.coordinates = page.getByText(/^Lat -?\d+\.\d{4} · Lng -?\d+\.\d{4}$/);

    this.fileInput = page.locator('input[type="file"]');
    this.payerEmail = page.getByLabel("Email con el que pagaste en Mercado Pago");
    this.payerName = page.getByLabel("Nombre completo del titular");
    this.submitForReview = page.getByRole("button", { name: "Enviar a revisión" });
  }

  async gotoStart(): Promise<void> {
    await this.goto("/owners/onboarding");
  }

  async expectStep(step: 1 | 2 | 3 | 4): Promise<void> {
    await expect(this.page.getByRole("heading", { level: 1 })).toContainText(`Paso ${step} De 4`);
  }

  async fillBusiness(data: {
    name: string;
    category: string;
    description?: string;
    phone: string;
  }): Promise<void> {
    await this.businessName.fill(data.name);
    await this.page.getByRole("radio", { name: data.category }).click();
    if (data.description) await this.businessDescription.fill(data.description);
    await this.businessPhone.fill(data.phone);
  }

  async fillAddress(data: { address: string; city: string; reference?: string }): Promise<void> {
    await this.address.fill(data.address);
    await this.city.fill(data.city);
    if (data.reference) await this.reference.fill(data.reference);
  }

  /** Busca la dirección y espera a que el mapa tenga el pin. */
  async searchAndPin(): Promise<void> {
    await this.searchOnMap.click();
    await expect(this.coordinates).toBeVisible();
  }

  async tapMap(x: number, y: number): Promise<void> {
    await this.map.click({ position: { x, y } });
  }
}
