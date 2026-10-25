import assert from "node:assert/strict";
import { test } from "node:test";

import { copyMondayToWeekdays, firstInvalidDay, toDayHours, toSlots } from "./hours.ts";

const slot = (weekday: number, start: string, end: string) => ({ weekday, start, end });

test("toDayHours gives the seven days in week order, closed unless a slot opens them", () => {
  const rows = toDayHours([slot(1, "08:00", "23:00"), slot(0, "10:00", "14:00")]);

  assert.ok(rows);
  assert.deepEqual(
    rows.map((row) => row.weekday),
    [1, 2, 3, 4, 5, 6, 0],
  );
  assert.deepEqual(rows[0], { weekday: 1, open: true, start: "08:00", end: "23:00" });
  assert.equal(rows[1].open, false);
  assert.deepEqual(rows[6], { weekday: 0, open: true, start: "10:00", end: "14:00" });
});

test("toDayHours of nothing is a closed week with a sensible default range to open a day", () => {
  const rows = toDayHours([]);

  assert.ok(rows);
  assert.equal(rows.length, 7);
  assert.ok(rows.every((row) => !row.open && row.start < row.end));
});

test("toDayHours refuses a day with split shifts because the form has one range per day", () => {
  assert.equal(toDayHours([slot(1, "09:00", "13:00"), slot(1, "16:00", "20:00")]), null);
});

test("toSlots keeps only the open days, in week order", () => {
  const rows = toDayHours([slot(6, "09:00", "22:00"), slot(1, "08:00", "23:00")]);
  assert.ok(rows);
  rows[2] = { ...rows[2], open: true, start: "10:00", end: "12:00" };

  assert.deepEqual(toSlots(rows), [
    slot(1, "08:00", "23:00"),
    slot(3, "10:00", "12:00"),
    slot(6, "09:00", "22:00"),
  ]);
});

test("copyMondayToWeekdays repeats Monday on Tuesday to Friday and leaves the weekend alone", () => {
  const rows = toDayHours([slot(1, "08:00", "23:00"), slot(6, "09:00", "22:00")]);
  assert.ok(rows);

  const copied = copyMondayToWeekdays(rows);

  for (const row of copied.slice(0, 5)) {
    assert.deepEqual([row.open, row.start, row.end], [true, "08:00", "23:00"]);
  }
  assert.deepEqual(copied[5], rows[5]);
  assert.deepEqual(copied[6], rows[6]);
});

test("copyMondayToWeekdays with Monday closed closes the weekdays", () => {
  const rows = toDayHours([slot(2, "08:00", "12:00")]);
  assert.ok(rows);

  assert.ok(
    copyMondayToWeekdays(rows)
      .slice(0, 5)
      .every((row) => !row.open),
  );
});

test("firstInvalidDay points at the first open day whose range is wrong", () => {
  const rows = toDayHours([slot(1, "08:00", "23:00")]);
  assert.ok(rows);
  assert.equal(firstInvalidDay(rows), null);

  rows[3] = { ...rows[3], open: true, start: "20:00", end: "10:00" };
  rows[4] = { ...rows[4], open: true, start: "", end: "10:00" };
  assert.equal(firstInvalidDay(rows), 4);
  assert.equal(firstInvalidDay(toDayHours([]) ?? []), null);
});
