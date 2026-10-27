import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { test } from "node:test";

import { locateWasm, WASM_BASE_PATH } from "./detector.ts";

test("the wasm is looked up in our own origin, never in a CDN", () => {
  assert.equal(
    locateWasm("zxing_reader.wasm", "https://fastly.jsdelivr.net/npm/zxing-wasm/"),
    "/zxing/zxing_reader.wasm",
  );
  assert.equal(WASM_BASE_PATH, "/zxing/");
  assert.doesNotMatch(locateWasm("zxing_reader.wasm", "https://cdn.example/"), /^https?:/);
});

test("any other file keeps the prefix the library chose", () => {
  assert.equal(locateWasm("helper.js", "/base/"), "/base/helper.js");
});

test("the copy of the wasm in public/ is the one the installed library expects", async () => {
  // Si se actualiza barcode-detector, hay que volver a copiar node_modules/zxing-wasm/dist/reader/zxing_reader.wasm.
  const { ZXING_WASM_SHA256 } = await import("barcode-detector/ponyfill");
  const copy = readFileSync(new URL("../../public/zxing/zxing_reader.wasm", import.meta.url));

  assert.equal(createHash("sha256").update(copy).digest("hex"), ZXING_WASM_SHA256);
});
