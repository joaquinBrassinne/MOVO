import { SessionProvider } from "@/lib/session/SessionProvider";

// El panel del super admin tiene su propia sesión: es la misma cuenta y cookie del
// portal, pero con otro marco y otro rol.
export default function AdminLayout({ children }: LayoutProps<"/admin">) {
  return <SessionProvider>{children}</SessionProvider>;
}
