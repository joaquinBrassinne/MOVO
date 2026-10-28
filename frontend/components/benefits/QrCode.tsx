import { qrShape } from "@/lib/scanner/qr";

interface Props {
  /** El código del cupón: es lo único que lleva el QR. */
  code: string;
  /** Lo que lee un lector de pantalla, por ejemplo "Código QR del cupón ABCD-EFGH". */
  label: string;
  className?: string;
}

/** El QR de un cupón, dibujado como SVG: nítido a cualquier tamaño y sin pedir nada a la red. */
export function QrCode({ code, label, className }: Props) {
  const { size, path } = qrShape(code);

  return (
    <svg
      viewBox={`0 0 ${size} ${size}`}
      role="img"
      aria-label={label}
      shapeRendering="crispEdges"
      className={className}
    >
      <rect width={size} height={size} className="fill-white" />
      <path d={path} className="fill-movo-ink" />
    </svg>
  );
}
