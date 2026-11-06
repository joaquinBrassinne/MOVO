import assert from "node:assert/strict";
import { test } from "node:test";

import { fromWhatsApp, localDigits, toWhatsApp } from "./phone.ts";

test("toWhatsApp builds the wa.me number from what the owner types", () => {
  assert.equal(toWhatsApp("11 5824 9912"), "5491158249912");
  assert.equal(toWhatsApp("11-5824-9912"), "5491158249912");
  assert.equal(toWhatsApp("(11) 5824.9912"), "5491158249912");
  assert.equal(toWhatsApp("0351 555 1234"), "5493515551234");
});

test("toWhatsApp refuses numbers that are not 10 digits", () => {
  assert.equal(toWhatsApp(""), null);
  assert.equal(toWhatsApp("5824 9912"), null);
  assert.equal(toWhatsApp("11 15 5824 9912"), null);
  assert.equal(toWhatsApp("abc"), null);
});

test("localDigits drops separators and the leading zero only", () => {
  assert.equal(localDigits(" 011 5824-9912 "), "1158249912");
  assert.equal(localDigits("1105"), "1105");
});

test("fromWhatsApp shows a stored number the way it is edited", () => {
  assert.equal(fromWhatsApp("5491158249912"), "11 5824 9912");
});

test("fromWhatsApp does not invent a number for other formats", () => {
  assert.equal(fromWhatsApp("34612345678"), "");
  assert.equal(fromWhatsApp(""), "");
  assert.equal(fromWhatsApp("549115"), "115");
});

test("a number survives the round trip", () => {
  assert.equal(toWhatsApp(fromWhatsApp("5493515551234")), "5493515551234");
});
