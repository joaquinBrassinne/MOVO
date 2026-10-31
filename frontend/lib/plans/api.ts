// Public plan catalog read from Next's server. The catalog only changes with an API restart,
// so a few minutes of cache is plenty.

import { cacheLife } from "next/cache";

import type { Plan, PlansResponse } from "./types.ts";

const API_URL = process.env.API_URL ?? "http://localhost:8080";
const TIMEOUT_MS = 10_000;

/** The plans, cheapest first. Throws when the API fails (a failure is never cached). */
export async function getPlans(): Promise<Plan[]> {
  "use cache";
  cacheLife({ stale: 60, revalidate: 300, expire: 900 });

  const response = await fetch(`${API_URL}/api/v1/plans`, {
    signal: AbortSignal.timeout(TIMEOUT_MS),
  });
  if (!response.ok) throw new Error(`plans: the API answered ${response.status}`);
  return ((await response.json()) as PlansResponse).plans;
}
