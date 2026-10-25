// Deja una base `movo_e2e` vacía y migrada, compila la API y la deja corriendo.
// Cada corrida parte de cero: los E2E no dependen de datos de corridas anteriores ni
// tocan la base de desarrollo. Requiere `make up` (Postgres y RustFS en Docker).

import { spawn, spawnSync } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { API_ENV, DATABASE } from "./env.mjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../..");
const backend = path.join(root, "backend");
const binary = path.join(backend, "bin", process.platform === "win32" ? "api-e2e.exe" : "api-e2e");
const env = { ...process.env, ...API_ENV };

function run(command, args, options) {
  const result = spawnSync(command, args, { stdio: "inherit", ...options });
  if (result.status !== 0) {
    console.error(`failed: ${command} ${args.join(" ")}`);
    process.exit(result.status ?? 1);
  }
}

function psql(sql) {
  run(
    "docker",
    [
      ...["compose", "--env-file", ".env", "-f", "deploy/docker-compose.yml"],
      ...["exec", "-T", "postgres", "psql", "-U", "movo", "-d", "postgres"],
      ...["-v", "ON_ERROR_STOP=1", "-c", sql],
    ],
    { cwd: root },
  );
}

psql(`DROP DATABASE IF EXISTS ${DATABASE} WITH (FORCE)`);
psql(`CREATE DATABASE ${DATABASE}`);
run("go", ["run", "./cmd/migrate", "up"], { cwd: backend, env });
run("go", ["build", "-o", binary, "./cmd/api"], { cwd: backend, env });

const api = spawn(binary, [], { cwd: backend, env, stdio: "inherit" });
api.on("exit", (code) => process.exit(code ?? 0));
for (const signal of ["SIGINT", "SIGTERM"]) process.on(signal, () => api.kill());
