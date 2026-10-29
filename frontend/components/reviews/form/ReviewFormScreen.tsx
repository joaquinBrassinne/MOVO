"use client";

import Link from "next/link";
import { useId } from "react";

import {
  ArrowLeftIcon,
  ArrowRightIcon,
  CheckCircleIcon,
  ClockIcon,
  InfoIcon,
  MapPinIcon,
} from "@/components/socios/icons";
import { StarRating } from "@/components/reviews/StarRating";
import { displayCode } from "@/lib/benefits/code";
import { useReviewForm } from "@/lib/reviews/useReviewForm";
import type { CreatedReview, ReviewFormContext } from "@/lib/reviews/types";

import { StarPicker } from "./StarPicker";
import {
  ANONYMOUS_NAME,
  COMMENT_MAX,
  commentLength,
  formatDay,
  placeLine,
  ratingLabel,
} from "@/lib/reviews/reviewFormLogic";

const CARD = "space-y-4 rounded-3xl bg-movo-cream p-6 text-movo-ink";

/**
 * "Dejá tu reseña": the customer's review of a redeemed coupon, opened with the access
 * token of the coupon (there is no account). Renders by the state the API reports.
 */
export function ReviewFormScreen({ token }: { token: string }) {
  const form = useReviewForm(token);
  const { view } = form;
  const couponHref = `/cupon/${token}`;

  return (
    <div className="min-h-dvh bg-movo-page font-sans text-white">
      <header className="mx-auto flex h-16 max-w-md items-center gap-3 px-5">
        <Link
          href={couponHref}
          aria-label="Volver a mi cupón"
          className="grid size-11 place-items-center rounded-full outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
        >
          <ArrowLeftIcon />
        </Link>
        <p className="font-display text-lg font-extrabold">Dejá Tu Reseña</p>
      </header>

      <main className="mx-auto max-w-md space-y-5 px-5 pb-10">
        {view.status === "loading" ? (
          <div
            aria-busy="true"
            aria-label="Cargando el formulario"
            className="h-96 animate-pulse rounded-3xl bg-movo-card"
          />
        ) : null}

        {view.status === "missing" ? (
          <Message
            title="No encontramos este cupón"
            body="El enlace puede estar incompleto. Si lo guardaste en este dispositivo, buscalo en tus cupones."
            href="/cupones"
            action="Ver mis cupones"
          />
        ) : null}
        {view.status === "failed" ? (
          <Message
            title="No pudimos cargar el formulario"
            body="Revisá tu conexión y probá de nuevo en unos minutos."
            href={couponHref}
            action="Volver a mi cupón"
          />
        ) : null}

        {view.status === "ready" && view.state === "open" ? (
          <>
            <PlaceCard context={view.context} />
            <FormCard context={view.context} form={form} />
          </>
        ) : null}
        {view.status === "ready" && view.state === "reviewed" ? (
          <>
            <PlaceCard context={view.context} />
            <Reviewed href={couponHref} />
          </>
        ) : null}
        {view.status === "ready" && view.state === "window_closed" ? (
          <Message
            title="Ya pasaron los 30 días"
            body="El plazo para dejar tu reseña es de 30 días desde el canje y ya terminó."
            href={couponHref}
            action="Volver a mi cupón"
          />
        ) : null}
        {view.status === "ready" && view.state === "not_redeemed" ? (
          <Message
            title="Primero canjeá tu cupón"
            body="Podés dejar tu reseña después de canjear el cupón en el local."
            href={couponHref}
            action="Volver a mi cupón"
          />
        ) : null}

        {view.status === "confirmed" ? (
          <>
            <PlaceCard context={view.context} />
            <Confirmed review={view.review} href={couponHref} />
          </>
        ) : null}
      </main>
    </div>
  );
}

function Message({
  title,
  body,
  href,
  action,
}: {
  title: string;
  body: string;
  href: string;
  action: string;
}) {
  return (
    <section role="alert" className="space-y-3 rounded-3xl bg-movo-card p-6 text-center">
      <h1 className="font-display text-2xl font-extrabold">{title}</h1>
      <p className="text-movo-muted">{body}</p>
      <Link
        href={href}
        className="inline-flex h-12 items-center rounded-full bg-movo-accent px-6 font-bold text-movo-ink outline-none focus-visible:ring-2 focus-visible:ring-white"
      >
        {action}
      </Link>
    </section>
  );
}

