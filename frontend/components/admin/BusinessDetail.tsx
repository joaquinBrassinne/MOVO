"use client";

import { useState, type ReactNode } from "react";

import { Notice } from "@/components/onboarding/fields";
import { Switch } from "@/components/panel/Switch";
import { MapPinIcon, XIcon } from "@/components/socios/icons";
import { applyAdminAction } from "@/lib/admin/api";
import { useAdminDetail } from "@/lib/admin/hooks";
import { describeAdminFailure, describeAuditAction } from "@/lib/admin/messages";
import { formatDate } from "@/lib/admin/subscriptions/format";
import { useSubscription } from "@/lib/admin/subscriptions/hooks";
import { ADMIN_ACTION, type AdminAction } from "@/lib/admin/types";
import type { FormMessage } from "@/lib/business/messages";
import { BUSINESS_STATUS } from "@/lib/business/types";

import { ConfirmPaymentForm } from "./ConfirmPaymentForm";
import { StatusPill } from "./StatusPill";
import { PLAN_NAME } from "./SubscriptionPills";

interface Props {
  id: number;
  onClose: () => void;
  /** Se llama después de una acción, para que el listado se actualice. */
  onChanged: () => void;
}

/** El detalle de un local con las acciones del admin, sus datos de contacto, el pago y el historial. */
export function BusinessDetail({ id, onClose, onChanged }: Props) {
  const detail = useAdminDetail(id);
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState<FormMessage | null>(null);

  async function act(action: AdminAction) {
    if (pending) return;
    setPending(true);
    setMessage(null);

    const result = await applyAdminAction(id, action);
    if (!result.ok) setMessage(describeAdminFailure(result));

    setPending(false);
    detail.reload();
    onChanged();
  }

  function refresh() {
    detail.reload();
    onChanged();
  }

  if (detail.state.status === "loading") {
    return <aside className="rounded-3xl bg-white p-6 text-movo-slate shadow-sm">Cargando…</aside>;
  }
  if (detail.state.status !== "ready") {
    return (
      <aside className="rounded-3xl bg-white p-6 shadow-sm">
        <Notice title="No pudimos cargar este local" body="Cerrá el detalle y volvé a elegirlo." />
        <button type="button" onClick={onClose} className="mt-4 text-sm font-bold underline">
          Cerrar
        </button>
      </aside>
    );
  }

  const { business, branches, services_total, services_active, history } = detail.state.data;
  const status = business.status;
  const isDeleted = status === BUSINESS_STATUS.DELETED;

  return (
    <aside
      className="space-y-5 rounded-3xl bg-white p-6 shadow-sm"
      aria-label={`Detalle de ${business.name}`}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="flex items-center gap-2 text-xs font-bold tracking-wide text-movo-slate uppercase">
            Auditoría y alta <StatusPill status={status} />
          </p>
          <h2 className="mt-2 truncate font-display text-2xl font-extrabold">{business.name}</h2>
          <p className="text-sm text-movo-slate">ID #{business.id}</p>
        </div>
        <button
          type="button"
          aria-label="Cerrar el detalle"
          onClick={onClose}
          className="grid size-10 shrink-0 place-items-center rounded-full bg-movo-sand outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
        >
          <XIcon />
        </button>
      </div>

      {message ? <Notice title={message.title} body={message.body} /> : null}

      <section className="space-y-3 rounded-2xl bg-movo-cream p-4">
        {status === BUSINESS_STATUS.PENDING_REVIEW && business.pilot ? (
          <ActionButton primary disabled={pending} onClick={() => void act(ADMIN_ACTION.APPROVE)}>
            Aprobar comercio y publicar
          </ActionButton>
        ) : null}
        {status === BUSINESS_STATUS.PENDING_REVIEW && !business.pilot ? (
          <ConfirmPaymentForm
            businessId={id}
            submitLabel="Confirmar pago y publicar"
            onConfirmed={refresh}
          />
        ) : null}
        {status === BUSINESS_STATUS.ACTIVE ? (
          <ActionButton disabled={pending} onClick={() => void act(ADMIN_ACTION.PAUSE)}>
            Pausar el local
          </ActionButton>
        ) : null}
        {status === BUSINESS_STATUS.ACTIVE && !business.pilot ? (
          <ActiveBilling businessId={id} onConfirmed={refresh} />
        ) : null}
        {status === BUSINESS_STATUS.PAUSED ? (
          <ActionButton
            primary
            disabled={pending}
            onClick={() => void act(ADMIN_ACTION.REACTIVATE)}
          >
            Reactivar el local
          </ActionButton>
        ) : null}
        {status === BUSINESS_STATUS.DRAFT ? (
          <p className="text-sm text-movo-slate">El dueño todavía no lo envió a revisión.</p>
        ) : null}
        {isDeleted ? (
          <p className="text-sm text-movo-slate">Este local está dado de baja.</p>
        ) : null}

        {isDeleted ? null : (
          <>
            <div className="grid grid-cols-2 gap-4 pt-1">
              <label className="flex items-center justify-between gap-2 text-sm font-semibold">
                Insignia verificado
                <Switch
                  checked={business.verified}
                  label="Insignia verificado"
                  disabled={pending}
                  onChange={(next) =>
                    void act(next ? ADMIN_ACTION.VERIFY : ADMIN_ACTION.REVOKE_VERIFICATION)
                  }
                />
              </label>
              <label className="flex items-center justify-between gap-2 text-sm font-semibold">
                Acceso piloto
                <Switch
                  checked={business.pilot}
                  label="Acceso piloto"
                  disabled={pending}
                  onChange={(next) =>
                    void act(next ? ADMIN_ACTION.MARK_PILOT : ADMIN_ACTION.UNMARK_PILOT)
                  }
                />
              </label>
            </div>
            <button
              type="button"
              disabled={pending}
              onClick={() => {
                if (window.confirm(`¿Dar de baja "${business.name}"? No se puede deshacer.`)) {
                  void act(ADMIN_ACTION.DELETE);
                }
              }}
              className="text-sm font-bold text-movo-danger underline underline-offset-4 disabled:opacity-60"
            >
              Dar de baja
            </button>
          </>
        )}
      </section>

      <Section title="Datos de contacto y sede">
        <p className="text-sm">
          <a
            href={`mailto:${business.owner_email}`}
            className="font-semibold underline underline-offset-4"
          >
            {business.owner_email}
          </a>
        </p>
        <a
          href={`https://wa.me/${business.whatsapp}`}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex rounded-full bg-movo-ok/15 px-4 py-2 text-sm font-bold text-movo-ok"
        >
          Abrir WhatsApp (+{business.whatsapp})
        </a>

        {branches.length === 0 ? (
          <p className="text-sm text-movo-slate">Todavía no cargó ninguna sede.</p>
        ) : (
          <ul className="space-y-2">
            {branches.map((branch) => (
              <li key={branch.id} className="flex items-start gap-2 text-sm">
                <MapPinIcon className="mt-0.5 size-4 shrink-0 text-movo-gold" />
                <span>
                  {branch.address}, {branch.city}
                  {branch.reference ? ` (${branch.reference})` : ""}
                  {branch.booking_url ? (
                    <>
                      {" · "}
                      <a
                        href={branch.booking_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="font-semibold underline underline-offset-4"
                      >
                        Reservas
                      </a>
                    </>
                  ) : null}
                </span>
              </li>
            ))}
          </ul>
        )}
        <p className="text-sm text-movo-slate">
          {business.category_name} · {services_active} de {services_total}{" "}
          {services_total === 1 ? "servicio activo" : "servicios activos"}
        </p>
      </Section>

      <Section title="Pago declarado">
        {business.mp_payer_email ? (
          <dl className="space-y-1 text-sm">
            <div>
              <dt className="inline text-movo-slate">Email en Mercado Pago: </dt>
              <dd className="inline font-semibold break-all">{business.mp_payer_email}</dd>
            </div>
            <div>
              <dt className="inline text-movo-slate">Nombre del titular: </dt>
              <dd className="inline font-semibold">{business.mp_payer_name}</dd>
            </div>
          </dl>
        ) : (
          <p className="text-sm text-movo-slate">El dueño todavía no declaró su pago.</p>
        )}
        {business.mp_plan_hint ? (
          <p className="text-sm">
            Pista de plan en la URL de retorno:{" "}
            <strong className="font-display text-base">{business.mp_plan_hint}</strong>
            <span className="block text-xs text-movo-slate">
              Es solo una pista: el plan se fija según el mail y el panel de Mercado Pago.
            </span>
          </p>
        ) : null}
        <ConfirmationChecklist
          key={business.id}
          legend={
            status === BUSINESS_STATUS.PENDING_REVIEW && !business.pilot
              ? "Antes de confirmar el pago"
              : "Antes de aprobar"
          }
        />
      </Section>

      <Section title="Historial de auditoría">
        {history.length === 0 ? (
          <p className="text-sm text-movo-slate">Todavía no hay acciones registradas.</p>
        ) : (
          <ol className="space-y-3">
            {history.map((entry) => (
              <li key={`${entry.at}-${entry.action}`} className="text-sm">
                <p className="font-semibold">{describeAuditAction(entry.action)}</p>
                <p className="text-xs text-movo-slate">
                  {entry.actor_email} · {new Date(entry.at).toLocaleString("es-AR")}
                </p>
              </li>
            ))}
          </ol>
        )}
      </Section>
    </aside>
  );
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="space-y-3">
      <h3 className="text-xs font-bold tracking-wide text-movo-slate uppercase">{title}</h3>
      <div className="space-y-3 rounded-2xl bg-movo-cream p-4">{children}</div>
    </section>
  );
}

