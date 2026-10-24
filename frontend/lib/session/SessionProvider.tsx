"use client";

import { createContext, useContext, useEffect, useState, type ReactNode } from "react";

import * as api from "./client";
import type { SessionPayload, SessionResult, SessionUser } from "./types";

type SessionState =
  | { status: "loading" }
  | { status: "anonymous" }
  | { status: "authenticated"; user: SessionUser; accessToken: string; accessExpiresAt: string };

type Session = {
  state: SessionState;
  login: (email: string, password: string) => Promise<SessionResult>;
  register: (email: string, password: string) => Promise<SessionResult>;
  logout: () => Promise<void>;
};

const SessionContext = createContext<Session | null>(null);

function authenticated(data: SessionPayload): SessionState {
  return {
    status: "authenticated",
    user: data.user,
    accessToken: data.access_token,
    accessExpiresAt: data.access_expires_at,
  };
}

/** Mantiene la sesión del portal de socios. El access token vive solo en memoria. */
export function SessionProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<SessionState>({ status: "loading" });

  // Al cargar, la cookie httpOnly decide si ya hay sesión.
  useEffect(() => {
    let ignore = false;

    void api.refresh().then((result) => {
      if (!ignore) setState(result.ok ? authenticated(result.data) : { status: "anonymous" });
    });

    return () => {
      ignore = true;
    };
  }, []);

  async function enter(request: Promise<SessionResult>): Promise<SessionResult> {
    const result = await request;
    if (result.ok) setState(authenticated(result.data));
    return result;
  }

  const session: Session = {
    state,
    login: (email, password) => enter(api.login(email, password)),
    register: (email, password) => enter(api.register(email, password)),
    logout: async () => {
      await api.logout();
      setState({ status: "anonymous" });
    },
  };

  return <SessionContext value={session}>{children}</SessionContext>;
}

export function useSession(): Session {
  const session = useContext(SessionContext);
  if (!session) throw new Error("useSession must be used inside <SessionProvider>");
  return session;
}
