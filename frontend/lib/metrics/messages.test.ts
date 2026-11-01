import assert from "node:assert/strict";
import { test } from "node:test";

import {
  direction,
  failureText,
  formatArs,
  formatChange,
  formatMultiple,
  formatPercent,
  formatPoints,
  formatStars,
  formatStarsChange,
  heatLevel,
  insufficientText,
  monthLabel,
  peakCell,
  peakText,
  quotaCopy,
  recordProgressText,
  recordsInsufficientText,
  returnFormula,
  returnMultiple,
  slotName,
  weekdayLong,
  weekdayShort,
} from "./messages.ts";

test("formatPercent shows comma decimals and no trailing zero", () => {
  assert.equal(formatPercent(0.084), "8,4 %");
  assert.equal(formatPercent(0.25), "25 %");
  assert.equal(formatPercent(0.65, 0), "65 %");
  assert.equal(formatPercent(null), "—");
});

test("formatChange signs the movement and hides a missing baseline", () => {
  assert.equal(formatChange(0.12), "+12 %");
  assert.equal(formatChange(-0.03), "-3 %");
  assert.equal(formatChange(0), "0 %");
  assert.equal(formatChange(-0.001), "0 %");
  assert.equal(formatChange(null), null);
});

test("formatPoints reads a rate change in percentage points", () => {
  assert.equal(formatPoints(0.012), "+1,2 pp");
  assert.equal(formatPoints(-0.035), "-3,5 pp");
  assert.equal(formatPoints(null), null);
});

test("direction maps the sign", () => {
  assert.equal(direction(0.1), "up");
  assert.equal(direction(-0.1), "down");
  assert.equal(direction(0), "flat");
  assert.equal(direction(null), null);
});

test("formatArs groups thousands with dots", () => {
  assert.equal(formatArs(184000), "$184.000");
  assert.equal(formatArs(6500), "$6.500");
  assert.equal(formatArs(950), "$950");
  assert.equal(formatArs(1234567), "$1.234.567");
});

test("stars formatting", () => {
  assert.equal(formatStars(4.56), "4,6");
  assert.equal(formatStars(null), "—");
  assert.equal(formatStarsChange(0.2), "+0,2");
  assert.equal(formatStarsChange(-0.14), "-0,1");
  assert.equal(formatStarsChange(null), null);
});

test("weekday and slot labels", () => {
  assert.equal(weekdayShort(1), "Lun");
  assert.equal(weekdayShort(6), "Sáb");
  assert.equal(weekdayLong(7), "domingo");
  assert.equal(weekdayShort(9), "—");
  assert.equal(slotName("evening"), "Noche");
});

test("peakCell picks the busiest cell and ignores an empty map", () => {
  const cells = [
    { weekday: 1, slot: 0, redemptions: 0 },
    { weekday: 6, slot: 3, redemptions: 9 },
    { weekday: 7, slot: 3, redemptions: 9 },
    { weekday: 2, slot: 1, redemptions: 4 },
  ];
  assert.deepEqual(peakCell(cells), cells[1]);
  assert.equal(peakCell([{ weekday: 1, slot: 0, redemptions: 0 }]), null);
  assert.equal(peakText(cells[1] ?? null), "Pico: sábado, noche (18 a 24 h)");
  assert.equal(peakText(null), "Todavía no hay un pico");
});

test("heatLevel is relative to the peak", () => {
  assert.equal(heatLevel(0, 10), 0);
  assert.equal(heatLevel(1, 10), 1);
  assert.equal(heatLevel(5, 10), 2);
  assert.equal(heatLevel(10, 10), 4);
  assert.equal(heatLevel(3, 0), 0);
});

test("monthLabel names the month", () => {
  assert.equal(monthLabel("2026-10"), "octubre de 2026");
  assert.equal(monthLabel("nope"), "nope");
});

test("recordProgressText says how far the month went or that it is a record", () => {
  const base = { month: "2026-11", redemptions: 42, progress: 0.65, record_in_progress: false };
  assert.equal(recordProgressText(base), "Vas al 65 %");
  assert.equal(
    recordProgressText({ ...base, progress: 1.1, record_in_progress: true }),
    "Récord en curso",
  );
});

test("quotaCopy maps every status", () => {
  assert.match(quotaCopy({ status: "no_quota" }).text, /no tiene un tope/);
  assert.match(quotaCopy({ status: "insufficient_data", minimum_days: 3 }).text, /3 días/);
  const exhausted = quotaCopy({ status: "exhausted", resets_on: "2026-12-01" });
  assert.equal(exhausted.tone, "warning");
  assert.match(exhausted.text, /1 de diciembre/);
  const runsOut = quotaCopy({ status: "will_run_out", exhausts_on: "2026-11-22" });
  assert.equal(runsOut.text, "A este ritmo, tu cupo se agota el 22 de noviembre.");
  assert.equal(runsOut.tone, "warning");
  assert.match(quotaCopy({ status: "on_track", resets_on: "2026-12-01" }).text, /alcanza/);
});

test("the return multiple only exists for a 30-day period", () => {
  assert.equal(returnMultiple(184000, 29800, 7), null);
  assert.equal(returnMultiple(184000, 0, 30), null);
  assert.equal(formatMultiple(returnMultiple(184000, 29800, 30) ?? 0), "6,2x");
  assert.equal(returnFormula(1, 6500), "1 canje × $6.500 de ticket promedio");
  assert.equal(returnFormula(28, 6500), "28 canjes × $6.500 de ticket promedio");
});

test("records name the history that is missing", () => {
  const text = recordsInsufficientText({
    status: "insufficient_data",
    history_months: 1,
    minimum_history_months: 2,
    minimum_best_month_redemptions: 20,
  });
  assert.match(text, /2 meses completos/);
  assert.match(text, /20 canjes/);
});

test("state copy", () => {
  assert.equal(insufficientText(20), "Sin datos suficientes todavía (mínimo 20 cupones)");
  assert.equal(insufficientText(undefined), "Sin datos suficientes todavía (mínimo 20 cupones)");
  assert.equal(failureText("plan_upgrade_required"), "Esta métrica es parte del Plan Pro.");
  assert.equal(failureText("network_error"), "No pudimos cargar esta métrica.");
});
