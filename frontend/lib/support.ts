/**
 * Enlace de WhatsApp del soporte. El reseteo de contraseña es asistido (SDD §5.7):
 * el super admin lo gestiona por WhatsApp, no hay email. Sin número configurado
 * devuelve null y las vistas ocultan el acceso en lugar de inventar uno.
 */
export function supportWhatsAppUrl(message: string): string | null {
  const digits = (process.env.NEXT_PUBLIC_SUPPORT_WHATSAPP ?? "").replace(/\D/g, "");
  if (digits === "") return null;

  return `https://wa.me/${digits}?text=${encodeURIComponent(message)}`;
}
