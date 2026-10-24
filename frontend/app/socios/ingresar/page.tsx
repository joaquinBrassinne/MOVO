import type { Metadata } from "next";

import { AuthIntro } from "@/components/socios/AuthIntro";
import { AuthTabs } from "@/components/socios/AuthTabs";
import { LoginForm } from "@/components/socios/LoginForm";
import { supportWhatsAppUrl } from "@/lib/support";

export const metadata: Metadata = { title: "Ingresar — MOVO Socios" };

export default function IngresarPage() {
  const supportUrl = supportWhatsAppUrl(
    "Hola, necesito ayuda para ingresar a mi cuenta de MOVO Socios.",
  );

  return (
    <>
      <AuthIntro badge />
      <div className="mt-6">
        <AuthTabs active="login" withIcons />
      </div>
      <section className="mt-6 rounded-[28px] bg-movo-card p-5">
        <LoginForm supportUrl={supportUrl} />
      </section>
    </>
  );
}
