import assert from "node:assert/strict";
import { test } from "node:test";

import { securityHeaders } from "./securityHeaders.ts";

const valuesOf = (production: boolean) =>
  Object.fromEntries(
    securityHeaders(production).flatMap((r) => r.headers.map((h) => [h.key, h.value])),
  );

test("every route gets the security headers", () => {
  const rules = securityHeaders(false);
  assert.equal(rules.length, 1);
  assert.equal(rules[0].source, "/:path*");
});

test("no page can be framed by another site", () => {
  const headers = valuesOf(false);
  assert.equal(headers["X-Frame-Options"], "DENY");
  assert.match(headers["Content-Security-Policy"], /frame-ancestors 'none'/);
});

test("the CSP closes plugins and base tag injection without restricting scripts", () => {
  const csp = valuesOf(false)["Content-Security-Policy"];
  assert.match(csp, /object-src 'none'/);
  assert.match(csp, /base-uri 'self'/);
  assert.match(csp, /form-action 'self'/);
  // Next's inline bootstrap scripts carry no nonce yet: a script-src would break the app.
  assert.doesNotMatch(csp, /script-src/);
});

test("the browser may not sniff types or leak the coupon URL to other sites", () => {
  const headers = valuesOf(false);
  assert.equal(headers["X-Content-Type-Options"], "nosniff");
  assert.equal(headers["Referrer-Policy"], "strict-origin-when-cross-origin");
});

test("the QR scanner keeps the camera and the map keeps geolocation, for this origin only", () => {
  const policy = valuesOf(false)["Permissions-Policy"];
  assert.match(policy, /camera=\(self\)/);
  assert.match(policy, /geolocation=\(self\)/);
  assert.match(policy, /microphone=\(\)/);
});

test("HSTS is sent in production only", () => {
  assert.equal(valuesOf(false)["Strict-Transport-Security"], undefined);
  assert.match(valuesOf(true)["Strict-Transport-Security"], /^max-age=\d+$/);
});
