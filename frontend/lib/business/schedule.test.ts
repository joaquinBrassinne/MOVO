import assert from "node:assert/strict";
import { test } from "node:test";

import {
  buildSlots,
  describeDays,
  findOverlap,
  isValidRange,
  toRanges,
  type Range,
} from "./schedule.ts";

const evening: Range = { start: "18:00", end: "19:00" };
const late: Range = { start: "19:30", end: "20:30" };

test("buildSlots repeats every range on every selected day", () => {
  const slots = buildSlots([1, 3, 5], [evening, late]);

  assert.equal(slots.length, 6);
  assert.deepEqual(slots[0], { weekday: 1, start: "18:00", end: "19:00" });
  assert.deepEqual(slots[5], { weekday: 5, start: "19:30", end: "20:30" });
});

test("buildSlots ignores repeated days and orders the result", () => {
  const slots = buildSlots([3, 1, 3], [evening]);

  assert.deepEqual(
    slots.map((s) => s.weekday),
    [1, 3],
  );
});

test("buildSlots gives nothing without days or ranges", () => {
  assert.deepEqual(buildSlots([], [evening]), []);
  assert.deepEqual(buildSlots([1], []), []);
});

test("isValidRange needs HH:MM times in order", () => {
  assert.equal(isValidRange({ start: "09:00", end: "10:30" }), true);
  assert.equal(isValidRange({ start: "10:00", end: "10:00" }), false);
  assert.equal(isValidRange({ start: "11:00", end: "10:00" }), false);
  assert.equal(isValidRange({ start: "9:00", end: "10:00" }), false);
  assert.equal(isValidRange({ start: "", end: "10:00" }), false);
  assert.equal(isValidRange({ start: "09:00", end: "24:00" }), false);
});

test("findOverlap finds two ranges that share time, and allows touching ones", () => {
  assert.deepEqual(findOverlap([evening, { start: "18:30", end: "19:30" }]), [0, 1]);
  assert.equal(findOverlap([evening, { start: "19:00", end: "20:00" }]), null);
  assert.equal(findOverlap([late, evening]), null);
  assert.equal(findOverlap([]), null);
});

test("toRanges recovers the days and ranges of a uniform schedule", () => {
  const slots = buildSlots([1, 3, 5], [evening, late]);

  const back = toRanges(slots);
  assert.deepEqual(back?.days, [1, 3, 5]);
  assert.deepEqual(back?.ranges, [evening, late]);
});

test("toRanges refuses a schedule that differs between days", () => {
  const uneven = [
    { weekday: 1, start: "09:00", end: "10:00" },
    { weekday: 2, start: "18:00", end: "19:00" },
  ];

  assert.equal(toRanges(uneven), null);
});

test("describeDays names runs of days the way the panel shows them", () => {
  assert.equal(describeDays([1, 2, 3, 4, 5, 6]), "Lun a Sáb");
  assert.equal(describeDays([2, 4]), "Mar y Jue");
  assert.equal(describeDays([1, 3, 5]), "Lun, Mié y Vie");
  assert.equal(describeDays([0]), "Dom");
  assert.equal(describeDays([]), "");
});
