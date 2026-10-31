import assert from "node:assert/strict";
import { test } from "node:test";

import { downgradeMessage, statusCopy } from "./messages.ts";
import type { Subscription } from "./types.ts";

const sub = (status: Subscription["status"]): Subscription => ({
  plan: "pro",
  paid_until: "2026-11-30",
  grace_until: "2026-12-07",
  status,
});

test("an active subscription shows a calm validity chip", () => {
  const copy = statusCopy({ pilot: false, subscription: sub("active") });
  assert.equal(copy.tone, "calm");
  assert.equal(copy.chip, "Vigente hasta el 30 de noviembre");
  assert.equal(copy.body, undefined);
});

test("grace warns with both dates", () => {
  const copy = statusCopy({ pilot: false, subscription: sub("grace") });
  assert.equal(copy.tone, "warning");
  assert.equal(
    copy.body,
    "Tu suscripción venció el 30 de noviembre. Seguís visible hasta el 7 de diciembre; si el cobro de Mercado Pago no se regulariza, tu local deja de aparecer en el mapa.",
  );
});

test("expired says the local left the map", () => {
  const copy = statusCopy({ pilot: false, subscription: sub("expired") });
  assert.equal(copy.tone, "danger");
  assert.equal(copy.title, "Tu local dejó de aparecer en el mapa");
});

test("a pilot shows no dates", () => {
  const copy = statusCopy({ pilot: true, subscription: null });
  assert.equal(copy.chip, "Local piloto");
  assert.match(copy.body ?? "", /acceso piloto sin costo, con los límites del plan Pro/);
});

test("no subscription and not pilot is waiting for review", () => {
  const copy = statusCopy({ pilot: false, subscription: null });
  assert.match(copy.body ?? "", /se confirma cuando el equipo de MOVO revise tu alta/);
});

test("downgradeMessage names the plans and what was switched off", () => {
  const text = downgradeMessage({
    plan_from: "pro",
    plan_to: "starter",
    branches: [{ id: 3, name: "Norte" }],
    services: [{ id: 9, name: "Yoga" }],
  });
  assert.match(text, /cambió de Pro a Starter/);
  assert.match(text, /Se desactivaron Norte y Yoga/);
  assert.match(text, /No se borró nada/);
});
