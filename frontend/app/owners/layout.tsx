import { SessionProvider } from "@/lib/session/SessionProvider";

// Todo el portal comparte la sesión; cada grupo de rutas pone su propio marco
// ((auth) el oscuro de ingreso y alta, onboarding el de los pasos).
export default function SociosLayout({ children }: LayoutProps<"/owners">) {
  return <SessionProvider>{children}</SessionProvider>;
}
