import assert from "node:assert/strict";
import { test } from "node:test";

import { formatPrice, planFeatures } from "./format.ts";
import type { Plan } from "./types.ts";

const starter: Plan = {
  plan: "starter",
  price: 19800,
  checkout_url: "https://example.com/s",
  max_branches: 2,
  max_active_services: 3,
};
const pro: Plan = { ...starter, plan: "pro", max_branches: 4, max_active_services: null };

test("formatPrice uses es-AR thousands separators and no decimals", () => {
  assert.equal(formatPrice(19800), "$19.800");
  assert.equal(formatPrice(29800), "$29.800");
  assert.equal(formatPrice(500), "$500");
});

test("starter bullets take the limits from the plan", () => {
  const bullets = planFeatures(starter);
  assert.ok(bullets.includes("Hasta 3 servicios"));
  assert.ok(bullets.includes("Hasta 2 sedes"));
  assert.ok(bullets.includes("Soporte estándar"));
});

test("pro bullets show unlimited services when the limit is null", () => {
  const bullets = planFeatures(pro);
  assert.ok(bullets.includes("Servicios ilimitados"));
  assert.ok(bullets.includes("Hasta 4 sedes"));
  assert.ok(bullets.includes("Soporte prioritario"));
  assert.ok(bullets.some((b) => b.startsWith("Visibilidad destacada")));
});

test("a limit of one is singular", () => {
  const bullets = planFeatures({ ...starter, max_branches: 1, max_active_services: 1 });
  assert.ok(bullets.includes("Hasta 1 servicio"));
  assert.ok(bullets.includes("Hasta 1 sede"));
});

test("a pro plan with a numeric limit shows it", () => {
  assert.ok(planFeatures({ ...pro, max_active_services: 10 }).includes("Hasta 10 servicios"));
});
