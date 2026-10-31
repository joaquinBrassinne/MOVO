// Valores compartidos por la configuración de Playwright y los scripts de soporte.
// Los puertos y la base son propios de los E2E para no pisar el entorno de desarrollo.

export const PORTS = { web: 3100, api: 8089, geocoder: 3199 };

export const DATABASE = "movo_e2e";
export const DATABASE_URL = `postgres://movo:movo@localhost:5432/${DATABASE}?sslmode=disable`;

export const ADMIN = { email: "admin-e2e@movo.test", password: "E2eAdminPass#2026" };

/** Variables de `cmd/api` y `cmd/migrate` para el entorno E2E. */
export const API_ENV = {
  HTTP_ADDR: `:${PORTS.api}`,
  DATABASE_URL,
  LOG_LEVEL: "warn",
  JWT_SECRET: "e2e-only-secret-change-me-0123456789abcdef",
  ACCESS_TOKEN_TTL: "15m",
  REFRESH_TOKEN_TTL: "720h",
  SUPER_ADMIN_EMAIL: ADMIN.email,
  SUPER_ADMIN_PASSWORD: ADMIN.password,
  PLAN_STARTER_MAX_BRANCHES: "2",
  PLAN_STARTER_MAX_ACTIVE_SERVICES: "3",
  PLAN_STARTER_PRICE: "19800",
  PLAN_PRO_MAX_BRANCHES: "4",
  PLAN_PRO_MAX_ACTIVE_SERVICES: "0",
  PLAN_PRO_PRICE: "29800",
  MP_STARTER_URL: "https://www.mercadopago.com.ar/subscriptions/checkout?preapproval_plan_id=e2e-starter",
  MP_PRO_URL: "https://www.mercadopago.com.ar/subscriptions/checkout?preapproval_plan_id=e2e-pro",
  S3_ENDPOINT: "http://localhost:9000",
  S3_BUCKET: "movo-e2e",
  S3_ACCESS_KEY: "movo",
  S3_SECRET_KEY: "movo12345",
  S3_PATH_STYLE: "true",
  S3_ENSURE_BUCKET: "true",
  S3_CORS_ORIGINS: `http://localhost:${PORTS.web}`,
  // Los specs de fotos suben seis y chocan con el límite: se fija acá para que no dependa del valor por defecto.
  MAX_BUSINESS_IMAGES: "6",
  // Next llega desde la propia máquina y reenvía la IP de cada test (support/client.ts): así cada
  // test cuenta contra su propio cupo de cupones, con el límite por defecto.
  TRUSTED_PROXIES: "127.0.0.1,::1",
  GEOCODER_URL: `http://localhost:${PORTS.geocoder}`,
  GEOCODER_USER_AGENT: "MOVO-e2e/0.1 (e2e@example.com)",
};
