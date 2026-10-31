import type { Metadata } from "next";

import { ServicesPanel } from "@/components/panel/ServicesPanel";

export const metadata: Metadata = { title: "Servicios y horarios — MOVO Socios" };

export default function ServicesPage() {
  return <ServicesPanel />;
}
