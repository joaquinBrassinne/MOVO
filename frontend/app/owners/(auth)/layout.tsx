import { PortalChrome } from "@/components/socios/PortalChrome";

export default function AccesoLayout({ children }: LayoutProps<"/owners">) {
  return <PortalChrome>{children}</PortalChrome>;
}
