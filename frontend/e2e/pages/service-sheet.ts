import { type Locator, type Page } from "@playwright/test";

export interface ServiceData {
  name: string;
  description?: string;
  price?: string;
  /** Nombres largos de los días: "Lunes", "Miércoles". */
  days?: string[];
  ranges?: [start: string, end: string][];
}

/** La hoja inferior para crear o editar un servicio; sirve en el alta y en el panel. */
export class ServiceSheet {
  readonly dialog: Locator;
  readonly name: Locator;
  readonly description: Locator;
  readonly price: Locator;
  readonly save: Locator;

  constructor(private readonly page: Page) {
    this.dialog = page.getByRole("dialog");
    this.name = this.dialog.getByLabel("Nombre del servicio");
    this.description = this.dialog.getByLabel("Descripción corta");
    this.price = this.dialog.getByLabel("Precio estimado");
    this.save = this.dialog.getByRole("button", { name: "Guardar" });
  }

  async fill(data: ServiceData): Promise<void> {
    await this.name.fill(data.name);
    if (data.description) await this.description.fill(data.description);
    if (data.price) await this.price.fill(data.price);

    for (const day of data.days ?? []) {
      await this.dialog.getByRole("button", { name: day, exact: true }).click();
    }
    for (const [start, end] of data.ranges ?? []) {
      await this.dialog.getByRole("button", { name: "Agregar franja" }).click();
      await this.dialog.getByLabel("Desde").fill(start);
      await this.dialog.getByLabel("Hasta").fill(end);
      await this.dialog.getByRole("button", { name: "Sumar" }).click();
    }
  }

  async saveAndClose(): Promise<void> {
    await this.save.click();
    await this.dialog.waitFor({ state: "hidden" });
  }
}
