/** One entry of the public plan catalog served by `GET /plans`. */
export interface Plan {
  plan: "starter" | "pro";
  /** Monthly price in whole Argentine pesos. */
  price: number;
  checkout_url: string;
  max_branches: number;
  /** `null` means unlimited. */
  max_active_services: number | null;
}

export interface PlansResponse {
  plans: Plan[];
}
