"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState, type ReactNode } from "react";

import { useMyBusiness } from "@/lib/business/hooks";
import { BUSINESS_STATUS, type MyBusiness } from "@/lib/business/types";
import { PANEL_PATH, useRequireSession } from "@/lib/session/hooks";

import { Notice } from "./fields";
import { ReviewScreen } from "./ReviewScreen";
import type { StepNumber } from "./Shell";
import { StepBranch } from "./StepBranch";
import { StepBusiness } from "./StepBusiness";
import { StepPhotos } from "./StepPhotos";
import { StepServices } from "./StepServices";

/**
 * El alta en cuatro pasos. Retoma donde el dueño quedó: sin negocio empieza por el 1,
 * sin sede por el 2, y si ya tiene ambos por el 3. Con el negocio enviado muestra la
 * pantalla de espera, y con el negocio aprobado lo manda al panel.
 */
export function OnboardingFlow() {
  const session = useRequireSession();
  const mine = useMyBusiness();
  const router = useRouter();
  const [chosen, setChosen] = useState<StepNumber | null>(null);

  const business = mine.state.status === "ready" ? mine.state.data.business : null;
  const approved =
    business?.status === BUSINESS_STATUS.ACTIVE || business?.status === BUSINESS_STATUS.PAUSED;

  useEffect(() => {
    if (approved) router.replace(PANEL_PATH);
  }, [approved, router]);

  if (session.status !== "authenticated" || mine.state.status === "loading" || approved) {
    return <Splash />;
  }
  if (mine.state.status === "failed") {
    return (
      <Splash>
        <Notice
          title="No pudimos cargar tu negocio"
          body="Recargá la página para volver a intentar."
        />
      </Splash>
    );
  }

  const data = mine.state.status === "ready" ? mine.state.data : null;
  if (data?.business.status === BUSINESS_STATUS.DELETED) {
    return (
      <Splash>
        <Notice title="Este negocio fue dado de baja" body="Escribinos si fue un error." />
      </Splash>
    );
  }
  if (data && data.business.status !== BUSINESS_STATUS.DRAFT) return <ReviewScreen data={data} />;

  const step = chosen ?? firstStep(data);
  const go = (next: StepNumber) => setChosen(next);

  switch (step) {
    case 1:
      return (
        <StepBusiness
          business={data?.business ?? null}
          onSaved={mine.reload}
          onNext={() => go(2)}
        />
      );
    case 2:
      return (
        <StepBranch
          branch={data?.branches[0] ?? null}
          onSaved={mine.reload}
          onNext={() => go(3)}
          onBack={() => go(1)}
        />
      );
    case 3:
      return <StepServices onNext={() => go(4)} onBack={() => go(2)} />;
    case 4:
      // Sin negocio no hay nada que enviar; el paso 1 siempre lo crea antes de llegar acá.
      return data ? (
        <StepPhotos
          business={data.business}
          photos={data.images}
          onChanged={mine.reload}
          onBack={() => go(3)}
        />
      ) : null;
  }
}

/** Primer paso con algo por completar: negocio, después sede, después servicios. */
function firstStep(data: MyBusiness | null): StepNumber {
  if (!data) return 1;
  if (data.branches.length === 0) return 2;
  return 3;
}

function Splash({ children }: { children?: ReactNode }) {
  return (
    <div className="grid min-h-dvh place-items-center bg-movo-cream px-6">
      {children ?? <p className="text-movo-slate">Cargando…</p>}
    </div>
  );
}
