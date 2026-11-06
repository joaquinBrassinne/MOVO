import type { Metadata } from "next";

import { BusinessData } from "@/components/panel/BusinessData";

export const metadata: Metadata = { title: "Datos del negocio — MOVO Socios" };

export default function BusinessDataPage() {
  return <BusinessData />;
}
