import { expect, type Page } from "@playwright/test";

/** Lo común a todas las pantallas: navegar y comprobar dónde se está. */
export class BasePage {
  constructor(protected readonly page: Page) {}

  async goto(path: string): Promise<void> {
    await this.page.goto(path);
  }

  async expectPath(path: string | RegExp): Promise<void> {
    await expect(this.page).toHaveURL(typeof path === "string" ? new RegExp(`${path}$`) : path);
  }
}
