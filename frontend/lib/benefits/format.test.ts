import assert from "node:assert/strict";
import { test } from "node:test";

import {
  benefitHeadline,
  benefitWorth,
  formatPesos,
  initialOf,
  relativeTime,
  shortName,
  timeLeft,
} from "./format.ts";
import type { ClaimedBenefit } from "./types.ts";

const base: ClaimedBenefit = {
  id: 1,
  type: "percent",
  percent: 15,
  amount_ars: null,
  title: null,
  conditions: null,
  short_label: "-15%",
};

test("pesos are written with a dot between thousands", () => {
  assert.equal(formatPesos(150), "150");
  assert.equal(formatPesos(2000), "2.000");
  assert.equal(formatPesos(1234567), "1.234.567");
});

test("each benefit type has a headline and a short worth", () => {
  const amount: ClaimedBenefit = { ...base, type: "amount", percent: null, amount_ars: 2000 };
  const custom: ClaimedBenefit = {
    ...base,
    type: "custom",
    percent: null,
    title: "Una clase gratis",
  };

  assert.equal(benefitHeadline(base), "15% de descuento");
  assert.equal(benefitHeadline(amount), "$2.000 de descuento");
  assert.equal(benefitHeadline(custom), "Una clase gratis");
  assert.equal(benefitWorth(base), "15 %");
  assert.equal(benefitWorth(amount), "$2.000");
  assert.equal(benefitWorth(custom), "Una clase gratis");
});

test("names show an initial and an abbreviated form", () => {
  assert.equal(initialOf("lucía"), "L");
  assert.equal(initialOf("  Ñandú"), "Ñ");
  assert.equal(initialOf("123"), "?");
  assert.equal(shortName("Lucía Martínez"), "Lucía M.");
  assert.equal(shortName("Ana María de la Torre"), "Ana T.");
  assert.equal(shortName("Tomás"), "Tomás");
  assert.equal(shortName("  "), "");
});

test("a moment is told relative to now", () => {
  const now = new Date("2026-10-27T12:00:00Z");
  const ago = (ms: number) => new Date(now.getTime() - ms).toISOString();
  const min = 60_000;

  assert.equal(relativeTime(ago(10_000), now), "recién");
  assert.equal(relativeTime(ago(2 * min), now), "hace 2 min");
  assert.equal(relativeTime(ago(60 * min), now), "hace 1 h");
  assert.equal(relativeTime(ago(5 * 60 * min), now), "hace 5 h");
  assert.equal(relativeTime(ago(30 * 60 * min), now), "ayer");
  assert.equal(relativeTime(ago(3 * 24 * 60 * min), now), "hace 3 días");
  assert.equal(relativeTime("not a date", now), "");
});

test("the time left of a coupon", () => {
  const now = new Date("2026-10-27T12:00:00Z");
  const at = (ms: number) => new Date(now.getTime() + ms).toISOString();
  const min = 60_000;

  assert.equal(timeLeft(at(31 * 60 * min), now), "vence en 31 h");
  assert.equal(timeLeft(at(25 * min), now), "vence en 25 min");
  assert.equal(timeLeft(at(10_000), now), "vence en 1 min");
  assert.equal(timeLeft(at(-1), now), "venció");
});
