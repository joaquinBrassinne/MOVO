import { authed } from "@/lib/api/authed";

import type { PlanAlert, SubscriptionView } from "./types";

export const getMySubscription = () => authed<SubscriptionView>("GET", "/me/subscription");
export const listMyAlerts = () => authed<{ alerts: PlanAlert[] }>("GET", "/me/alerts");
export const markAlertRead = (id: number) => authed<void>("POST", `/me/alerts/${id}/read`);
