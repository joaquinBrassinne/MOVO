import type { Metadata } from "next";

import { BranchEditor } from "@/components/panel/BranchEditor";

export const metadata: Metadata = { title: "Nueva sede — MOVO Socios" };

export default function NewBranchPage() {
  return <BranchEditor branchId={null} />;
}
