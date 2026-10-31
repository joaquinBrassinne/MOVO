import { crc32, deflateSync } from "node:zlib";

import type { Page } from "@playwright/test";

import { ADMIN } from "./support/env.mjs";
import { AccessPage } from "./pages/access-page";

export const PASSWORD = "ClaveSegura#2026";

export const OWNER = { email: "duena-e2e@example.com", password: PASSWORD };
export { ADMIN };

export const BUSINESS = {
  name: "Estudio Aire Libre",
  category: "Yoga",
  description: "Clases de yoga y pilates en un espacio luminoso.",
  phone: "11 5824 9912",
};

export const BRANCH = {
  address: "Av. San Martín 100",
  city: "Río Tercero",
  reference: "PB al fondo",
  booking: "https://turnos.example.com",
};

/** Ingresa por la pantalla de acceso y espera a que termine la redirección. */
export async function signIn(page: Page, account: { email: string; password: string }) {
  const access = new AccessPage(page);
  await access.gotoLogin();
  await access.login(account.email, account.password);
  await page.waitForURL((url) => !url.pathname.endsWith("/owners/sign-in"));
}

function chunk(type: string, data: Buffer): Buffer {
  const body = Buffer.concat([Buffer.from(type, "ascii"), data]);
  const length = Buffer.alloc(4);
  length.writeUInt32BE(data.length);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(body));
  return Buffer.concat([length, body, crc]);
}

/** Un PNG válido de 1×1 píxel. Cada color da una imagen distinta (la API rechaza repetidas). */
export function png(red: number, green: number, blue: number): Buffer {
  const header = Buffer.alloc(13);
  header.writeUInt32BE(1, 0);
  header.writeUInt32BE(1, 4);
  header.set([8, 2, 0, 0, 0], 8);

  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", header),
    chunk("IDAT", deflateSync(Buffer.from([0, red, green, blue]))),
    chunk("IEND", Buffer.alloc(0)),
  ]);
}

export function photo(name: string, red: number, green: number, blue: number) {
  return { name, mimeType: "image/png", buffer: png(red, green, blue) };
}
