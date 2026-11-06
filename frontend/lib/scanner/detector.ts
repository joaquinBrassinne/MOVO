// El lector de QR del escáner. La librería y su módulo WASM pesan bastante, por eso solo se
// cargan (import dinámico) cuando el dueño toca "Escanear QR".

/**
 * De dónde se baja el WASM: de nuestro propio origen (public/zxing), nunca de un CDN. Por
 * defecto zxing-wasm lo busca en jsDelivr; `locateWasm` lo redirige.
 */
export const WASM_BASE_PATH = "/zxing/";

/** Dónde buscar cada archivo que la librería pide: el `.wasm` va a nuestro origen. */
export function locateWasm(path: string, prefix: string): string {
  return path.endsWith(".wasm") ? `${WASM_BASE_PATH}${path}` : prefix + path;
}

export interface QrReader {
  /** El texto del primer QR que se ve en el cuadro actual del video, o `null` si no hay. */
  read(video: HTMLVideoElement): Promise<string | null>;
}

export async function loadQrReader(): Promise<QrReader> {
  // El ponyfill no toca el `BarcodeDetector` global: todos los navegadores leen igual y el
  // WASM sale siempre de `WASM_BASE_PATH`.
  const { BarcodeDetector, prepareZXingModule } = await import("barcode-detector/ponyfill");
  prepareZXingModule({ overrides: { locateFile: locateWasm }, fireImmediately: true });

  const detector = new BarcodeDetector({ formats: ["qr_code"] });
  return {
    async read(video) {
      const found = await detector.detect(video);
      return found[0]?.rawValue ?? null;
    },
  };
}
