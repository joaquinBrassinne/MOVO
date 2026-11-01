import { authed } from "@/lib/api/authed";

import type { AdminMetrics } from "./types";

export const getAdminMetrics = () => authed<AdminMetrics>("GET", "/admin/metrics");