function PlaceCard({ context }: { context: ReviewFormContext }) {
  const place = placeLine(context.branch_name, context.city);
  return (
    <section aria-label="Tu canje" className={CARD}>
      <h1 className="font-display text-2xl leading-tight font-extrabold">
        {context.business_name}
      </h1>
      {place ? (
        <p className="-mt-2 flex items-center gap-1.5 text-sm text-movo-slate">
          <MapPinIcon className="size-4" />
          {place}
        </p>
      ) : null}
      <div className="flex flex-wrap gap-2">
        <p className="inline-flex items-center gap-1.5 rounded-full bg-movo-accent px-3 py-1.5 text-xs font-extrabold">
          <CheckCircleIcon className="size-4" />
          Reseña verificada por canje
        </p>
        <p className="inline-flex items-center gap-1.5 rounded-full bg-movo-sand px-3 py-1.5 text-xs font-bold text-movo-slate">
          Canjeado {formatDay(context.redeemed_at)} · {displayCode(context.code)}
        </p>
      </div>
    </section>
  );
}

function FormCard({
  context,
  form,
}: {
  context: ReviewFormContext;
  form: ReturnType<typeof useReviewForm>;
}) {
  const ids = useId();
  const { rating, comment, showName, submitting, errors } = form;
  const shownAs = context.display_name || ANONYMOUS_NAME;
  const until = formatDay(context.review_until);

  return (
    <form
      noValidate
      className={CARD}
      onSubmit={(event) => {
        event.preventDefault();
        void form.submit();
      }}
    >
      <div className="space-y-1 text-center">
        <h2 className="font-display text-2xl leading-tight font-extrabold">
          ¿Cómo fue tu experiencia en {context.business_name}?
        </h2>
        <p className="text-sm text-movo-slate">
          Calificá tu visita para ayudar a otros usuarios de MOVO
        </p>
      </div>

      <div className="space-y-2 text-center">
        <StarPicker
          value={rating}
          onChange={form.setRating}
          invalid={Boolean(errors.rating)}
          describedBy={errors.rating ? `${ids}-rating` : undefined}
        />
        <p
          aria-live="polite"
          className="mx-auto min-h-8 w-fit rounded-full bg-movo-sand px-3 py-1.5 text-xs font-bold"
        >
          {rating ? (
            <>
              <span aria-hidden="true" className="mr-1.5 text-movo-accent">
                ●
              </span>
              {ratingLabel(rating)}
            </>
          ) : null}
        </p>
        {errors.rating ? (
          <p id={`${ids}-rating`} role="alert" className="text-sm font-semibold text-movo-danger">
            {errors.rating}
          </p>
        ) : null}
      </div>

      <div className="space-y-2">
        <div className="flex items-baseline justify-between">
          <label htmlFor={`${ids}-comment`} className="text-sm font-bold">
            ¿Querés sumar un comentario?{" "}
            <span className="font-normal text-movo-slate">(opcional)</span>
          </label>
          <span aria-hidden="true" className="text-xs font-bold text-movo-slate">
            {commentLength(comment)}/{COMMENT_MAX}
          </span>
        </div>
        <textarea
          id={`${ids}-comment`}
          value={comment}
          maxLength={COMMENT_MAX}
          rows={4}
          placeholder="Contanos qué te gustó (opcional)"
          aria-invalid={errors.comment ? true : undefined}
          aria-describedby={errors.comment ? `${ids}-comment-error` : undefined}
          onChange={(event) => form.setComment(event.target.value)}
          className="w-full resize-none rounded-2xl bg-movo-sand p-4 text-sm outline-none placeholder:text-movo-slate focus-visible:ring-2 focus-visible:ring-movo-ink"
        />
        {errors.comment ? (
          <p
            id={`${ids}-comment-error`}
            role="alert"
            className="text-sm font-semibold text-movo-danger"
          >
            {errors.comment}
          </p>
        ) : null}
      </div>

      <div className="space-y-1.5 rounded-2xl bg-movo-sand p-4">
        <div className="flex items-center justify-between gap-3">
          <label htmlFor={`${ids}-name`} className="text-sm font-extrabold">
            Mostrar mi nombre
          </label>
          <button
            id={`${ids}-name`}
            type="button"
            role="switch"
            aria-checked={showName}
            aria-describedby={`${ids}-name-hint`}
            onClick={() => form.setShowName(!showName)}
            className={`relative h-8 w-14 shrink-0 rounded-full outline-none focus-visible:ring-2 focus-visible:ring-movo-ink focus-visible:ring-offset-2 ${
              showName ? "bg-movo-ink" : "bg-movo-slate/40"
            }`}
          >
            <span
              aria-hidden="true"
              className={`absolute top-1 size-6 rounded-full bg-movo-accent transition-all ${
                showName ? "left-7" : "left-1 bg-white"
              }`}
            />
          </button>
        </div>
        <p id={`${ids}-name-hint`} className="text-xs text-movo-slate">
          Se va a mostrar como <strong>{shownAs}</strong>. Si lo apagás, aparece como{" "}
          <strong>{ANONYMOUS_NAME}</strong>.
        </p>
      </div>

      {errors.form ? (
        <p role="alert" className="text-center text-sm font-semibold text-movo-danger">
          {errors.form}
        </p>
      ) : null}

      <button
        type="submit"
        disabled={submitting}
        className="flex h-14 w-full items-center justify-center gap-2 rounded-full bg-movo-accent font-display text-lg font-bold text-movo-ink outline-none focus-visible:ring-2 focus-visible:ring-movo-ink focus-visible:ring-offset-2 disabled:opacity-60"
      >
        {submitting ? "Publicando…" : "Publicar reseña"}
        {submitting ? null : <ArrowRightIcon className="size-5" />}
      </button>

      <p className="flex items-start gap-2 text-xs text-movo-slate">
        <InfoIcon className="mt-0.5 size-4 shrink-0" />
        {until
          ? `Podés dejar tu reseña hasta el ${until}, 30 días después de canjear.`
          : "Podés dejar tu reseña hasta 30 días después de canjear."}
      </p>
    </form>
  );
}

