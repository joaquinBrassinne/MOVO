"use client";

import { useEffect, useState } from "react";

import type { Loaded } from "@/lib/business/hooks";

import { getMySubscription, listMyAlerts, markAlertRead } from "./api";
import { ALERT_TYPE, type PlanAlert, type SubscriptionView } from "./types";

interface Resource<T> {
  state: Loaded<T>;
  reload: () => void;
}

export function useSubscription(): Resource<SubscriptionView> {
  const [version, setVersion] = useState(0);
  const [state, setState] = useState<Loaded<SubscriptionView>>({ status: "loading" });

  useEffect(() => {
    let ignore = false;

    void getMySubscription().then((result) => {
      if (ignore) return;
      if (result.ok) setState({ status: "ready", data: result.data });
      else if (result.error.code === "business_not_found") setState({ status: "missing" });
      else setState({ status: "failed", error: result.error });
    });

    return () => {
      ignore = true;
    };
  }, [version]);

  return { state, reload: () => setVersion((current) => current + 1) };
}

/** Unread plan-downgrade alerts. A failed load shows none: the plan view still works. */
export function useDowngradeAlerts() {
  const [alerts, setAlerts] = useState<PlanAlert[]>([]);

  useEffect(() => {
    let ignore = false;

    void listMyAlerts().then((result) => {
      if (ignore || !result.ok) return;
      setAlerts(result.data.alerts.filter((alert) => alert.type === ALERT_TYPE.PLAN_DOWNGRADE));
    });

    return () => {
      ignore = true;
    };
  }, []);

  /** Marks it read and removes it; a 404 means it was already gone. */
  async function dismiss(id: number): Promise<boolean> {
    const result = await markAlertRead(id);
    if (!result.ok && result.error.code !== "alert_not_found") return false;
    setAlerts((current) => current.filter((alert) => alert.id !== id));
    return true;
  }

  return { alerts, dismiss };
}
