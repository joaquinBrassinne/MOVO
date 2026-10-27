"use client";

import { useEffect, useState } from "react";

import type { ApiResult } from "@/lib/api/request";
import type { ApiError } from "@/lib/session/types";

import { getActivity, getCouponQuota, getOverview, listBenefits } from "./api";
import type { Activity, BenefitListItem, CouponQuota, Overview } from "./types";

export type Remote<T> =
  { status: "loading" } | { status: "ready"; data: T } | { status: "failed"; error: ApiError };

interface Resource<T> {
  state: Remote<T>;
  /** Vuelve a pedir los datos, por ejemplo después de guardar. */
  reload: () => void;
}

/**
 * Pide `load` al montar y cada vez que se llama a `reload`; con `pollMs` también la repite
 * mientras la pestaña está a la vista. Mientras repite, los datos anteriores se siguen
 * mostrando: no vuelve a "cargando". `load` tiene que ser estable (una función del módulo).
 */
function useResource<T>(load: () => Promise<ApiResult<T>>, pollMs?: number): Resource<T> {
  const [version, setVersion] = useState(0);
  const [state, setState] = useState<Remote<T>>({ status: "loading" });

  useEffect(() => {
    let ignore = false;

    void load().then((result) => {
      if (ignore) return;
      if (result.ok) setState({ status: "ready", data: result.data });
      // Un fallo del refresco no tira abajo lo que ya se mostraba.
      else setState((current) => (current.status === "ready" ? current : failed(result.error)));
    });

    return () => {
      ignore = true;
    };
  }, [load, version]);

  useEffect(() => {
    if (!pollMs) return;
    const timer = setInterval(() => {
      if (document.visibilityState === "visible") setVersion((current) => current + 1);
    }, pollMs);
    return () => clearInterval(timer);
  }, [pollMs]);

  return { state, reload: () => setVersion((current) => current + 1) };
}

function failed(error: ApiError): Remote<never> {
  return { status: "failed", error };
}

/** Los beneficios del local con lo que hizo cada uno este mes. */
export function useBenefits(): Resource<BenefitListItem[]> {
  const { state, reload } = useResource(listBenefits);
  return {
    state: state.status === "ready" ? { status: "ready", data: state.data.benefits } : state,
    reload,
  };
}

/** El cupo mensual del local. */
export function useCouponQuota(): Resource<CouponQuota> {
  return useResource(getCouponQuota);
}

/** Los números del inicio del panel; se actualizan solos cada 60 segundos. */
export function useOverview(): Resource<Overview> {
  return useResource(getOverview, 60_000);
}

/** La actividad reciente del local, consultada cada 45 segundos mientras la página está abierta. */
export function useActivity(since: string | undefined): Resource<Activity> {
  const [version, setVersion] = useState(0);
  const [state, setState] = useState<Remote<Activity>>({ status: "loading" });

  useEffect(() => {
    let ignore = false;

    void getActivity(since).then((result) => {
      if (ignore) return;
      if (result.ok) setState({ status: "ready", data: result.data });
      else setState((current) => (current.status === "ready" ? current : failed(result.error)));
    });

    return () => {
      ignore = true;
    };
  }, [since, version]);

  useEffect(() => {
    const timer = setInterval(() => {
      if (document.visibilityState === "visible") setVersion((current) => current + 1);
    }, 45_000);
    return () => clearInterval(timer);
  }, []);

  return { state, reload: () => setVersion((current) => current + 1) };
}