function Confirmed({ review, href }: { review: CreatedReview; href: string }) {
  return (
    <section aria-live="polite" className={`${CARD} text-center`}>
      <p className="mx-auto grid size-14 place-items-center rounded-full bg-movo-accent">
        <CheckCircleIcon className="size-8" />
      </p>
      <h2 className="font-display text-2xl font-extrabold">¡Gracias por tu reseña!</h2>
      <p className="text-sm text-movo-slate">
        Ya está publicada y ayuda a otros usuarios de MOVO a elegir.
      </p>
      <div className="space-y-2 rounded-2xl bg-movo-sand p-4 text-left">
        <StarRating
          average={review.rating}
          count={Number.MAX_SAFE_INTEGER}
          starClassName="size-5"
        />
        {review.comment ? <p className="text-sm">{review.comment}</p> : null}
        <p className="text-xs font-bold text-movo-slate">Publicada como {review.display_name}</p>
      </div>
      <BackLink href={href} />
    </section>
  );
}

function Reviewed({ href }: { href: string }) {
  return (
    <section className={`${CARD} text-center`}>
      <p className="mx-auto grid size-14 place-items-center rounded-full bg-movo-sand">
        <ClockIcon className="size-7" />
      </p>
      <h2 className="font-display text-2xl font-extrabold">Ya dejaste tu reseña</h2>
      <p className="text-sm text-movo-slate">
        Cada canje tiene una sola reseña. ¡Gracias por contarnos tu experiencia!
      </p>
      <BackLink href={href} />
    </section>
  );
}

function BackLink({ href }: { href: string }) {
  return (
    <Link
      href={href}
      className="inline-flex h-12 items-center rounded-full bg-movo-ink px-6 font-bold text-white outline-none focus-visible:ring-2 focus-visible:ring-movo-ink focus-visible:ring-offset-2"
    >
      Volver a mi cupón
    </Link>
  );
}