const CONFIRMATION_CHECKS = [
  "Mail de Mercado Pago con el mismo email",
  "Plan y monto correctos",
  "Perfil revisado",
] as const;

/** Los tres chequeos antes de aprobar. Es una ayuda de pantalla: no se guarda ni bloquea la acción. */
function ConfirmationChecklist({ legend }: { legend: string }) {
  const [done, setDone] = useState<ReadonlySet<string>>(new Set());

  function toggle(label: string) {
    setDone((current) => {
      const next = new Set(current);
      if (!next.delete(label)) next.add(label);
      return next;
    });
  }

  return (
    <fieldset className="space-y-2 border-t border-movo-slate/20 pt-3">
      <legend className="text-xs font-bold text-movo-slate">{legend}</legend>
      {CONFIRMATION_CHECKS.map((label) => (
        <label key={label} className="flex items-center gap-3 text-sm font-semibold">
          <input
            type="checkbox"
            checked={done.has(label)}
            onChange={() => toggle(label)}
            className="size-5 accent-movo-ink"
          />
          {label}
        </label>
      ))}
    </fieldset>
  );
}

/**
 * An active, non-pilot business should have a subscription. Without one (rare, the API allows
 * it) the admin can still confirm the payment; with one, a one-line summary links to the list.
 */
