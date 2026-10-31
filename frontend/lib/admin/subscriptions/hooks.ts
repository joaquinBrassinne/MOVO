"use client";

import { useEffect, useState } from "react";

import type { ApiResult } from "@/lib/api/request";
import type { Loaded } from "@/lib/business/hooks";
import type { Plan } from "@/lib/plans/types";

import { getPlanCatalog, getSubscription, listSubscriptions } from "./api";
import type { Subscription, SubscriptionRow } from "./types";

interface Resource<T> {
  state: Loaded<T>;
  reload: () => void;
}

/** Loads `fetcher` on mount and on every `reload()`; a 404 is `missing` when `missingOn404`. */
function useLoaded<T, R>(
  fetcher: () => Promise<ApiResult<R>>,
  pick: (data: R) => T,
  deps: readonly unknown[],
  missingOn404 = false,
): Resource<T> {
  const [version, setVersion] = useState(0);
  const [state, setState] = useState<Loaded<T>>({ status: "loading" });

  useEffect(() => {
    let ignore = false;

    void fetcher().then((result) => {
      if (ignore) return;
      if (result.ok) setState({ status: "ready", data: pick(result.data) });
      else if (missingOn404 && result.status === 404) setState({ status: "missing" });
      else setState({ status: "failed", error: result.error });
    });

    return () => {
      ignore = true;
    };
    // `fetcher` and `pick` are new on every render; the caller names what really changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, version]);

  return { state, reload: () => setVersion((current) => current + 1) };
}

const rowsOf = (data: { subscriptions: SubscriptionRow[] }) => data.subscriptions;
const plansOf = (data: { plans: Plan[] }) => data.plans;
const same = (data: Subscription) => data;

/** Every subscription, ordered by expiry; the tabs are derived on the client. */
export function useSubscriptions(): Resource<SubscriptionRow[]> {
  return useLoaded(listSubscriptions, rowsOf, []);
}

/** The subscription of a business; `missing` when it has none. */
export function useSubscription(businessId: number): Resource<Subscription> {
  return useLoaded(() => getSubscription(businessId), same, [businessId], true);
}

/** The public plan catalog, for the prices next to the plan names. */
export function usePlanCatalog(): Resource<Plan[]> {
  return useLoaded(getPlanCatalog, plansOf, []);
}
