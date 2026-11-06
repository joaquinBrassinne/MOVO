import assert from "node:assert/strict";
import { test } from "node:test";

import { describeOpenNow, groupHours, weekdayInCordoba } from "./profile.ts";

const slot = (weekday: number, start: string, end: string) => ({ weekday, start, end });
const weekdays = (start: string, end: string, days = [1, 2, 3, 4, 5]) =>
  days.map((day) => slot(day, start, end));

test("groupHours joins consecutive days with the same ranges, split shifts included", () => {
  const groups = groupHours([
    ...weekdays("08:00", "12:00"),
    ...weekdays("14:00", "22:00"),
    slot(6, "09:00", "13:00"),
  ]);

  assert.deepEqual(groups, [
    { label: "Lunes a Viernes", ranges: ["08:00 – 12:00", "14:00 – 22:00"] },
    { label: "Sábados", ranges: ["09:00 – 13:00"] },
    { label: "Domingos", ranges: [] },
  ]);
});

test("groupHours names two consecutive days with y and a lone weekday as it is", () => {
  const groups = groupHours([
    slot(1, "09:00", "18:00"),
    slot(2, "09:00", "18:00"),
    slot(4, "10:00", "12:00"),
  ]);

  assert.deepEqual(
    groups.map((g) => g.label),
    ["Lunes y Martes", "Miércoles", "Jueves", "Viernes a Domingo"],
  );
  assert.deepEqual(groups[3].ranges, []);
});

test("groupHours of nothing is every day closed in one group", () => {
  assert.deepEqual(groupHours([]), [{ label: "Lunes a Domingo", ranges: [] }]);
});

test("describeOpenNow says when the branch closes while it is open", () => {
  assert.equal(
    describeOpenNow({ open: true, closes_at: "12:00", opens_at: null, opens_weekday: null }, 1),
    "Abierto ahora · cierra a las 12:00",
  );
});

test("describeOpenNow says when it opens: today, another day, or never", () => {
  const closed = { open: false, closes_at: null };
  assert.equal(
    describeOpenNow({ ...closed, opens_at: "14:00", opens_weekday: 1 }, 1),
    "Cerrado · abre a las 14:00",
  );
  assert.equal(
    describeOpenNow({ ...closed, opens_at: "09:00", opens_weekday: 3 }, 1),
    "Cerrado · abre el miércoles a las 09:00",
  );
  assert.equal(describeOpenNow({ ...closed, opens_at: null, opens_weekday: null }, 1), "Cerrado");
});

test("weekdayInCordoba reads the day in Argentina, not in UTC", () => {
  // Domingo 25/10/2026 23:00 en Córdoba ya es lunes 02:00 en UTC.
  assert.equal(weekdayInCordoba(new Date("2026-10-26T02:00:00Z")), 0);
  assert.equal(weekdayInCordoba(new Date("2026-10-26T15:00:00Z")), 1);
});
