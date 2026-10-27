import assert from "node:assert/strict";
import { afterEach, test } from "node:test";

import { postToBackend, proxyToBackend } from "./backend.ts";

const realFetch = globalThis.fetch;

/** Reemplaza fetch y devuelve las cabeceras con las que Next llamó a la API. */
function captureUpstream(): { sent: () => Headers } {
  let headers = new Headers();
  globalThis.fetch = (async (_url: unknown, init?: RequestInit) => {
    headers = new Headers(init?.headers);
    return new Response("{}", { status: 200, headers: { "content-type": "application/json" } });
  }) as typeof fetch;
  return { sent: () => headers };
}

afterEach(() => {
  globalThis.fetch = realFetch;
});

const browser = (forwarded?: string) =>
  new Request("http://localhost:3000/api/v1/benefits/7/codes", {
    method: "POST",
    headers: {
      "content-type": "application/json",
      ...(forwarded ? { "x-forwarded-for": forwarded } : {}),
    },
    body: "{}",
  });

test("the proxy passes the visitor's chain on without replacing it", async () => {
  const upstream = captureUpstream();

  await proxyToBackend(browser("198.51.100.7, 192.0.2.10"), ["benefits", "7", "codes"]);

  assert.equal(upstream.sent().get("x-forwarded-for"), "198.51.100.7, 192.0.2.10");
});

test("the proxy sends no X-Forwarded-For when the request had none", async () => {
  const upstream = captureUpstream();

  await proxyToBackend(browser(), ["benefits", "7", "codes"]);

  assert.equal(upstream.sent().get("x-forwarded-for"), null);
});

test("the session calls carry the visitor's chain too, so the login limit counts the visitor", async () => {
  const upstream = captureUpstream();

  await postToBackend(browser("198.51.100.7"), "/auth/login", { email: "a@b.c", password: "x" });

  assert.equal(upstream.sent().get("x-forwarded-for"), "198.51.100.7");
  assert.equal(upstream.sent().get("content-type"), "application/json");
});

test("the session calls send no chain when the request had none", async () => {
  const upstream = captureUpstream();

  await postToBackend(browser(), "/auth/login", { email: "a@b.c", password: "x" });

  assert.equal(upstream.sent().get("x-forwarded-for"), null);
});
