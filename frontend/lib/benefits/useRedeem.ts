"use client";

import { useState } from "react";

import { previewCoupon, redeemCoupon } from "./api";
import { normalizeCode } from "./code";
import { problemOf, PROBLEM, type Problem } from "./redeem";
import type { CouponPreview, RedeemedCoupon } from "./types";

export type RedeemState =
  /** Esperando un código; `error` si el que se escribió no puede ser un código. */
  | { step: "idle"; error?: string }
  | { step: "looking" }
  /** El cupón visto, listo para confirmar; `problem` si la confirmación falló y hay que corregir algo. */
  | { step: "preview"; preview: CouponPreview; problem?: Problem }
  | { step: "confirming"; preview: CouponPreview }
  | { step: "done"; preview: CouponPreview; redeemed: RedeemedCoupon }
  | { step: "problem"; problem: Problem };

const INVALID_CODE =
  "Revisá el código: tiene 8 letras y números, sin O, I, 0 ni 1. Pedíselo de nuevo al cliente.";

/**
 * El flujo de canjear: buscar el código, ver el cupón y recién entonces confirmar. Buscar
 * no canjea nada; el canje es una acción aparte y la API lo hace una sola vez.
 */
export function useRedeem() {
  const [state, setState] = useState<RedeemState>({ step: "idle" });

  async function lookup(raw: string) {
    const code = normalizeCode(raw);
    if (!code) {
      setState({ step: "idle", error: INVALID_CODE });
      return;
    }

    setState({ step: "looking" });
    const result = await previewCoupon(code);
    setState(
      result.ok
        ? { step: "preview", preview: result.data }
        : { step: "problem", problem: problemOf(result.error) },
    );
  }

  /** Canjea el cupón que se está viendo. `branchId` solo si el negocio tiene varias sedes. */
  async function confirm(branchId?: number) {
    if (state.step !== "preview") return;
    const { preview } = state;

    setState({ step: "confirming", preview });
    const result = await redeemCoupon(preview.code, branchId);
    if (result.ok) {
      setState({ step: "done", preview, redeemed: result.data });
      return;
    }

    const problem = problemOf(result.error);
    // Falta la sede: se sigue viendo el cupón para elegirla. Cualquier otra cosa lo cierra.
    setState(
      problem.kind === PROBLEM.BRANCH
        ? { step: "preview", preview, problem }
        : { step: "problem", problem },
    );
  }

  function reset() {
    setState({ step: "idle" });
  }

  return { state, lookup, confirm, reset };
}
