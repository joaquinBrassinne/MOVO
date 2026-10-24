import { PortalChrome } from "@/components/socios/PortalChrome";
import { SessionProvider } from "@/lib/session/SessionProvider";

export default function SociosLayout({ children }: LayoutProps<"/socios">) {
  return (
    <SessionProvider>
      <PortalChrome>{children}</PortalChrome>
    </SessionProvider>
  );
}
