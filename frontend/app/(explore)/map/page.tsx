import { redirect } from "next/navigation";

/** El mapa es la home: se conserva la ruta vieja para los enlaces que ya circulan. */
export default function MapPage() {
  redirect("/");
}
