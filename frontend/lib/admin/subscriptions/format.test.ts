import assert from "node:assert/strict";
import { test } from "node:test";

import {
  addDays,
  argentineToday,
  defaultPaidUntil,
  describeDaysLeft,
  filterRows,
  formatDate,
  initials,
  paymentState,
  tabCounts,
} from "./format.ts";
import type { SubscriptionRow } from "./types.ts";

const row = (overrides: Partial<SubscriptionRow>): SubscriptionRow => ({
  business_id: 1,
  plan: "starter",
  paid_until: "2026-11-30",
  grace_until: "2026-12-07",
  status: "active",
  business_name: "Alma Haus",
  business_status: "active",
  pilot: false,
  expiring_soon: false,
  days_left: 20,
  reminder_url: "",
  ...overrides,
});

const rows = [
  row({ business_id: 1, business_name: "Alma Haus", expiring_soon: true, days_left: 5 }),
  row({ business_id: 2, business_name: "Kinésis Studio", status: "grace", days_left: -3 }),
  row({ business_id: 3, business_name: "Boxing Gym", status: "expired", days_left: -15 }),
  row({ business_id: 4, business_name: "Yoga Sur" }),
  row({ business_id: 5, business_name: "Pilates Norte" }),
];

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

test("the tab counts are disjoint and add up to the total", () => {
  assert.deepEqual(tabCounts(rows), { all: 5, ok: 2, expiring: 1, grace: 1, expired: 1 });
  assert.deepEqual(tabCounts([]), { all: 0, ok: 0, expiring: 0, grace: 0, expired: 0 });
});

test("a tab keeps only its rows and the search ignores case and accents", () => {
  assert.deepEqual(
    filterRows(rows, "grace", "").map((r) => r.business_id),
    [2],
  );
  assert.equal(filterRows(rows, "all", "").length, 5);
  assert.deepEqual(
    filterRows(rows, "all", "KINESIS").map((r) => r.business_id),
    [2],
  );
  assert.deepEqual(filterRows(rows, "ok", "kinesis"), []);
});

test("initials take up to two words", () => {
  assert.equal(initials("Alma Haus Training Club"), "AH");
  assert.equal(initials("movo"), "M");
  assert.equal(initials(" "), "?");
});
