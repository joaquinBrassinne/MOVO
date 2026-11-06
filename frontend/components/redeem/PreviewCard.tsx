import { GiftIcon } from "@/components/benefits/icons";
import { ClockIcon } from "@/components/socios/icons";
import { benefitHeadline, initialOf, timeLeft } from "@/lib/benefits/format";
import { describeReturning, olderRedemptions, problemOfStatus } from "@/lib/benefits/redeem";
import { BENEFIT_TYPE, COUPON_STATUS, type CouponPreview } from "@/lib/benefits/types";

interface Props {
  preview: CouponPreview;
  now: Date;
}

const DATE_TIME: Intl.DateTimeFormatOptions = {
  day: "numeric",
  month: "short",
  hour: "2-digit",
  minute: "2-digit",
};

/** El cupón que se está por canjear: quién es, qué vale, bajo qué condiciones y hasta cuándo. */
export function PreviewCard({ preview, now }: Props) {
  const blocked = problemOfStatus(preview.status);
  const expires = new Date(preview.expires_at);
  const { benefit } = preview;
  const older = olderRedemptions(preview);

  return (
    <section aria-label="Cupón" className="space-y-4 rounded-3xl bg-white p-5 shadow-sm">
      <div className="flex items-center gap-3">
        <span
          aria-hidden="true"
          className="grid size-14 shrink-0 place-items-center rounded-full bg-movo-accent font-display text-2xl font-extrabold"
        >
          {initialOf(preview.customer_name)}
        </span>
        <div className="min-w-0">
          <p className="text-xs font-bold tracking-wide text-movo-slate uppercase">Cliente</p>
          <p className="truncate font-display text-2xl leading-tight font-extrabold">
            {preview.customer_name}
          </p>
        </div>
      </div>

      <div className="rounded-2xl bg-movo-cream p-4">
        <span className="inline-flex items-center gap-1.5 rounded-full bg-movo-accent px-3 py-1.5 text-sm font-extrabold">
          {benefit.type === BENEFIT_TYPE.CUSTOM ? <GiftIcon className="size-4" /> : null}
          {benefit.short_label}
        </span>
        <p className="mt-3 font-display text-2xl leading-tight font-bold">
          {benefitHeadline(benefit)}
        </p>
        {benefit.conditions ? (
          <p className="mt-3 rounded-xl bg-movo-accent/30 px-3 py-2 text-sm font-semibold">
            Condiciones: {benefit.conditions}
          </p>
        ) : null}
      </div>

      {blocked ? (
        <p role="alert" className="rounded-2xl bg-movo-danger/10 p-4 text-sm">
          <strong className="block text-base text-movo-danger">{blocked.title}</strong>
          {preview.status === COUPON_STATUS.REDEEMED && preview.redeemed_at
            ? `Se canjeó el ${new Date(preview.redeemed_at).toLocaleString("es-AR", DATE_TIME)}.`
            : blocked.body}
        </p>
      ) : (
        <p className="flex items-center gap-2 text-sm font-semibold">
          <ClockIcon className="size-4 shrink-0" />
          {timeLeft(preview.expires_at, now).replace(/^./, (c) => c.toUpperCase())} ·{" "}
          {expires.toLocaleString("es-AR", DATE_TIME)}
        </p>
      )}

      <div className="space-y-2">
        <p className="text-sm text-movo-slate">
          {describeReturning(preview.customer_redeemed_count)}
        </p>
        {preview.previous_redemptions.length > 0 ? (
          <ul aria-label="Cupones que canjeó antes" className="space-y-1.5 text-sm">
            {preview.previous_redemptions.map((past, i) => (
              <li
                key={`${past.redeemed_at}-${i}`}
                className="flex items-baseline justify-between gap-3 rounded-xl bg-movo-cream px-3 py-2"
              >
                <span className="min-w-0 truncate font-semibold">
                  {benefitHeadline(past.benefit)}
                </span>
                <span className="shrink-0 text-xs text-movo-slate">
                  {past.branch_name} ·{" "}
                  {new Date(past.redeemed_at).toLocaleString("es-AR", DATE_TIME)}
                </span>
              </li>
            ))}
            {older > 0 ? (
              <li className="px-3 text-xs text-movo-slate">
                y {older} {older === 1 ? "canje más antiguo" : "canjes más antiguos"}
              </li>
            ) : null}
          </ul>
        ) : null}
      </div>
    </section>
  );
}
