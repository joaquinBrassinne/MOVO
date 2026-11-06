// Qué se le dice al dueño cuando la cámara no se puede usar. Función pura: se prueba sin
// navegador. En todos los casos el ingreso manual del código sigue disponible.

export const CAMERA_PROBLEM = {
  DENIED: "denied",
  UNAVAILABLE: "unavailable",
  FAILED: "failed",
} as const;

export type CameraProblemKind = (typeof CAMERA_PROBLEM)[keyof typeof CAMERA_PROBLEM];

export interface CameraProblem {
  kind: CameraProblemKind;
  title: string;
  body: string;
}

const MANUAL = "Podés ingresar el código a mano.";

/**
 * El problema que corresponde al error de `getUserMedia`. Con `undefined` (el navegador no
 * ofrece cámara, por ejemplo en una página sin HTTPS) también se explica.
 */
export function cameraProblemOf(error: unknown): CameraProblem {
  const name = error instanceof Error || error instanceof DOMException ? error.name : "";

  if (name === "NotAllowedError" || name === "SecurityError") {
    return {
      kind: CAMERA_PROBLEM.DENIED,
      title: "No tenemos permiso para usar la cámara",
      body: `Habilitala en los permisos del navegador. ${MANUAL}`,
    };
  }
  if (name === "NotFoundError" || name === "OverconstrainedError" || error === undefined) {
    return {
      kind: CAMERA_PROBLEM.UNAVAILABLE,
      title: "No encontramos una cámara",
      body: `Este dispositivo o navegador no ofrece cámara. ${MANUAL}`,
    };
  }
  return {
    kind: CAMERA_PROBLEM.FAILED,
    title: "No pudimos abrir la cámara",
    body: `Puede estar en uso por otra aplicación. ${MANUAL}`,
  };
}
