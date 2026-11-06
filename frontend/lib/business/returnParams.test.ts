import assert from "node:assert/strict";
import { test } from "node:test";

import { parseReturnParams } from "./returnParams.ts";

test("parseReturnParams keeps what Mercado Pago appended, as a hint", () => {
  assert.deepEqual(parseReturnParams("?plan=pro&preapproval_plan_id=2c93"), {
    plan: "pro",
    preapproval_plan_id: "2c93",
  });
});

test("parseReturnParams drops empty and oversized entries instead of failing", () => {
  assert.deepEqual(parseReturnParams(`?plan=&=x&long=${"v".repeat(201)}&ok=1`), { ok: "1" });
  assert.deepEqual(parseReturnParams(""), {});
});

test("parseReturnParams keeps at most ten parameters", () => {
  const search = Array.from({ length: 15 }, (_, i) => `k${i}=v`).join("&");
  assert.equal(Object.keys(parseReturnParams(search)).length, 10);
});
