// La última sede en la que el dueño canjeó, para no pedírsela de nuevo en cada canje. Vive en
// este dispositivo: es una comodidad, no un dato. Si el navegador no deja guardarla, se elige
// de nuevo cada vez.

const KEY = "movo.redeem.lastBranch";

/** El id de la última sede elegida, como texto; vacío si no hay. */
export function readLastBranch(): string {
  try {
    return window.localStorage.getItem(KEY) ?? "";
  } catch {
    return "";
  }
}

export function writeLastBranch(id: string): void {
  try {
    window.localStorage.setItem(KEY, id);
  } catch {
    // Sin almacenamiento: la sede vale solo mientras la página está abierta.
  }
}
