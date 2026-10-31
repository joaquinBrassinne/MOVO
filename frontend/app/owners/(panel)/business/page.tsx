import type { Metadata } from "next";

import { LocalHub } from "@/components/panel/LocalHub";

export const metadata: Metadata = { title: "Mi local — MOVO Socios" };

export default function LocalPage() {
  return <LocalHub />;
}
