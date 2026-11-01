import type { Metadata } from "next";
import { Suspense } from "react";

import { ResetPasswordForm } from "@/components/passwordreset/ResetPasswordForm";
import { PortalChrome } from "@/components/socios/PortalChrome";
import { supportWhatsAppUrl } from "@/lib/support";

export const metadata: Metadata = {
  title: "Elegí tu nueva clave — MOVO Socios",
  robots: { index: false, follow: false },
  referrer: "no-referrer",
};

export default function ResetPasswordPage() {
  const supportUrl = supportWhatsAppUrl(
    "Hola, necesito un enlace nuevo para elegir mi clave de MOVO Socios.",
  );

  return (
    <PortalChrome>
      <div className="mb-6 text-center">
        <h1 className="font-display text-[1.75rem] leading-tight font-extrabold tracking-tight">
          Elegí tu nueva clave
        </h1>
        <p className="mt-3 text-base text-movo-muted">
          Creá una contraseña segura para ingresar a tu panel de socio MOVO.
        </p>
      </div>
      <Suspense fallback={null}>
        <ResetPasswordForm supportUrl={supportUrl} />
      </Suspense>
    </PortalChrome>
  );
}