function ActiveBilling({
  businessId,
  onConfirmed,
}: {
  businessId: number;
  onConfirmed: () => void;
}) {
  const subscription = useSubscription(businessId);
  const { state } = subscription;

  if (state.status === "loading") return <p className="text-sm text-movo-slate">Cargando pago…</p>;
  if (state.status === "missing") {
    return (
      <div className="space-y-2">
        <p className="text-sm font-bold">Confirmar pago</p>
        <p className="text-sm text-movo-slate">Este local está activo pero no tiene suscripción.</p>
        <ConfirmPaymentForm
          businessId={businessId}
          submitLabel="Confirmar pago"
          onConfirmed={() => {
            subscription.reload();
            onConfirmed();
          }}
        />
      </div>
    );
  }
  if (state.status === "failed") {
    return <p className="text-sm text-movo-slate">No pudimos cargar su suscripción.</p>;
  }

  const sub = state.data;
  return (
    <p className="text-sm">
      Plan {PLAN_NAME[sub.plan]} hasta el {formatDate(sub.paid_until)}.{" "}
      <a href="/admin/subscriptions" className="font-bold underline underline-offset-4">
        Ver suscripciones
      </a>
    </p>
  );
}

function ActionButton({
  primary,
  disabled,
  onClick,
  children,
}: {
  primary?: boolean;
  disabled: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      className={`h-14 w-full rounded-full font-display text-lg font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink disabled:opacity-60 ${
        primary ? "bg-movo-accent" : "bg-white text-movo-ink ring-1 ring-movo-slate/30"
      }`}
    >
      {children}
    </button>
  );
}
