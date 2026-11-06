import assert from "node:assert/strict";
import { test } from "node:test";

import {
  addDays,
  argentineToday,
  defaultPaidUntil,
  describeDaysLeft,
  formatDate,
  initials,
  paymentState,
} from "./format.ts";

test("a date is formatted from its parts, never through a timezone", () => {
  assert.equal(formatDate("2026-11-30"), "30 nov 2026");
  assert.equal(formatDate("2026-01-01"), "1 ene 2026");
  assert.equal(formatDate(""), "");
  assert.equal(formatDate("not-a-date"), "not-a-date");
});

test("the Argentine day is UTC-3, so late evening UTC is still the previous day there", () => {
  assert.equal(argentineToday(new Date("2026-11-30T02:59:00Z")), "2026-11-29");
  assert.equal(argentineToday(new Date("2026-11-30T03:00:00Z")), "2026-11-30");
});

test("the default paid_until is the Argentine today plus 30 days, across month and year ends", () => {
  assert.equal(defaultPaidUntil(new Date("2026-10-31T12:00:00Z")), "2026-11-30");
  assert.equal(defaultPaidUntil(new Date("2026-12-15T12:00:00Z")), "2027-01-14");
  assert.equal(defaultPaidUntil(new Date("2026-03-01T01:00:00Z")), "2026-03-30");
  assert.equal(addDays("2028-02-28", 1), "2028-02-29");
});

test("days left read naturally on both sides of the expiry", () => {
  assert.equal(describeDaysLeft(5), "en 5 días");
  assert.equal(describeDaysLeft(1), "mañana");
  assert.equal(describeDaysLeft(0), "hoy");
  assert.equal(describeDaysLeft(-1), "venció ayer");
  assert.equal(describeDaysLeft(-3), "vencido hace 3 días");
});

test("the payment state merges the paid status with the expiry warning", () => {
  assert.equal(paymentState({ status: "active", expiring_soon: false }), "ok");
  assert.equal(paymentState({ status: "active", expiring_soon: true }), "expiring");
  assert.equal(paymentState({ status: "grace", expiring_soon: false }), "grace");
  assert.equal(paymentState({ status: "expired", expiring_soon: false }), "expired");
});

test("initials take up to two words", () => {
  assert.equal(initials("Alma Haus Training Club"), "AH");
  assert.equal(initials("movo"), "M");
  assert.equal(initials(" "), "?");
});
