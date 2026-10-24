import { SessionProvider } from "@/lib/session/SessionProvider";

// Todo el portal comparte la sesión; cada grupo de rutas pone su propio marco
// ((acceso) el oscuro de ingreso y alta, onboarding el de los pasos).
export default function SociosLayout({ children }: LayoutProps<"/socios">) {
  return <SessionProvider>{children}</SessionProvider>;
}
