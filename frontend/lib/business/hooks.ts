"use client";

import { useEffect, useState } from "react";

import type { ApiError } from "@/lib/session/types";

import { getMyBusiness, listCategories, listServices } from "./api";
import type { Category, MyBusiness, Service } from "./types";

export type Loaded<T> =
  | { status: "loading" }
  | { status: "ready"; data: T }
  | { status: "missing" }
  | { status: "failed"; error: ApiError };

interface Resource<T> {
  state: Loaded<T>;
  /** Vuelve a pedir los datos, por ejemplo después de guardar. */
  reload: () => void;
}

/** El negocio del dueño con sedes, horarios y fotos. `missing` si todavía no lo creó. */
export function useMyBusiness(): Resource<MyBusiness> {
  const [version, setVersion] = useState(0);
  const [state, setState] = useState<Loaded<MyBusiness>>({ status: "loading" });

  useEffect(() => {
    let ignore = false;

    void getMyBusiness().then((result) => {
      if (ignore) return;
      if (result.ok) setState({ status: "ready", data: result.data });
      else if (result.error.code === "business_not_found") setState({ status: "missing" });
      else setState({ status: "failed", error: result.error });
    });

    return () => {
      ignore = true;
    };
  }, [version]);

  return { state, reload: () => setVersion((current) => current + 1) };
}

/** Los servicios del negocio, activos o no. */
export function useServices(): Resource<Service[]> {
  const [version, setVersion] = useState(0);
  const [state, setState] = useState<Loaded<Service[]>>({ status: "loading" });

  useEffect(() => {
    let ignore = false;

    void listServices().then((result) => {
      if (ignore) return;
      if (result.ok) setState({ status: "ready", data: result.data.services });
      else if (result.error.code === "business_not_found") setState({ status: "missing" });
      else setState({ status: "failed", error: result.error });
    });

    return () => {
      ignore = true;
    };
  }, [version]);

  return { state, reload: () => setVersion((current) => current + 1) };
}

/** El catálogo de categorías (público, cambia solo con una migración). */
export function useCategories(): Loaded<Category[]> {
  const [state, setState] = useState<Loaded<Category[]>>({ status: "loading" });

  useEffect(() => {
    let ignore = false;

    void listCategories().then((result) => {
      if (ignore) return;
      setState(
        result.ok
          ? { status: "ready", data: result.data.categories }
          : { status: "failed", error: result.error },
      );
    });

    return () => {
      ignore = true;
    };
  }, []);

  return state;
}
