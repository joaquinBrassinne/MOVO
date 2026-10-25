import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { BranchEditor } from "@/components/panel/BranchEditor";

export const metadata: Metadata = { title: "Editar sede — MOVO Socios" };

export default async function EditBranchPage({ params }: PageProps<"/socios/sedes/[id]">) {
  const id = Number((await params).id);
  if (!Number.isSafeInteger(id) || id <= 0) notFound();

  return <BranchEditor branchId={id} />;
}
