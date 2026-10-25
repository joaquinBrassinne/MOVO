import type { Metadata } from "next";

import { AuthIntro } from "@/components/socios/AuthIntro";
import { AuthTabs } from "@/components/socios/AuthTabs";
import { RegisterForm } from "@/components/socios/RegisterForm";

export const metadata: Metadata = { title: "Crear cuenta — MOVO Socios" };

export default function CrearCuentaPage() {
  return (
    <>
      <AuthIntro />
      <div className="mt-6">
        <AuthTabs active="register" />
      </div>
      <section className="mt-6 rounded-[28px] bg-movo-card p-5">
        <RegisterForm />
      </section>
    </>
  );
}
