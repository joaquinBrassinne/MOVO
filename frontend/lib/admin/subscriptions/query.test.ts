import assert from "node:assert/strict";
import { test } from "node:test";

import { hasActiveFilters, NO_FILTERS, pageRange, subscriptionsQuery } from "./query.ts";

test("the query string always pages and leaves empty filters out", () => {
  assert.equal(subscriptionsQuery(NO_FILTERS), "limit=20&offset=0");
});

test("every filter is sent under the name of the API and the offset follows the page", () => {
  const query = subscriptionsQuery({
    status: "expiring",
    plan: "pro",
    query: "  Alma Haus ",
    page: 3,
  });
  assert.deepEqual(Object.fromEntries(new URLSearchParams(query)), {
    limit: "20",
    offset: "40",
    status: "expiring",
    plan: "pro",
    q: "Alma Haus",
  });
});

test("a blank search is not a filter", () => {
  assert.equal(subscriptionsQuery({ ...NO_FILTERS, query: "   " }), "limit=20&offset=0");
  assert.equal(hasActiveFilters({ ...NO_FILTERS, query: "  ", page: 4 }), false);
  assert.equal(hasActiveFilters({ ...NO_FILTERS, plan: "starter" }), true);
});

test("the page range is 1-based and clamped to the total", () => {
  assert.deepEqual(pageRange(1, 87), { from: 1, to: 20 });
  assert.deepEqual(pageRange(3, 45), { from: 41, to: 45 });
  assert.deepEqual(pageRange(1, 0), { from: 0, to: 0 });
});
