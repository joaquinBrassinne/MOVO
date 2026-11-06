import assert from "node:assert/strict";
import { test } from "node:test";

import { describeAdminFailure, describeAuditAction, STATUS_LABEL } from "./messages.ts";

const failure = (code: string, status = 409) => ({ status, error: { code, message: "x" } });

test("a state conflict is explained in the admin's words, not the owner's", () => {
  const message = describeAdminFailure(failure("invalid_state"));

  assert.match(message.title, /no corresponde/i);
  assert.doesNotMatch(message.body, /enviado a revisión/i);
});

test("a concurrent change tells the admin to review and retry", () => {
  assert.match(describeAdminFailure(failure("conflict")).body, /Revisalos/);
});

test("unknown failures fall back to the shared wording without leaking the server text", () => {
  const message = describeAdminFailure({
    status: 500,
    error: { code: "internal_error", message: "pq: boom" },
  });

  assert.ok(message.title);
  assert.doesNotMatch(JSON.stringify(message), /pq:/);
  assert.match(describeAdminFailure(failure("network_error", 0)).title, /conexión/i);
});

test("audit actions have readable names and unknown ones are shown as they are", () => {
  assert.equal(describeAuditAction("business.approve"), "Aprobado y publicado");
  assert.equal(describeAuditAction("business.verify"), "Marcado como verificado");
  assert.equal(describeAuditAction("business.profile_update"), "Datos del negocio actualizados");
  assert.equal(describeAuditAction("business.something_new"), "business.something_new");
});

test("billing errors have their own Spanish wording", () => {
  assert.match(describeAdminFailure(failure("business_not_found", 404)).title, /No encontramos/);
  assert.match(describeAdminFailure(failure("subscription_not_found", 404)).title, /suscripción/);
  assert.match(describeAdminFailure(failure("business_not_confirmable")).title, /No se puede/);
  assert.match(describeAdminFailure(failure("subscription_exists")).body, /Registrar pago/);
});

test("validation failures name the field and reason of the billing form", () => {
  const invalid = (field: string, reason: string) =>
    describeAdminFailure({
      status: 422,
      error: { code: "validation_failed", message: "x", details: { field, reason } },
    });

  assert.match(invalid("paid_until", "in_the_past").body, /anterior a hoy/);
  assert.match(invalid("paid_until", "invalid").body, /fecha válida/);
  assert.match(invalid("notes", "too_long").body, /demasiado largas/);
  assert.match(invalid("plan", "invalid").body, /plan válido/);
  assert.match(invalid("mp_subscription_ref", "too_long").body, /Mercado Pago/);
  assert.match(invalid("status", "invalid").body, /estado/);
  assert.ok(invalid("something_else", "x").title);
});

test("billing audit actions are readable", () => {
  assert.equal(describeAuditAction("billing.confirm_payment"), "Pago confirmado y local publicado");
  assert.equal(describeAuditAction("billing.register_payment"), "Pago mensual registrado");
  assert.equal(describeAuditAction("billing.change_plan"), "Plan cambiado");
});

test("every status has a label", () => {
  assert.deepEqual(Object.keys(STATUS_LABEL).sort(), [
    "active",
    "deleted",
    "draft",
    "paused",
    "pending_review",
  ]);
});
