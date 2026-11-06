// Una cámara falsa para los E2E del escáner: Chromium lee un video y lo ofrece como si fuera
// la cámara. Acá se arma ese video (formato y4m, sin comprimir) con un QR de un texto dado.
// Chromium abre el archivo cada vez que alguien pide la cámara, así que cada test puede
// reescribirlo con el QR que necesita antes de tocar "Escanear QR".

import { mkdirSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";

import { encode } from "uqr";

export const FAKE_CAMERA_FILE = join(tmpdir(), "movo-e2e-camera", "qr.y4m");

const WIDTH = 640;
const HEIGHT = 480;
const FRAMES = 10;

/** Escribe un video de `FRAMES` cuadros iguales con el QR de `text`, negro sobre blanco. */
export function writeQrVideo(text, file = FAKE_CAMERA_FILE) {
  const { data, size } = encode(text, { ecc: "M", border: 4 });
  const scale = Math.floor(HEIGHT / size);
  const offsetX = Math.floor((WIDTH - size * scale) / 2);
  const offsetY = Math.floor((HEIGHT - size * scale) / 2);

  // Plano de luminancia: 255 es blanco y 0 negro. Los de color (4:2:0) quedan en gris neutro.
  const luma = Buffer.alloc(WIDTH * HEIGHT, 255);
  for (let y = 0; y < size * scale; y++) {
    for (let x = 0; x < size * scale; x++) {
      if (data[Math.floor(y / scale)][Math.floor(x / scale)]) {
        luma[(offsetY + y) * WIDTH + offsetX + x] = 0;
      }
    }
  }
  const chroma = Buffer.alloc((WIDTH / 2) * (HEIGHT / 2), 128);

  const header = Buffer.from(`YUV4MPEG2 W${WIDTH} H${HEIGHT} F30:1 Ip A1:1 C420jpeg\n`);
  const frame = Buffer.concat([Buffer.from("FRAME\n"), luma, chroma, chroma]);

  mkdirSync(dirname(file), { recursive: true });
  writeFileSync(file, Buffer.concat([header, ...Array.from({ length: FRAMES }, () => frame)]));
}
