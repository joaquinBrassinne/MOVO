import { PanelChrome } from "@/components/panel/PanelChrome";
import { OnboardingGate } from "@/components/socios/OnboardingGate";
import { BusinessProvider } from "@/lib/business/BusinessProvider";

// El panel del dueño: carga su negocio una vez, lo manda al alta si todavía no lo envió
// y le pone el marco con la navegación.
export default function PanelLayout({ children }: LayoutProps<"/socios">) {
  return (
    <BusinessProvider>
      <OnboardingGate>
        <PanelChrome>{children}</PanelChrome>
      </OnboardingGate>
    </BusinessProvider>
  );
}
