import assert from "node:assert/strict";
import { test } from "node:test";

import { formatLongDate, joinNames, planFeatures } from "./format.ts";

test("formatLongDate keeps the calendar day whatever the timezone", () => {
  const previous = process.env.TZ;
  process.env.TZ = "America/Argentina/Buenos_Aires";
  try {
    assert.equal(formatLongDate("2026-11-30"), "30 de noviembre");
    assert.equal(formatLongDate("2026-01-01"), "1 de enero");
    assert.equal(formatLongDate("2026-12-07"), "7 de diciembre");
  } finally {
    if (previous === undefined) delete process.env.TZ;
    else process.env.TZ = previous;
  }
});

test("formatLongDate returns what it cannot read untouched", () => {
  assert.equal(formatLongDate("mañana"), "mañana");
  assert.equal(formatLongDate("2026-13-01"), "2026-13-01");
});

test("planFeatures builds the limit bullets from the API numbers", () => {
  const starter = planFeatures("starter", { max_branches: 2, max_active_services: 3 });
  assert.ok(starter.includes("Hasta 3 servicios activos"));
  assert.ok(starter.includes("Hasta 2 sedes"));
  assert.ok(starter.includes("Soporte estándar"));
  assert.ok(!starter.includes("Soporte prioritario"));
});

test("planFeatures treats null as unlimited and adds the Pro extras", () => {
  const pro = planFeatures("pro", { max_branches: 4, max_active_services: null });
  assert.ok(pro.includes("Servicios y horarios ilimitados"));
  assert.ok(pro.includes("Hasta 4 sedes"));
  assert.ok(pro.includes("Soporte prioritario"));
  assert.ok(pro.includes("Métricas de inteligencia comercial"));
});

test("planFeatures singularizes a limit of one", () => {
  const one = planFeatures("starter", { max_branches: 1, max_active_services: 1 });
  assert.ok(one.includes("Hasta 1 servicio activo"));
  assert.ok(one.includes("Hasta 1 sede"));
});

test("joinNames joins in Spanish", () => {
  assert.equal(joinNames(["Norte"]), "Norte");
  assert.equal(joinNames(["Norte", "Sur"]), "Norte y Sur");
  assert.equal(joinNames(["Norte", "Sur", "Yoga"]), "Norte, Sur y Yoga");
});
