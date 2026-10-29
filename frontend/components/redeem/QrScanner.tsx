"use client";

import { useEffect, useEffectEvent, useRef, useState } from "react";

import { QrIcon } from "@/components/benefits/icons";
import { Notice } from "@/components/onboarding/fields";
import { Svg } from "@/components/socios/icons";
import { normalizeCode } from "@/lib/benefits/code";
import { cameraProblemOf, type CameraProblem } from "@/lib/scanner/camera";
import { loadQrReader, type QrReader } from "@/lib/scanner/detector";

interface Props {
  /** El código del cupón que se leyó, ya normalizado. */
  onCode: (code: string) => void;
  onClose: () => void;
  /** Cierra el escáner y deja el ingreso manual del código listo para escribir. */
  onManual: () => void;
  /** La sede donde se canjea, si ya se sabe. */
  branchName?: string;
}

const STARTING = "Abriendo la cámara…";
const POINT = "Apuntá la cámara al QR del cupón de tu cliente.";

const BoltIcon = ({ className }: { className?: string }) => (
  <Svg className={className}>
    <path d="M13 2 3 14h9l-1 8 10-12h-9z" />
  </Svg>
);

// La linterna no está en los tipos de DOM: se detecta por capacidades y se oculta si falta.
type TorchCapabilities = MediaTrackCapabilities & { torch?: boolean };

const SCAN_EVERY_MS = 250;

type Status = "starting" | "scanning" | "problem";

/**
 * El escáner de QR del panel: abre la cámara, lee el código del cupón y lo entrega. Se
 * carga solo cuando el dueño toca "Escanear QR". Leer no canjea nada: el cupón se ve y se
 * confirma como con el ingreso manual, que sigue a la vista.
 */
export default function QrScanner({ onCode, onClose, onManual, branchName }: Props) {
  const video = useRef<HTMLVideoElement>(null);
  const track = useRef<MediaStreamTrack | null>(null);
  const [status, setStatus] = useState<Status>("starting");
  const [problem, setProblem] = useState<CameraProblem | null>(null);
  const [hint, setHint] = useState("");
  const [torchAvailable, setTorchAvailable] = useState(false);
  const [torchOn, setTorchOn] = useState(false);
  const deliver = useEffectEvent(onCode);

  async function toggleTorch() {
    const next = !torchOn;
    try {
      await track.current?.applyConstraints({
        advanced: [{ torch: next } as MediaTrackConstraintSet],
      });
      setTorchOn(next);
    } catch {
      // Si el dispositivo la rechaza, se oculta el botón y el escaneo sigue igual.
      setTorchAvailable(false);
    }
  }

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

      const videoTrack = stream.getVideoTracks()[0];
      const capabilities = videoTrack?.getCapabilities?.() as TorchCapabilities | undefined;
      if (videoTrack && capabilities?.torch) {
        track.current = videoTrack;
        setTorchAvailable(true);
      }

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
    <section
      role="dialog"
      aria-modal="true"
      aria-label="Escáner de QR"
      className="fixed inset-0 z-50 flex flex-col overflow-hidden bg-movo-tabs text-white"
    >
      {status !== "problem" ? (
        <video
          ref={video}
          muted
          playsInline
          aria-label="Cámara"
          className="absolute inset-0 size-full object-cover opacity-70"
        />
      ) : null}
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 bg-linear-to-b from-movo-tabs/80 via-transparent to-movo-tabs/95"
      />

      <header className="relative flex items-center gap-3 p-4">
        <span
          aria-hidden="true"
          className="size-2.5 shrink-0 rounded-full bg-movo-accent motion-safe:animate-pulse"
        />
        <div className="min-w-0 flex-1">
          <h2 className="font-display text-lg leading-tight font-extrabold">Escáner MOVO</h2>
          <p className="text-xs text-movo-muted">Recepción · En vivo</p>
        </div>
        {torchAvailable ? (
          <button
            type="button"
            onClick={() => void toggleTorch()}
            aria-pressed={torchOn}
            aria-label="Linterna"
            className={`grid size-10 place-items-center rounded-full outline-none focus-visible:ring-2 focus-visible:ring-movo-accent ${torchOn ? "bg-movo-accent text-movo-ink" : "bg-movo-card/80 text-white"}`}
          >
            <BoltIcon className="size-5" />
          </button>
        ) : null}
        <span
          aria-hidden="true"
          className="grid size-10 place-items-center rounded-full bg-movo-card/80 text-white"
        >
          <QrIcon className="size-5" />
        </span>
      </header>

      <div className="relative flex flex-1 items-center justify-center px-8">
        {status === "problem" && problem ? (
          <div className="w-full max-w-sm rounded-2xl bg-white">
            <Notice title={problem.title} body={problem.body} />
          </div>
        ) : (
          <div
            aria-hidden="true"
            className="relative aspect-square w-full max-w-xs rounded-3xl border-4 border-movo-accent/90"
          >
            <span className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full bg-movo-tabs/80 px-3 py-1 text-xs font-semibold whitespace-nowrap text-movo-accent">
              Apuntá al QR del cliente
            </span>
          </div>
        )}
      </div>

      <div className="relative space-y-4 p-5 pb-8 text-center">
        <div>
          <h3 className="font-display text-2xl font-extrabold">Enfocá el pase</h3>
          <p className="mt-1 text-sm text-movo-muted">
            Mantené el código dentro del recuadro para validar el turno al instante sin demoras.
          </p>
          <p role="status" className="mt-2 min-h-5 text-sm font-semibold text-movo-accent">
            {status === "problem" ? null : hint || (status === "starting" ? STARTING : POINT)}
          </p>
        </div>
        {branchName ? (
          <p className="mx-auto w-fit max-w-full truncate rounded-full bg-movo-card/80 px-3 py-1 text-xs font-semibold">
            {branchName}
          </p>
        ) : null}
        <div className="space-y-2">
          <button
            type="button"
            onClick={onManual}
            className="h-12 w-full rounded-full bg-movo-card font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
          >
            Ingresar código manual
          </button>
          <button
            type="button"
            onClick={onClose}
            autoFocus
            className="h-12 w-full rounded-full bg-movo-accent font-bold text-movo-ink outline-none focus-visible:ring-2 focus-visible:ring-white"
          >
            Cerrar escáner
          </button>
        </div>
      </div>
    </section>
  );
}
