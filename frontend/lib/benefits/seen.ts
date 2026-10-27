// La marca de "ya vi la actividad hasta acá": el momento del último hecho que el dueño vio
// en el panel. Vive en este navegador (es una comodidad, no un dato: si falta, no hay
// novedades que contar) y nunca lleva datos de clientes.

const KEY = "movo.activity.seenAt";

/** La marca guardada en RFC 3339, o `undefined` si no hay o el navegador no deja leerla. */
export function readSeenAt(): string | undefined {
  try {
    const stored = window.localStorage.getItem(KEY);
    return stored && !Number.isNaN(Date.parse(stored)) ? stored : undefined;
  } catch {
    return undefined;
  }
}

export function writeSeenAt(iso: string): void {
  try {
    window.localStorage.setItem(KEY, iso);
  } catch {
    // Sin almacenamiento (modo privado, bloqueado): la marca vale solo mientras la página está abierta.
  }
}
