import assert from "node:assert/strict";
import { test } from "node:test";

import {
  buildResetLink,
  checkPasswords,
  classifyResetFailure,
  describeIssueFailure,
  resetWhatsAppUrl,
} from "./rules.ts";

const failure = (code: string, status = 400) => ({ status, error: { code } });

test("the checklist needs the minimum length and matching fields", () => {
  assert.equal(checkPasswords("short", "short").valid, false);
  assert.equal(checkPasswords("longenough1", "different11").matches, false);
  assert.deepEqual(checkPasswords("longenough1", "longenough1"), {
    longEnough: true,
    tooLong: false,
    matches: true,
    valid: true,
  });
});

test("an empty pair does not count as matching", () => {
  assert.equal(checkPasswords("", "").matches, false);
});

test("a password over the maximum is rejected", () => {
  const long = "a".repeat(129);
  assert.equal(checkPasswords(long, long).valid, false);
});

test("the reset link carries the encoded token", () => {
  assert.equal(
    buildResetLink("https://movo.test", "a+b/c"),
    "https://movo.test/owners/reset-password?token=a%2Bb%2Fc",
  );
});

test("the WhatsApp link keeps only digits and embeds the reset link", () => {
  const url = resetWhatsAppUrl("+54 9 11 5555-0000", "https://movo.test/x?token=t");
  assert.ok(url?.startsWith("https://wa.me/5491155550000?text="));
  assert.ok(url?.includes(encodeURIComponent("https://movo.test/x?token=t")));
  assert.equal(resetWhatsAppUrl("", "x"), null);
});

test("an unknown owner email is explained and server text is never shown", () => {
  assert.match(describeIssueFailure(failure("user_not_found", 404)).title, /No encontramos/);
  assert.doesNotMatch(describeIssueFailure(failure("boom", 500)).body, /boom/);
});

test("reset failures map to an outcome", () => {
  assert.equal(classifyResetFailure(failure("invalid_reset_token")), "invalid_token");
  assert.equal(classifyResetFailure(failure("too_many_attempts", 429)), "too_many");
  assert.equal(classifyResetFailure(failure("validation_failed", 422)), "password");
  assert.equal(classifyResetFailure(failure("network_error", 0)), "generic");
});
