import { type Locator, type Page } from "@playwright/test";

import { BasePage } from "../base-page";

/** Las dos pantallas de acceso del portal de socios: ingresar y crear cuenta. */
export class AccessPage extends BasePage {
  readonly loginEmail: Locator;
  readonly loginPassword: Locator;
  readonly loginButton: Locator;

  readonly registerEmail: Locator;
  readonly registerPassword: Locator;
  readonly registerConfirm: Locator;
  readonly registerButton: Locator;

  constructor(page: Page) {
    super(page);
    this.loginEmail = page.getByLabel("Email comercial");
    this.loginPassword = page.getByLabel("Contraseña", { exact: true });
    this.loginButton = page.getByRole("button", { name: "Ingresar al panel" });

    this.registerEmail = page.getByLabel("Email");
    this.registerPassword = page.getByLabel("Contraseña", { exact: true });
    this.registerConfirm = page.getByLabel("Repetir contraseña");
    this.registerButton = page.getByRole("button", { name: "Crear cuenta" });
  }

  async gotoLogin(): Promise<void> {
    await this.goto("/socios/ingresar");
  }

  async gotoRegister(): Promise<void> {
    await this.goto("/socios/crear-cuenta");
  }

  async login(email: string, password: string): Promise<void> {
    await this.loginEmail.fill(email);
    await this.loginPassword.fill(password);
    await this.loginButton.click();
  }

  async register(email: string, password: string, confirm = password): Promise<void> {
    await this.registerEmail.fill(email);
    await this.registerPassword.fill(password);
    await this.registerConfirm.fill(confirm);
    await this.registerButton.click();
  }
}
