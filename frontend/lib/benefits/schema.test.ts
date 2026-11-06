import assert from "node:assert/strict";
import { test } from "node:test";

import { parseBenefitForm, parseQuotaForm, type BenefitFormValues } from "./schema.ts";

const empty: BenefitFormValues = {
  type: "percent",
  percent: "",
  amount: "",
  title: "",
  conditions: "",
};

function fields(values: Partial<BenefitFormValues>): Record<string, string> {
  const result = parseBenefitForm({ ...empty, ...values });
  assert.equal(result.ok, false);
  return result.ok ? {} : result.fields;
}

test("a percent is any whole number from 1 to 100, not only the shortcuts", () => {
  for (const percent of ["1", "10", "15", "37", "100"]) {
    const result = parseBenefitForm({ ...empty, percent });
    assert.deepEqual(result, { ok: true, input: { type: "percent", percent: Number(percent) } });
  }
  for (const percent of ["", "0", "101", "-5", "12.5", "abc"]) {
    assert.ok(fields({ percent }).percent, `"${percent}" must be refused`);
  }
});

test("an amount is a positive whole number of pesos, written however the owner likes", () => {
  assert.deepEqual(parseBenefitForm({ ...empty, type: "amount", amount: "$ 2.000" }), {
    ok: true,
    input: { type: "amount", amount_ars: 2000 },
  });
  for (const amount of ["", "0", "abc", "12,50"]) {
    assert.ok(fields({ type: "amount", amount }).amount_ars, `"${amount}" must be refused`);
  }
});

test("a custom benefit needs a title of up to 80 characters", () => {
  assert.deepEqual(parseBenefitForm({ ...empty, type: "custom", title: "  Una clase gratis " }), {
    ok: true,
    input: { type: "custom", title: "Una clase gratis" },
  });
  assert.ok(fields({ type: "custom", title: "   " }).title);
  assert.ok(fields({ type: "custom", title: "a".repeat(81) }).title);
  assert.equal(parseBenefitForm({ ...empty, type: "custom", title: "ñ".repeat(80) }).ok, true);
});

test("the conditions are optional, up to 120 characters, and sent only when written", () => {
  const withConditions = parseBenefitForm({ ...empty, percent: "15", conditions: " Solo lunes " });
  assert.deepEqual(withConditions, {
    ok: true,
    input: { type: "percent", percent: 15, conditions: "Solo lunes" },
  });
  assert.ok(fields({ percent: "15", conditions: "a".repeat(121) }).conditions);
});

test("links and phone numbers are refused in the free texts, like the backend does", () => {
  const link = "No incluyas links";
  assert.equal(fields({ type: "custom", title: "mirá https://promo.example" }).title, link);
  assert.equal(fields({ type: "custom", title: "entrá a www.promo" }).title, link);
  assert.equal(fields({ percent: "10", conditions: "promo.com.ar" }).conditions, link);
  assert.equal(fields({ percent: "10", conditions: "más en wa.me/549111" }).conditions, link);

  const phone = "No incluyas teléfonos";
  assert.equal(fields({ type: "custom", title: "Llamá al 11 4567-8901" }).title, phone);
  assert.equal(fields({ percent: "10", conditions: "+54 9 351 123 4567" }).conditions, phone);
});

test("harmless texts that look a little like links or numbers are accepted", () => {
  for (const conditions of [
    "2x1 en clases de lunes a viernes",
    "Válido de 18.30 a 20 hs",
    "Compra mínima $10.000",
    "Hasta agotar stock. Comprando hoy",
  ]) {
    assert.equal(parseBenefitForm({ ...empty, percent: "10", conditions }).ok, true, conditions);
  }
});

test("the fields of the other types are not sent", () => {
  const result = parseBenefitForm({ ...empty, percent: "10", amount: "500", title: "x" });
  assert.deepEqual(result, { ok: true, input: { type: "percent", percent: 10 } });
});

test("the quota is a whole number above zero, or no limit at all", () => {
  assert.deepEqual(parseQuotaForm(true, "", 99), { ok: true, limit: null });
  assert.deepEqual(parseQuotaForm(false, "30", 18), { ok: true, limit: 30 });
  assert.deepEqual(parseQuotaForm(false, "18", 18), { ok: true, limit: 18 });
  for (const text of ["", "0", "-1", "1.5", "mucho"]) {
    assert.equal(parseQuotaForm(false, text, 0).ok, false, `"${text}" must be refused`);
  }
});

test("a quota below what was already generated is refused with that number", () => {
  const result = parseQuotaForm(false, "10", 18);
  assert.equal(result.ok, false);
  assert.match(result.ok ? "" : result.message, /18 cupones/);
});
