import { authed } from "@/lib/api/authed";
import type { PlansResponse } from "@/lib/plans/types";

import type {
  ConfirmPaymentInput,
  PlanChange,
  PlanId,
  Subscription,
  SubscriptionList,
} from "./types";

export const listSubscriptions = () => authed<SubscriptionList>("GET", "/admin/subscriptions");

export const getSubscription = (businessId: number) =>
  authed<Subscription>("GET", `/admin/subscriptions/${businessId}`);

export const confirmPayment = (businessId: number, input: ConfirmPaymentInput) =>
  authed<Subscription>("POST", `/admin/subscriptions/${businessId}`, input);

export const registerPayment = (businessId: number) =>
  authed<Subscription>("POST", `/admin/subscriptions/${businessId}/payments`);

export const changePlan = (businessId: number, plan: PlanId) =>
  authed<PlanChange>("PUT", `/admin/subscriptions/${businessId}/plan`, { plan });

export const getPlanCatalog = () => authed<PlansResponse>("GET", "/plans");
