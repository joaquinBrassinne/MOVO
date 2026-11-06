import assert from "node:assert/strict";
import { test } from "node:test";

import { jsonLd } from "./jsonld.ts";

test("jsonLd cannot close the script block", () => {
  const out = jsonLd({ name: "</script><script>alert(1)</script>" });
  assert.ok(!out.includes("<"));
  assert.ok(!out.includes(">"));
});

test("jsonLd round-trips the original data", () => {
  const data = { name: "a < b & c > d ", n: 1 };
  assert.deepEqual(JSON.parse(jsonLd(data)), data);
});
