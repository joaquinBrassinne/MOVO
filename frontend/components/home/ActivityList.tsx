import { CheckCircleIcon } from "@/components/socios/icons";
import { TicketIcon } from "@/components/benefits/icons";
import { benefitWorth, initialOf, relativeTime, shortName } from "@/lib/benefits/format";
import { ACTIVITY_TYPE, type ActivityItem } from "@/lib/benefits/types";

interface Props {
  items: ActivityItem[];
  now: Date;
}

/** Una frase por hecho: quién generó o canjeó qué. Nunca lleva el teléfono del cliente. */
export function describeActivity(item: ActivityItem): string {
  const who = shortName(item.customer_name) || "Un cliente";
  const worth = benefitWorth(item.benefit);
  return item.type === ACTIVITY_TYPE.REDEEMED
    ? `${who} canjeó su cupón de ${worth}`
    : `${who} generó un cupón de ${worth}`;
}

/** La lista de "Actividad reciente": cupones generados y canjeados, del más nuevo al más viejo. */
export function ActivityList({ items, now }: Props) {
  return (
    <ul className="divide-y divide-movo-sand">
      {items.map((item) => {
        const redeemed = item.type === ACTIVITY_TYPE.REDEEMED;
        return (
          <li key={`${item.type}-${item.at}-${item.customer_name}`} className="flex gap-3 py-3">
            <span
              aria-hidden="true"
              className={`grid size-10 shrink-0 place-items-center rounded-full text-sm font-extrabold ${
                redeemed ? "bg-movo-ok/15 text-movo-ok" : "bg-movo-sand"
              }`}
            >
              {redeemed ? (
                <CheckCircleIcon className="size-5" />
              ) : (
                <TicketIcon className="size-5" />
              )}
            </span>
            <div className="min-w-0 flex-1">
              <p className="leading-snug font-semibold">
                <span className="mr-1 inline-grid size-5 place-items-center rounded-full bg-movo-accent/40 align-middle text-[10px] font-extrabold">
                  {initialOf(item.customer_name)}
                </span>
                {describeActivity(item)}
              </p>
              <p className="text-sm text-movo-slate">{relativeTime(item.at, now)}</p>
            </div>
          </li>
        );
      })}
    </ul>
  );
}
