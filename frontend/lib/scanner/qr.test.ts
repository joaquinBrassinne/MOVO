import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";

import { encode } from "uqr";
import { prepareZXingModule, readBarcodes } from "zxing-wasm/reader";

import { qrShape } from "./qr.ts";

/** Dibuja el trazo en una grilla de `size` × `size` para compararlo con la matriz original. */
function rasterize(path: string, size: number): boolean[][] {
  const grid = Array.from({ length: size }, () => Array<boolean>(size).fill(false));
  for (const [, x, y, width] of path.matchAll(/M(\d+) (\d+)h(\d+)v1h-\d+z/g)) {
    for (let i = 0; i < Number(width); i++) grid[Number(y)][Number(x) + i] = true;
  }
  return grid;
}

test("the path draws exactly the dark modules of the QR", () => {
  const { size, path } = qrShape("ABCDEFGH");
  const { data, size: expectedSize } = encode("ABCDEFGH", { ecc: "M", border: 2 });

  assert.equal(size, expectedSize);
  assert.deepEqual(rasterize(path, size), data);
});

test("the QR of a coupon is read back as exactly its code and nothing else", async () => {
  // El mismo lector y el mismo WASM que usa el escáner del panel.
  prepareZXingModule({
    overrides: {
      wasmBinary: readFileSync(new URL("../../public/zxing/zxing_reader.wasm", import.meta.url)),
    },
    fireImmediately: true,
  });

  for (const code of ["ABCDEFGH", "ALMA2345", "ZZZZ9999"]) {
    const { size, path } = qrShape(code);
    const scale = 8;
    const side = size * scale;
    const pixels = new Uint8ClampedArray(side * side * 4).fill(255);
    const grid = rasterize(path, size);
    for (let y = 0; y < side; y++) {
      for (let x = 0; x < side; x++) {
        if (!grid[Math.floor(y / scale)][Math.floor(x / scale)]) continue;
        const at = (y * side + x) * 4;
        pixels[at] = pixels[at + 1] = pixels[at + 2] = 0;
      }
    }

    const found = await readBarcodes(
      { data: pixels, width: side, height: side, colorSpace: "srgb" },
      { formats: ["QRCode"] },
    );
    assert.deepEqual(
      found.map((item) => item.text),
      [code],
    );
  }
});
