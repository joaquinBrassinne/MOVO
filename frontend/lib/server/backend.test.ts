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

/** Whether the proxy forwards a browser call to the API (true) or answers 404 itself (false). */
async function reachesTheApi(method: string, path: string): Promise<boolean> {
  let called = false;
  globalThis.fetch = (async () => {
    called = true;
    return new Response("{}", { status: 200, headers: { "content-type": "application/json" } });
  }) as typeof fetch;
  const request = new Request(`http://localhost:3000/api/v1/${path}`, {
    method,
    ...(method === "GET" ? {} : { body: "{}", headers: { "content-type": "application/json" } }),
  });
  await proxyToBackend(request, path.split("/"));
  return called;
}

const COUPON_TOKEN = "a".repeat(43);

test("the proxy lets the customer's review routes through, with no session", async () => {
  assert.equal(await reachesTheApi("GET", `access/${COUPON_TOKEN}/review`), true);
  assert.equal(await reachesTheApi("POST", `access/${COUPON_TOKEN}/reviews`), true);
});

test("the proxy lets the public reviews of a branch through", async () => {
  assert.equal(await reachesTheApi("GET", "branches/12/reviews"), true);
  assert.equal(await reachesTheApi("GET", "branches/12/reviews/summary"), true);
});

test("the proxy still refuses what is next to the review routes", async () => {
  for (const path of [
    `access/${COUPON_TOKEN}/other`,
    "access/short-token/review",
    "access/short-token/reviews",
    `access/${COUPON_TOKEN}/review/extra`,
    "branches/abc/reviews",
    "branches/12/reviews/other",
    "branches/12/other",
    "branches/12",
  ]) {
    assert.equal(await reachesTheApi("GET", path), false, `${path} must not pass`);
  }
});

test("the proxy refuses a segment that the URL parser would read as something else", async () => {
  // fetch lee `\` como `/` y resuelve `..`: sin este filtro, me/<..\..\auth\refresh> llegaría a /auth/refresh.
  for (const path of [
    "me/\\..\\..\\auth\\refresh",
    "me/..\\x",
    "me/x?y=1",
    "me/x#y",
    "me/x y",
    "me/x;y",
    "admin/\\businesses",
  ]) {
    assert.equal(await reachesTheApi("GET", path), false, `${path} must not pass`);
  }
});

test("the proxy still passes the routes the app uses, with ids, slugs and tokens", async () => {
  for (const path of [
    "me/business/benefits/12/active",
    "admin/businesses/3",
    "businesses/gym-del-centro/view",
    "r/wa/12",
    `access/${COUPON_TOKEN}`,
    "search",
  ]) {
    assert.equal(await reachesTheApi("GET", path), true, `${path} must pass`);
  }
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
