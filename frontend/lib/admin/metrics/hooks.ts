"use client";

import { useEffect, useState } from "react";

import type { Loaded } from "@/lib/business/hooks";

import { getAdminMetrics } from "./api";
import type { AdminMetrics } from "./types";

/** The platform figures; `reload` asks for them again. */
export function useAdminMetrics(): { state: Loaded<AdminMetrics>; reload: () => void } {
  const [version, setVersion] = useState(0);
  const [state, setState] = useState<Loaded<AdminMetrics>>({ status: "loading" });

  useEffect(() => {
    let ignore = false;

    void getAdminMetrics().then((result) => {
      if (ignore) return;
      setState(
        result.ok
          ? { status: "ready", data: result.data }
          : { status: "failed", error: result.error },
      );
    });

    return () => {
      ignore = true;
    };
  }, [version]);

  return { state, reload: () => setVersion((current) => current + 1) };
}
