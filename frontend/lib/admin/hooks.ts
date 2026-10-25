"use client";

import { useEffect, useState } from "react";

import type { Loaded } from "@/lib/business/hooks";

import { getAdminBusiness, listAdminBusinesses } from "./api";
import type { AdminDetail, AdminList, DirectoryFilters } from "./types";

interface Resource<T> {
  state: Loaded<T>;
  /** Vuelve a pedir los datos, por ejemplo después de una acción del admin. */
  reload: () => void;
}

/** La página del directorio de negocios que piden los filtros. */
export function useAdminDirectory(filters: DirectoryFilters): Resource<AdminList> {
  const [version, setVersion] = useState(0);
  const [state, setState] = useState<Loaded<AdminList>>({ status: "loading" });
  const { status, query, categoryId, page } = filters;

  useEffect(() => {
    let ignore = false;

    void listAdminBusinesses({ status, query, categoryId, page }).then((result) => {
      if (ignore) return;
      setState(
        result.ok
          ? { status: "ready", data: result.data }
          : { status: "failed", error: result.error },
      );
    });

    return () => {
      ignore = true;
    };
  }, [status, query, categoryId, page, version]);

  return { state, reload: () => setVersion((current) => current + 1) };
}

/** El detalle del negocio elegido; `null` mientras no se eligió ninguno. */
export function useAdminDetail(id: number | null): Resource<AdminDetail> {
  const [version, setVersion] = useState(0);
  const [state, setState] = useState<Loaded<AdminDetail>>({ status: "loading" });

  useEffect(() => {
    if (id === null) return;
    let ignore = false;

    void getAdminBusiness(id).then((result) => {
      if (ignore) return;
      setState(
        result.ok
          ? { status: "ready", data: result.data }
          : { status: "failed", error: result.error },
      );
    });

    return () => {
      ignore = true;
    };
  }, [id, version]);

  return { state, reload: () => setVersion((current) => current + 1) };
}
