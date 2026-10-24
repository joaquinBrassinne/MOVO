import assert from "node:assert/strict";
import { test } from "node:test";

import { describeFailure } from "./messages.ts";

const failure = (code: string, details?: unknown, status = 422) => ({
  status,
  error: { code, message: "x", details },
});

test("a validation error points at the field it is about", () => {
  const feedback = describeFailure(
    failure("validation_failed", { field: "whatsapp", reason: "invalid" }),
  );

  assert.equal(feedback.fields.whatsapp, "Ingresá un WhatsApp válido");
  assert.equal(feedback.form, undefined);
});

test("the description limit is named in its message", () => {
  const feedback = describeFailure(
    failure("validation_failed", { field: "description", reason: "too_long" }),
  );

  assert.match(feedback.fields.description ?? "", /500/);
});

test("overlapping time slots are explained", () => {
  const feedback = describeFailure(
    failure("validation_failed", { field: "slots", reason: "overlap" }),
  );

  assert.match(feedback.fields.slots ?? "", /pisan/);
});

test("an unknown validation field falls back to a form message instead of vanishing", () => {
  const feedback = describeFailure(
    failure("validation_failed", { field: "owner_user_id", reason: "x" }),
  );

  assert.deepEqual(feedback.fields, {});
  assert.ok(feedback.form);
});

test("plan limits say which resource is full", () => {
  const branches = describeFailure(failure("plan_limit_reached", { resource: "branches" }, 409));
  const services = describeFailure(failure("plan_limit_reached", { resource: "services" }, 409));

  assert.match(branches.form?.body ?? "", /sede/i);
  assert.match(services.form?.body ?? "", /servicio/i);
});

test("an address that cannot be found is shown next to the address and offers the manual pin", () => {
  const feedback = describeFailure(failure("address_not_found", undefined, 404));

  assert.match(feedback.fields.address ?? "", /pin/);
});

test("the geocoder being busy or down is not blamed on the owner", () => {
  assert.match(
    describeFailure(failure("geocoder_busy", undefined, 429)).form?.body ?? "",
    /segundos/,
  );
  assert.match(
    describeFailure(failure("geocoder_unavailable", undefined, 502)).form?.body ?? "",
    /pin/,
  );
});

test("network and session failures have their own wording", () => {
  assert.match(
    describeFailure(failure("network_error", undefined, 0)).form?.title ?? "",
    /conexión/i,
  );
  assert.match(
    describeFailure(failure("unauthorized", undefined, 401)).form?.title ?? "",
    /sesión/i,
  );
});

test("an unknown error code never leaks its text to the owner", () => {
  const feedback = describeFailure({
    status: 500,
    error: { code: "internal_error", message: "pq: syntax error" },
  });

  assert.ok(feedback.form);
  assert.doesNotMatch(JSON.stringify(feedback), /pq:/);
});
