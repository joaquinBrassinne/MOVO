import { defineConfig, devices } from "@playwright/test";

import { PORTS } from "./e2e/support/env.mjs";
import { FAKE_CAMERA_FILE } from "./e2e/support/fake-camera.mjs";

// Los E2E levantan su propio entorno: base `movo_e2e` nueva, API, geocodificador falso y
// la web. Solo necesitan Postgres y RustFS corriendo (`make up`).
//
// Los specs son una secuencia (alta → aprobación → panel) que comparte la base, por eso
// van de a uno y en orden numérico: 01-access, 02-onboarding, 03-admin, 04-panel.
export default defineConfig({
  testDir: "./e2e",
  fullyParallel: false,
  workers: 1,
  forbidOnly: Boolean(process.env.CI),
  retries: 0,
  timeout: 90_000,
  expect: { timeout: 10_000 },
  reporter: [["list"], ["html", { open: "never" }]],
  globalSetup: "./e2e/support/warm-up.ts",
  use: {
    baseURL: `http://localhost:${PORTS.web}`,
    locale: "es-AR",
    // La cámara es un video que cada test escribe (ver support/fake-camera.mjs) y los permisos
    // se conceden solos: lo que se prueba es la lectura, no el cuadro de permisos del navegador.
    launchOptions: {
      args: [
        "--use-fake-ui-for-media-stream",
        "--use-fake-device-for-media-stream",
        `--use-file-for-fake-video-capture=${FAKE_CAMERA_FILE}`,
      ],
    },
    actionTimeout: 15_000,
    navigationTimeout: 60_000,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
  webServer: [
    {
      command: "node e2e/support/geocoder-stub.mjs",
      url: `http://localhost:${PORTS.geocoder}/`,
      reuseExistingServer: false,
    },
    {
      command: "node e2e/support/start-api.mjs",
      url: `http://localhost:${PORTS.api}/healthz`,
      reuseExistingServer: false,
      timeout: 240_000,
    },
    {
      command: `npx next dev -p ${PORTS.web}`,
      url: `http://localhost:${PORTS.web}/socios/ingresar`,
      reuseExistingServer: false,
      timeout: 180_000,
      env: { API_URL: `http://localhost:${PORTS.api}`, NEXT_TELEMETRY_DISABLED: "1" },
    },
  ],
});
