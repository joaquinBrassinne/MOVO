"use client";

import { useEffect, useState } from "react";

import { resetPassword } from "./api";
import { checkPasswords, classifyResetFailure, describeResetFailure, type Message } from "./rules";

export type ResetStatus = "form" | "done" | "invalid";

/**
 * State of the owner's "choose a new password" page. The token is read once and then removed
 * from the address bar so it does not stay in the history or get copied by accident.
 */
export function useResetPassword(initialToken: string | null) {
  const [token] = useState(initialToken);
  const [password, setPassword] = useState("");
  const [repeat, setRepeat] = useState("");
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState<Message | null>(null);
  const [status, setStatus] = useState<ResetStatus>(initialToken ? "form" : "invalid");

  useEffect(() => {
    if (window.location.search.includes("token")) {
      window.history.replaceState(null, "", window.location.pathname);
    }
  }, []);

  const checks = checkPasswords(password, repeat);

  async function submit(event: { preventDefault: () => void }) {
    event.preventDefault();
    if (pending || !token || !checks.valid) return;
    setPending(true);
    setMessage(null);

    const result = await resetPassword(token, password);
    setPending(false);
    if (result.ok) {
      setPassword("");
      setRepeat("");
      setStatus("done");
      return;
    }

    const outcome = classifyResetFailure(result);
    if (outcome === "invalid_token") setStatus("invalid");
    else setMessage(describeResetFailure(outcome));
  }

  return { status, password, repeat, setPassword, setRepeat, checks, pending, message, submit };
}
