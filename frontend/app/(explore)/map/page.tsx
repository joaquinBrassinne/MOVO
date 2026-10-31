import type { Metadata } from "next";

import { MapScreen } from "@/components/explorar/MapScreen";

export const metadata: Metadata = {
  title: "Mapa — MOVO",
  description: "Mirá en el mapa los gimnasios, pilates, yoga y más que tenés cerca.",
};

export default function MapPage() {
  return <MapScreen />;
}
