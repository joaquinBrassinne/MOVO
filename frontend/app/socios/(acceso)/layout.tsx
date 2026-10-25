import { PortalChrome } from "@/components/socios/PortalChrome";

export default function AccesoLayout({ children }: LayoutProps<"/socios">) {
  return <PortalChrome>{children}</PortalChrome>;
}
