"use client";

import { createContext, useContext, useEffect, useState, type ReactNode } from "react";

import { getMyPlan } from "./api";
import { useMyBusiness, type Loaded } from "./hooks";
import type { MyBusiness, PlanSummary } from "./types";

interface BusinessContext {
  business: Loaded<MyBusiness>;
  /** Vuelve a pedir el negocio, por ejemplo después de guardar una sede. */
  reload: () => void;
  /** El plan y sus límites, para los contadores "2 de 3 servicios activos". */
  plan: Loaded<PlanSummary>;
}

const Context = createContext<BusinessContext | null>(null);

/** Carga una sola vez el negocio del dueño y su plan para todas las vistas del panel. */
export function BusinessProvider({ children }: { children: ReactNode }) {
  const { state, reload } = useMyBusiness();
  const [plan, setPlan] = useState<Loaded<PlanSummary>>({ status: "loading" });

  useEffect(() => {
    let ignore = false;

    void getMyPlan().then((result) => {
      if (ignore) return;
      if (result.ok) setPlan({ status: "ready", data: result.data });
      else if (result.error.code === "business_not_found") setPlan({ status: "missing" });
      else setPlan({ status: "failed", error: result.error });
    });

    return () => {
      ignore = true;
    };
  }, []);

  return <Context value={{ business: state, reload, plan }}>{children}</Context>;
}

export function useBusiness(): BusinessContext {
  const context = useContext(Context);
  if (!context) throw new Error("useBusiness must be used inside <BusinessProvider>");
  return context;
}
