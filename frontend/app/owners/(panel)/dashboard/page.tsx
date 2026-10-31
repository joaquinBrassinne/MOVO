import type { Metadata } from "next";

import { OwnerHome } from "@/components/home/OwnerHome";

export const metadata: Metadata = { title: "Panel — MOVO Socios" };

export default function PanelPage() {
  return <OwnerHome />;
}
