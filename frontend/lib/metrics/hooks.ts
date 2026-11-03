"use client";

import { useEffect, useState } from "react";

import type { Loaded } from "@/lib/business/hooks";

import { getMetric, getMetricsAccess, getPublicPlans } from "./api";
import type { MetricMap, MetricName, Period } from "./types";

interface Resource<T> {
  state: Loaded<T>;
  reload: () => void;
}

/**
 * One metric for one period. The answer is kept with the key it was asked for, so a new period
 * (or a reload) shows `loading` again without setting state inside the effect.
 */
export function useMetric<K extends MetricName>(
  name: K,
  period: Period,
  enabled = true,
): Resource<MetricMap[K]> {
  const [version, setVersion] = useState(0);
  const [answer, setAnswer] = useState<{ key: string; state: Loaded<MetricMap[K]> } | null>(null);
  const key = `${name}:${period}:${version}`;

  useEffect(() => {
    if (!enabled) return;
    let ignore = false;

    void getMetric(name, period).then((result) => {
      if (ignore) return;
      if (result.ok) setAnswer({ key, state: { status: "ready", data: result.data } });
      else if (result.error.code === "business_not_found")
        setAnswer({ key, state: { status: "missing" } });
      else setAnswer({ key, state: { status: "failed", error: result.error } });
    });

    return () => {
      ignore = true;
    };
  }, [key, name, period, enabled]);

  const state: Loaded<MetricMap[K]> = answer?.key === key ? answer.state : { status: "loading" };
  return { state, reload: () => setVersion((current) => current + 1) };
}

/** Whether the plan includes the Pro metrics, as the plan says (never by plan name). */
export function useAdvancedMetrics(): Loaded<boolean> {
  const [state, setState] = useState<Loaded<boolean>>({ status: "loading" });

  useEffect(() => {
    let ignore = false;

    void getMetricsAccess().then((result) => {
      if (ignore) return;
      if (result.ok) setState({ status: "ready", data: result.data.advanced_metrics });
      else if (result.error.code === "business_not_found") setState({ status: "missing" });
      else setState({ status: "failed", error: result.error });
    });

    return () => {
      ignore = true;
    };
  }, []);

  return state;
}

/** Monthly price of the Pro plan from the public catalog; null while loading or on failure. */
export function useProPrice(enabled: boolean): number | null {
  const [price, setPrice] = useState<number | null>(null);

  useEffect(() => {
    if (!enabled) return;
    let ignore = false;

    void getPublicPlans().then((result) => {
      if (ignore || !result.ok) return;
      setPrice(result.data.plans.find((plan) => plan.plan === "pro")?.price ?? null);
    });

    return () => {
      ignore = true;
    };
  }, [enabled]);

  return price;
}
