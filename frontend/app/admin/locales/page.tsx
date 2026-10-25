import type { Metadata } from "next";

import { LocalesView } from "@/components/admin/LocalesView";

export const metadata: Metadata = { title: "Locales — MOVO Super admin" };

export default function LocalesPage() {
  return <LocalesView />;
}
