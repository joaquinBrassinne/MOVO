import assert from "node:assert/strict";
import { test } from "node:test";

import {
  addRange,
  copyDay,
  firstProblem,
  removeRange,
  setDayOpen,
  toDayHours,
  toSlots,
  updateRange,
} from "./hours.ts";

const slot = (weekday: number, start: string, end: string) => ({ weekday, start, end });
const range = (start: string, end: string) => ({ start, end });
const day = (rows: ReturnType<typeof toDayHours>, weekday: number) =>
  rows.find((row) => row.weekday === weekday)!;

test("toDayHours gives the seven days in week order, closed when a day has no slots", () => {
  const rows = toDayHours([slot(1, "08:00", "23:00"), slot(0, "10:00", "14:00")]);

  assert.deepEqual(
    rows.map((row) => row.weekday),
    [1, 2, 3, 4, 5, 6, 0],
  );
  assert.deepEqual(day(rows, 1).ranges, [range("08:00", "23:00")]);
  assert.deepEqual(day(rows, 2).ranges, []);
  assert.deepEqual(day(rows, 0).ranges, [range("10:00", "14:00")]);
});

test("toDayHours keeps split shifts as several ranges of the same day, in time order", () => {
  const rows = toDayHours([slot(1, "16:00", "20:00"), slot(1, "09:00", "13:00")]);

  assert.deepEqual(day(rows, 1).ranges, [range("09:00", "13:00"), range("16:00", "20:00")]);
});

test("toSlots lists every range of every day, in week order", () => {
  const rows = toDayHours([slot(6, "09:00", "22:00"), slot(1, "08:00", "12:00")]);
  const edited = addRange(setDayOpen(rows, 3, true), 1);

  assert.deepEqual(toSlots(edited), [
    slot(1, "08:00", "12:00"),
    slot(1, "12:00", "13:00"),
    slot(3, "09:00", "18:00"),
    slot(6, "09:00", "22:00"),
  ]);
});

test("setDayOpen opens a day with the default range and closing it drops every range", () => {
  const rows = toDayHours([slot(1, "09:00", "13:00"), slot(1, "16:00", "20:00")]);

  assert.deepEqual(day(setDayOpen(rows, 2, true), 2).ranges, [range("09:00", "18:00")]);
  assert.deepEqual(day(setDayOpen(rows, 1, false), 1).ranges, []);
});

test("addRange starts where the last range ends so the new one does not overlap", () => {
  const rows = toDayHours([slot(1, "09:00", "13:00")]);

  const added = addRange(rows, 1);

  assert.deepEqual(day(added, 1).ranges, [range("09:00", "13:00"), range("13:00", "14:00")]);
  assert.equal(firstProblem(added), null);
});

test("addRange on a day too late to fit another hour repeats the default range to correct", () => {
  const rows = toDayHours([slot(1, "10:00", "23:00")]);

  assert.deepEqual(day(addRange(rows, 1), 1).ranges.at(-1), range("09:00", "18:00"));
});

test("removeRange and updateRange touch only the chosen range of the chosen day", () => {
  const rows = toDayHours([
    slot(1, "09:00", "13:00"),
    slot(1, "16:00", "20:00"),
    slot(2, "09:00", "13:00"),
  ]);

  const updated = updateRange(rows, 1, 1, { end: "21:00" });
  assert.deepEqual(day(updated, 1).ranges, [range("09:00", "13:00"), range("16:00", "21:00")]);
  assert.deepEqual(day(updated, 2).ranges, [range("09:00", "13:00")]);

  assert.deepEqual(day(removeRange(rows, 1, 0), 1).ranges, [range("16:00", "20:00")]);
});

test("copyDay replaces the ranges of the chosen days and leaves the rest alone", () => {
  const rows = toDayHours([
    slot(1, "09:00", "13:00"),
    slot(1, "16:00", "20:00"),
    slot(2, "08:00", "09:00"),
    slot(6, "10:00", "12:00"),
  ]);

  const copied = copyDay(rows, 1, [2, 3, 1]);

  assert.deepEqual(day(copied, 2).ranges, day(rows, 1).ranges);
  assert.deepEqual(day(copied, 3).ranges, day(rows, 1).ranges);
  assert.deepEqual(day(copied, 1).ranges, day(rows, 1).ranges);
  assert.deepEqual(day(copied, 6).ranges, [range("10:00", "12:00")]);
  assert.notEqual(day(copied, 2).ranges[0], day(copied, 1).ranges[0]);
});

test("copyDay of a closed day closes the targets", () => {
  const rows = toDayHours([slot(2, "08:00", "12:00")]);

  assert.deepEqual(day(copyDay(rows, 1, [2]), 2).ranges, []);
});

test("firstProblem points at the first day with a wrong or overlapping range", () => {
  const rows = toDayHours([slot(1, "08:00", "23:00")]);
  assert.equal(firstProblem(rows), null);

  const touching = toDayHours([slot(1, "09:00", "13:00"), slot(1, "13:00", "18:00")]);
  assert.equal(firstProblem(touching), null);

  const backwards = updateRange(setDayOpen(rows, 4, true), 4, 0, { start: "20:00", end: "10:00" });
  assert.deepEqual(firstProblem(backwards), { weekday: 4, kind: "invalid" });

  const empty = updateRange(setDayOpen(rows, 5, true), 5, 0, { start: "" });
  assert.deepEqual(firstProblem(empty), { weekday: 5, kind: "invalid" });

  const overlapping = toDayHours([slot(3, "09:00", "13:00"), slot(3, "12:00", "18:00")]);
  assert.deepEqual(firstProblem(overlapping), { weekday: 3, kind: "overlap" });

  assert.equal(firstProblem(toDayHours([])), null);
});
