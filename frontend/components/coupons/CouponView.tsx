"use client";

import { use } from "react";
import { useSearchParams } from "next/navigation";

import { CouponScreen } from "./CouponScreen";

/** Lee el token de la ruta y si el cupón viene de generarse recién (`?nuevo=1`). */
export function CouponView({ params }: { params: Promise<{ token: string }> }) {
  const { token } = use(params);
  const fresh = useSearchParams().get("nuevo") === "1";

  return <CouponScreen token={token} fresh={fresh} />;
}
