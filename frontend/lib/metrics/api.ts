import { authed } from "@/lib/api/authed";
import { requestPublic } from "@/lib/api/request";
import type { PlansResponse } from "@/lib/plans/types";

import { PERIODLESS, type MetricMap, type MetricName, type Period } from "./types";

const BASE = "/me/business/metrics";

export function getMetric<K extends MetricName>(name: K, period: Period) {
  const path = name === "essentials" ? BASE : `${BASE}/${name}`;
  const query = PERIODLESS.includes(name) ? "" : `?period=${period}`;
  return authed<MetricMap[K]>("GET", `${path}${query}`);
}

/** Only the flag the metrics page needs from `GET /me/plan`. */
export const getMetricsAccess = () => authed<{ advanced_metrics: boolean }>("GET", "/me/plan");

export const getPublicPlans = () => requestPublic<PlansResponse>("/plans");
