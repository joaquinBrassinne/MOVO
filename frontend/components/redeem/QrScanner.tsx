"use client";

import { useEffect, useEffectEvent, useRef, useState } from "react";

import { Notice } from "@/components/onboarding/fields";
import { normalizeCode } from "@/lib/benefits/code";
import { cameraProblemOf, type CameraProblem } from "@/lib/scanner/camera";
import { loadQrReader, type QrReader } from "@/lib/scanner/detector";

interface Props {
  /** El código del cupón que se leyó, ya normalizado. */
  onCode: (code: string) => void;
  onClose: () => void;
}

const SCAN_EVERY_MS = 250;

type Status = "starting" | "scanning" | "problem";

/**
 * El escáner de QR del panel: abre la cámara, lee el código del cupón y lo entrega. Se
 * carga solo cuando el dueño toca "Escanear QR". Leer no canjea nada: el cupón se ve y se
 * confirma como con el ingreso manual, que sigue a la vista.
 */
export default function QrScanner({ onCode, onClose }: Props) {
  const video = useRef<HTMLVideoElement>(null);
  const [status, setStatus] = useState<Status>("starting");
  const [problem, setProblem] = useState<CameraProblem | null>(null);
  const [hint, setHint] = useState("");
  const deliver = useEffectEvent(onCode);

  useEffect(() => {
    let stopped = false;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let stream: MediaStream | undefined;

    function fail(error: unknown) {
      if (stopped) return;
      setProblem(cameraProblemOf(error));
      setStatus("problem");
    }

    async function start() {
      if (!navigator.mediaDevices?.getUserMedia) return fail(undefined);

      try {
        stream = await navigator.mediaDevices.getUserMedia({
          video: { facingMode: { ideal: "environment" } },
          audio: false,
        });
      } catch (error) {
        return fail(error);
      }
      const element = video.current;
      if (stopped || !element) return;

      element.srcObject = stream;
      try {
        await element.play();
      } catch (error) {
        return fail(error);
      }

      let reader: QrReader;
      try {
        reader = await loadQrReader();
      } catch (error) {
        return fail(error);
      }
      if (stopped) return;
      setStatus("scanning");

      async function tick() {
        if (stopped || !element) return;
        try {
          const text = await reader.read(element);
          if (text) {
            const code = normalizeCode(text);
            if (code) {
              stopped = true;
              deliver(code);
              return;
            }
            setHint("Ese QR no es de un cupón de MOVO. Probá con otro.");
          }
        } catch {
          // Un cuadro que no se pudo leer no es un error: se prueba con el siguiente.
        }
        timer = setTimeout(() => void tick(), SCAN_EVERY_MS);
      }
      void tick();
    }

    void start();

    return () => {
      stopped = true;
      clearTimeout(timer);
      stream?.getTracks().forEach((track) => track.stop());
    };
  }, []);

  return (
    <section aria-label="Escáner de QR" className="space-y-3 rounded-3xl bg-white p-5 shadow-sm">
      {status === "problem" && problem ? (
        <Notice title={problem.title} body={problem.body} />
      ) : (
        <>
          <div className="relative overflow-hidden rounded-3xl bg-black">
            <video
              ref={video}
              muted
              playsInline
              aria-label="Cámara"
              className="aspect-square w-full object-cover"
            />
            <div
              aria-hidden="true"
              className="pointer-events-none absolute inset-8 rounded-3xl border-4 border-movo-accent/80"
            />
          </div>
          <p role="status" className="text-center text-sm font-semibold">
            {hint ||
              (status === "starting"
                ? "Abriendo la cámara…"
                : "Apuntá la cámara al QR del cupón de tu cliente.")}
          </p>
        </>
      )}
      <button
        type="button"
        onClick={onClose}
        className="h-12 w-full rounded-full bg-movo-sand font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
      >
        {status === "problem" ? "Cerrar" : "Cancelar"}
      </button>
    </section>
  );
}
