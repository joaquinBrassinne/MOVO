"use client";

import { useState } from "react";

import { addBranch, editBranch, geocode } from "./api";
import { describeFailure, type Feedback } from "./messages";
import type { Branch } from "./types";

export interface Pin {
  latitude: number;
  longitude: number;
}

/** A dónde mira el mapa. `nonce` cambia con cada pedido para que "Recentrar" repita el movimiento. */
export interface MapView extends Pin {
  nonce: number;
}

/** Buenos Aires: dónde arranca el mapa mientras no hay pin. */
export const DEFAULT_VIEW: Pin = { latitude: -34.6037, longitude: -58.3816 };

/** El alta del negocio no pregunta el nombre de la primera sede: arranca con este. */
export const DEFAULT_BRANCH_NAME = "Sede principal";

/** Estado y envío del paso 2: la sede, con su pin en el mapa. */
export function useBranchForm(existing: Branch | null, defaultName = DEFAULT_BRANCH_NAME) {
  const [name, setName] = useState(existing?.name ?? defaultName);
  const [address, setAddress] = useState(existing?.address ?? "");
  const [city, setCity] = useState(existing?.city ?? "");
  const [reference, setReference] = useState(existing?.reference ?? "");
  const [bookingUrl, setBookingUrl] = useState(existing?.booking_url ?? "");
  const [pin, setPin] = useState<Pin | null>(
    existing ? { latitude: existing.latitude, longitude: existing.longitude } : null,
  );
  const [view, setView] = useState<MapView>({ ...(pin ?? DEFAULT_VIEW), nonce: 0 });
  const [searching, setSearching] = useState(false);
  const [pending, setPending] = useState(false);
  const [feedback, setFeedback] = useState<Feedback>({ fields: {} });

  function showPin(next: Pin) {
    setPin(next);
    setView((current) => ({ ...next, nonce: current.nonce + 1 }));
  }

  /** Busca la dirección escrita y pone el pin ahí. Una sola búsqueda por pulsación, nunca al tipear. */
  async function search() {
    if (searching) return;
    if (address.trim() === "" || city.trim() === "") {
      setFeedback({
        fields: {
          ...(address.trim() === "" ? { address: "Ingresá la dirección" } : {}),
          ...(city.trim() === "" ? { city: "Ingresá la ciudad" } : {}),
        },
      });
      return;
    }

    setSearching(true);
    setFeedback({ fields: {} });
    const result = await geocode(address, city);
    setSearching(false);

    if (result.ok) showPin(result.data);
    else setFeedback(describeFailure(result));
  }

  /** El pin va a la ubicación del dispositivo (sin geocodificación inversa: la dirección la escribe el dueño). */
  function locateMe() {
    if (!("geolocation" in navigator)) {
      setFeedback({
        fields: {},
        form: { title: "Tu navegador no da la ubicación", body: "Ubicá el pin a mano en el mapa." },
      });
      return;
    }

    navigator.geolocation.getCurrentPosition(
      ({ coords }) => {
        setFeedback({ fields: {} });
        showPin({ latitude: coords.latitude, longitude: coords.longitude });
      },
      () =>
        setFeedback({
          fields: {},
          form: {
            title: "No pudimos obtener tu ubicación",
            body: "Revisá los permisos del navegador o ubicá el pin a mano.",
          },
        }),
      { enableHighAccuracy: true, timeout: 10_000 },
    );
  }

  function recenter() {
    if (pin) showPin(pin);
  }

  /** Valida, guarda (crea o actualiza) y dice si se pudo. */
  async function submit(): Promise<boolean> {
    if (pending) return false;

    const fields: Record<string, string> = {};
    if (name.trim() === "") fields.name = "Ingresá el nombre de la sede";
    if (address.trim() === "") fields.address = "Ingresá la dirección";
    if (city.trim() === "") fields.city = "Ingresá la ciudad";
    if (pin === null) fields.location = "Marcá la ubicación en el mapa";
    if (bookingUrl.trim() !== "" && !/^https?:\/\/\S+$/i.test(bookingUrl.trim())) {
      fields.booking_url = "Ingresá un link que empiece con https://";
    }
    if (Object.keys(fields).length > 0 || pin === null) {
      setFeedback({ fields });
      return false;
    }

    setPending(true);
    setFeedback({ fields: {} });

    const input = {
      name: name.trim(),
      address,
      city,
      latitude: pin.latitude,
      longitude: pin.longitude,
      booking_url: bookingUrl.trim(),
      reference: reference.trim(),
    };
    const result = await (existing ? editBranch(existing.id, input) : addBranch(input));
    setPending(false);

    if (result.ok) return true;
    setFeedback(describeFailure(result));
    return false;
  }

  return {
    address,
    setAddress,
    city,
    setCity,
    bookingUrl,
    name,
    setName,
    reference,
    setReference,
    setBookingUrl,
    pin,
    movePin: setPin,
    view,
    searching,
    pending,
    feedback,
    search,
    locateMe,
    recenter,
    submit,
  };
}
