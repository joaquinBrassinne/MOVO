import type { Metadata } from "next";

import { PhotosPanel } from "@/components/panel/PhotosPanel";

export const metadata: Metadata = { title: "Fotos — MOVO Socios" };

export default function PhotosPage() {
  return <PhotosPanel />;
}
