import assert from "node:assert/strict";
import { test } from "node:test";

import { relativeTime } from "./relativeTime.ts";

const now = new Date("2026-06-30T12:00:00Z");
const ago = (ms: number) => new Date(now.getTime() - ms).toISOString();
const MIN = 60_000;
const DAY = 24 * 60 * MIN;

test("very recent reviews read as just now", () => {
  assert.equal(relativeTime(ago(10_000), now), "hace un momento");
  assert.equal(relativeTime(ago(-5 * MIN), now), "hace un momento");
});

test("minutes and hours", () => {
  assert.equal(relativeTime(ago(MIN), now), "hace 1 minuto");
  assert.equal(relativeTime(ago(45 * MIN), now), "hace 45 minutos");
  assert.equal(relativeTime(ago(60 * MIN), now), "hace 1 hora");
  assert.equal(relativeTime(ago(5 * 60 * MIN), now), "hace 5 horas");
});

test("days, weeks, months and years use singular and plural", () => {
  assert.equal(relativeTime(ago(DAY), now), "hace 1 día");
  assert.equal(relativeTime(ago(3 * DAY), now), "hace 3 días");
  assert.equal(relativeTime(ago(7 * DAY), now), "hace 1 semana");
  assert.equal(relativeTime(ago(21 * DAY), now), "hace 3 semanas");
  assert.equal(relativeTime(ago(30 * DAY), now), "hace 1 mes");
  assert.equal(relativeTime(ago(90 * DAY), now), "hace 3 meses");
  assert.equal(relativeTime(ago(365 * DAY), now), "hace 1 año");
  assert.equal(relativeTime(ago(800 * DAY), now), "hace 2 años");
});

test("an invalid date does not break the card", () => {
  assert.equal(relativeTime("nope", now), "hace un momento");
});
