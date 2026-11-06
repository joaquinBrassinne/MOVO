"use client";

import dynamic from "next/dynamic";
import { useState, type SubmitEvent } from "react";

import { QrIcon } from "@/components/benefits/icons";
import { Field, Notice } from "@/components/onboarding/fields";
import { CheckCircleIcon } from "@/components/socios/icons";
import { benefitHeadline } from "@/lib/benefits/format";
import { displayCode } from "@/lib/benefits/code";
import { readLastBranch, writeLastBranch } from "@/lib/benefits/lastBranch";
import { problemOfStatus } from "@/lib/benefits/redeem";
import { useRedeem } from "@/lib/benefits/useRedeem";
import { useBusiness } from "@/lib/business/BusinessProvider";

import { PreviewCard } from "./PreviewCard";

// El escáner y su librería se bajan recién cuando el dueño toca "Escanear QR".
const QrScanner = dynamic(() => import("./QrScanner"), {
  ssr: false,
  loading: () => <p className="text-center text-sm text-movo-slate">Abriendo la cámara…</p>,
});

/** Canjear cupón: se escribe el código, se ve el cupón y se confirma. Buscar no canjea nada. */
export function RedeemView() {
  const { business } = useBusiness();
  const redeem = useRedeem();
  const [code, setCode] = useState("");
  const [chosen, setChosen] = useState(() => readLastBranch());
  const [scanning, setScanning] = useState(false);
  const { state } = redeem;

  const branches =
    business.status === "ready" ? business.data.branches.filter((b) => b.active) : [];
  // Con una sola sede activa la API la elige; con varias, el dueño dice dónde canjeó.
  const needsBranch = branches.length > 1;
  // La última sede elegida en este dispositivo, mientras siga siendo una sede activa.
  const branch = branches.some((item) => String(item.id) === chosen) ? chosen : "";
  const busy = state.step === "looking" || state.step === "confirming";

  function lookup(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    void redeem.lookup(code);
  }

  function scanned(scannedCode: string) {
    setScanning(false);
    setCode(displayCode(scannedCode));
    void redeem.lookup(scannedCode);
  }

  // Con una sola sede activa es esa; con varias, la última que eligió el dueño.
  const scanBranch = needsBranch
    ? branches.find((item) => String(item.id) === branch)
    : branches[0];

  function manual() {
    setScanning(false);
    requestAnimationFrame(() => document.getElementById("coupon-code")?.focus());
  }

  function again() {
    setCode("");
    redeem.reset();
  }

  if (state.step === "done") {
    return (
      <div className="space-y-5">
        <section role="status" className="space-y-3 rounded-3xl bg-white p-6 text-center shadow-sm">
          <CheckCircleIcon className="mx-auto size-16 text-movo-ok" />
          <h2 className="font-display text-3xl font-extrabold">Canje confirmado</h2>
          <p className="text-movo-slate">
            Entregá{" "}
            <strong className="text-movo-ink">{benefitHeadline(state.preview.benefit)}</strong> a{" "}
            {state.preview.customer_name}.
          </p>
        </section>
        <button
          type="button"
          onClick={again}
          className="flex h-14 w-full items-center justify-center gap-2 rounded-full bg-movo-accent font-display text-lg font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
        >
          <QrIcon className="size-5" />
          Canjear otro cupón
        </button>
      </div>
    );
  }

  const preview = state.step === "preview" || state.step === "confirming" ? state.preview : null;
  const canConfirm = preview !== null && problemOfStatus(preview.status) === null;

  return (
    <div className="space-y-5">
      <div>
        <h2 className="font-display text-3xl leading-tight font-extrabold">Canjear cupón</h2>
        <p className="mt-1 text-base text-movo-slate">
          Escaneá el QR de tu cliente o ingresá su código para ver el cupón antes de confirmar.
        </p>
      </div>

      {scanning ? (
        <QrScanner
          onCode={scanned}
          onClose={() => setScanning(false)}
          onManual={manual}
          branchName={scanBranch?.name}
        />
      ) : (
        <button
          type="button"
          onClick={() => setScanning(true)}
          disabled={busy}
          className="flex h-14 w-full items-center justify-center gap-2 rounded-full bg-movo-accent font-display text-lg font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink disabled:opacity-50"
        >
          <QrIcon className="size-5" />
          Escanear QR
        </button>
      )}

      <form onSubmit={lookup} noValidate className="space-y-4 rounded-3xl bg-white p-5 shadow-sm">
        <Field
          id="coupon-code"
          label="Código del cupón"
          error={state.step === "idle" ? state.error : undefined}
        >
          <input
            id="coupon-code"
            name="coupon-code"
            value={code}
            onChange={(event) => {
              setCode(displayCode(event.target.value));
              if (state.step !== "idle" && state.step !== "looking") redeem.reset();
            }}
            disabled={busy}
            autoComplete="off"
            autoCapitalize="characters"
            spellCheck={false}
            inputMode="text"
            maxLength={9}
            placeholder="ABCD-EFGH"
            aria-invalid={state.step === "idle" && state.error ? true : undefined}
            aria-describedby={
              state.step === "idle" && state.error ? "coupon-code-error" : undefined
            }
            className="h-16 w-full rounded-2xl bg-movo-sand px-4 text-center font-display text-3xl font-extrabold tracking-widest text-movo-ink uppercase outline-none placeholder:text-movo-slate/40 focus:ring-2 focus:ring-movo-accent disabled:opacity-60"
          />
        </Field>
        <button
          type="submit"
          disabled={busy || code.length === 0}
          className="h-14 w-full rounded-full bg-movo-page font-display text-lg font-bold text-white outline-none focus-visible:ring-2 focus-visible:ring-movo-accent disabled:opacity-50"
        >
          {state.step === "looking" ? "Buscando…" : "Ver cupón"}
        </button>
      </form>

      {state.step === "problem" ? (
        <Notice title={state.problem.title} body={state.problem.body} />
      ) : null}

      {preview ? (
        <>
          <PreviewCard preview={preview} now={new Date()} />

          {needsBranch && canConfirm ? (
            <Field
              id="redeem-branch"
              label="Sede donde se canjea"
              error={state.step === "preview" ? state.problem?.body : undefined}
            >
              <select
                id="redeem-branch"
                value={branch}
                onChange={(event) => {
                  setChosen(event.target.value);
                  writeLastBranch(event.target.value);
                }}
                disabled={busy}
                className="h-14 w-full rounded-2xl bg-movo-sand px-4 text-base text-movo-ink outline-none focus:ring-2 focus:ring-movo-accent"
              >
                <option value="">Elegí una sede</option>
                {branches.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.name}
                  </option>
                ))}
              </select>
            </Field>
          ) : null}

          {canConfirm ? (
            <button
              type="button"
              disabled={busy || (needsBranch && branch === "")}
              onClick={() => void redeem.confirm(needsBranch ? Number(branch) : undefined)}
              className="flex h-14 w-full items-center justify-center gap-2 rounded-full bg-movo-accent font-display text-lg font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink disabled:opacity-50"
            >
              {state.step === "confirming" ? "Confirmando…" : "Confirmar canje"}
            </button>
          ) : null}
        </>
      ) : null}
    </div>
  );
}
