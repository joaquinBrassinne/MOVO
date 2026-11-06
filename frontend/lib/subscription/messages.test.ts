import assert from "node:assert/strict";
import { test } from "node:test";

import { alertView, downgradeMessage, statusCopy, unredeemedMessage } from "./messages.ts";
import type { PlanAlert, Subscription } from "./types.ts";

const alert = (type: string, payload: unknown): PlanAlert => ({
  id: 1,
  type,
  payload,
  created_at: "2026-11-01T10:00:00Z",
});

const unredeemed = {
  period_days: 30,
  coupons_generated: 40,
  coupons_redeemed: 4,
  redeem_rate: 0.1,
  threshold: 0.2,
};

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

test("unredeemedMessage states the counts, the period and the rounded rate", () => {
  assert.match(
    unredeemedMessage(unredeemed),
    /^Solo 4 de cada 40 cupones de los últimos 30 días se canjearon \(10 %\)\./,
  );
  assert.match(unredeemedMessage({ ...unredeemed, redeem_rate: 0.1234 }), /\(12 %\)/);
});

test("alertView links an unredeemed alert to the metrics", () => {
  const view = alertView(alert("unredeemed_coupons", unredeemed));
  assert.equal(view?.href, "/owners/metrics");
  assert.match(view?.body ?? "", /Solo 4 de cada 40/);
});

test("alertView shows the downgrade notice without a link", () => {
  const view = alertView(
    alert("plan_downgrade", { plan_from: "pro", plan_to: "starter", branches: [], services: [] }),
  );
  assert.equal(view?.title, "Cambió tu plan");
  assert.equal(view?.href, undefined);
});

test("alertView skips unknown types and malformed payloads instead of crashing", () => {
  assert.equal(alertView(alert("something_new", {})), null);
  assert.equal(alertView(alert("unredeemed_coupons", null)), null);
  assert.equal(alertView(alert("unredeemed_coupons", { period_days: 30 })), null);
  assert.equal(alertView(alert("plan_downgrade", { plan_from: "pro" })), null);
});
