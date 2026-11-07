import Link from "next/link";

import { CoverTile } from "@/components/CoverImage";
import { Icon, VerifiedIcon } from "@/components/explorar/icons";
import { Logo } from "@/components/Logo";
import { StarRating } from "@/components/reviews/StarRating";
import { ArrowLeftIcon } from "@/components/socios/icons";
import { cn } from "@/lib/cn";
import { formatPrice } from "@/lib/business/price";
import { describeSchedule } from "@/lib/business/schedule";
import {
  describeOpenNow,
  groupHours,
  type Profile,
  type ProfileBranch,
} from "@/lib/search/profile";

import { PhotoSection } from "./PhotoSection";
import { ProfileBenefits } from "./ProfileBenefits";
import { ProfileTabs, type ProfileTab } from "./ProfileTabs";
import { ProfileViewBeacon } from "./ProfileViewBeacon";
import { ServiceViewBeacon } from "./ServiceViewBeacon";
import { BUTTON_FOCUS, DIVIDER, HEADING, SECTION } from "./profileStyles";

/** El botón de WhatsApp pasa por la API, que cuenta el clic y redirige a la conversación. */
const whatsappHref = (branchId: number, serviceId?: number) =>
  `/api/v1/r/wa/${branchId}${serviceId ? `?service=${serviceId}` : ""}`;

const directionsHref = (branch: ProfileBranch) =>
  `https://www.google.com/maps/dir/?api=1&destination=${branch.lat},${branch.lng}`;

/** La ficha pública del local, del lado del servidor. Los números salen de la API, nunca el WhatsApp. */
export function ProfilePage({ profile, today }: { profile: Profile; today: number }) {
  // Todavía no hay "sede principal" (SDD §13): la ficha usa la más antigua.
  const main = profile.branches[0];

  // Solo hay pestaña para los apartados que se dibujan.
  const tabs: ProfileTab[] = [
    ...(profile.description ? [{ id: "section-about", label: "Info" }] : []),
    ...(profile.benefits.length > 0 ? [{ id: "section-benefits", label: "Beneficios" }] : []),
    ...(profile.images.length > 0 ? [{ id: "section-photos", label: "Fotos" }] : []),
    ...(profile.services.length > 0 ? [{ id: "section-services", label: "Servicios" }] : []),
    { id: "section-hours", label: "Horarios" },
  ];

  return (
    <div className="min-h-dvh bg-movo-tabs pb-28 font-sans text-white">
      <header className="flex items-center justify-between px-4 py-3.5">
        <Link
          href="/"
          aria-label="Volver al mapa"
          className={cn(
            "grid size-[2.625rem] place-items-center rounded-[0.875rem] bg-movo-page",
            BUTTON_FOCUS,
          )}
        >
          <ArrowLeftIcon />
        </Link>
        <Link href="/" aria-label="MOVO, ir al inicio" className={cn("rounded", BUTTON_FOCUS)}>
          <Logo className="h-7" />
        </Link>
        <span className="size-[2.625rem]" aria-hidden="true" />
      </header>

      <main>
        <div className="flex items-center gap-4 px-5 pt-1.5">
          <CoverTile
            src={profile.images[0] ?? null}
            categorySlug={profile.category.slug}
            className="size-[4.75rem] shrink-0 rounded-[1.375rem] bg-movo-card"
            iconClassName="size-9"
          />
          <div className="min-w-0">
            <h1 className="font-display text-[1.75rem] leading-[1.05] font-extrabold tracking-tight text-balance">
              {profile.name}
            </h1>
            <p className="mt-1.5 flex flex-wrap items-center gap-x-2 text-[0.8125rem] text-movo-muted">
              {profile.category.name}
              {profile.verified ? (
                <span className="inline-flex items-center gap-1 font-bold text-movo-accent">
                  <VerifiedIcon className="size-3.5" />
                  Verificado
                </span>
              ) : null}
            </p>
          </div>
        </div>

        <Stats branch={main} today={today} slug={profile.slug} benefits={profile.benefits.length} />

        <div className="grid grid-cols-2 gap-2.5 px-5 pt-[1.125rem]">
          <a
            href={whatsappHref(main.branch_id)}
            target="_blank"
            rel="noopener noreferrer"
            aria-label="Reservar por WhatsApp"
            className={cn(
              "flex h-[3.25rem] items-center justify-center gap-2 rounded-[0.875rem] bg-movo-accent font-bold text-movo-ink",
              BUTTON_FOCUS,
            )}
          >
            <Icon name="chat" className="size-[1.125rem]" />
            WhatsApp
          </a>
          <a
            href={directionsHref(main)}
            target="_blank"
            rel="noopener noreferrer"
            className={cn(
              "flex h-[3.25rem] items-center justify-center gap-2 rounded-[0.875rem] font-bold ring-[1.5px] ring-movo-muted/35",
              BUTTON_FOCUS,
            )}
          >
            <Icon name="route" className="size-[1.125rem]" />
            Llegar
          </a>
        </div>

        {main.external_booking_url ? (
          <a
            href={main.external_booking_url}
            target="_blank"
            rel="noopener noreferrer"
            className={cn(
              "mx-5 mt-2.5 flex h-12 items-center justify-center gap-2 rounded-[0.875rem] bg-movo-page text-sm font-semibold",
              BUTTON_FOCUS,
            )}
          >
            <Icon name="calendar" className="size-5" />
            Reservar cancha
          </a>
        ) : null}

        <ProfileTabs tabs={tabs} />

        {profile.description ? (
          <section id="section-about" aria-labelledby="about" className={SECTION}>
            <h2 id="about" className={HEADING}>
              Sobre el lugar
            </h2>
            <p className="mt-3 text-[1.0625rem] leading-relaxed whitespace-pre-line">
              {profile.description}
            </p>
          </section>
        ) : null}

        <ProfileBenefits businessName={profile.name} benefits={profile.benefits} />

        <PhotoSection images={profile.images} name={profile.name} />

        {profile.services.length > 0 ? (
          <section id="section-services" aria-labelledby="services" className={SECTION}>
            <h2 id="services" className={HEADING}>
              Servicios · {profile.services.length}
            </h2>
            <ul className="mt-1">
              {profile.services.map((service) => (
                <li
                  key={service.id}
                  className={cn(
                    "relative flex items-center justify-between gap-3 border-b py-4",
                    DIVIDER,
                  )}
                >
                  <ServiceViewBeacon
                    branchId={service.branch_id ?? main.branch_id}
                    serviceId={service.id}
                  />
                  <div className="min-w-0">
                    <h3 className="font-display text-base font-bold">{service.name}</h3>
                    {service.description ? (
                      <p className="mt-0.5 text-[0.8125rem] text-movo-muted">
                        {service.description}
                      </p>
                    ) : null}
                    <p className="mt-0.5 text-[0.8125rem] text-movo-muted">
                      {[
                        service.price ? formatPrice(service.price) : "",
                        describeSchedule(service.schedule),
                      ]
                        .filter(Boolean)
                        .join(" · ")}
                    </p>
                  </div>
                  <a
                    href={whatsappHref(service.branch_id ?? main.branch_id, service.id)}
                    target="_blank"
                    rel="noopener noreferrer"
                    aria-label={`Consultar por ${service.name} por WhatsApp`}
                    className={cn(
                      "inline-flex min-h-11 shrink-0 items-center rounded-[0.625rem] bg-movo-page px-3.5 text-[0.8125rem] font-bold",
                      BUTTON_FOCUS,
                    )}
                  >
                    Consultar
                  </a>
                </li>
              ))}
            </ul>
          </section>
        ) : null}

        <section id="section-hours" aria-labelledby="hours" className={SECTION}>
          <h2 id="hours" className={HEADING}>
            Horarios
          </h2>
          <div className="mt-1 space-y-4">
            {profile.branches.map((branch) => (
              <HoursCard
                key={branch.branch_id}
                branch={branch}
                title={profile.branches.length > 1 ? branch.name : null}
              />
            ))}
          </div>
        </section>

        <section aria-labelledby="branches" className={SECTION}>
          <h2 id="branches" className={HEADING}>
            {profile.branches.length > 1 ? "Sedes" : "Dónde estamos"}
          </h2>
          <ul className="mt-3.5 space-y-3">
            {profile.branches.map((branch) => (
              <li key={branch.branch_id} className="flex items-center justify-between gap-3">
                <div className="min-w-0 text-[0.9375rem]">
                  <p>
                    {branch.address}, {branch.city}
                  </p>
                  <p className="mt-0.5 text-[0.8125rem] text-movo-muted">
                    {branch.name}
                    {branch.reference ? ` · ${branch.reference}` : ""}
                  </p>
                </div>
                <a
                  href={directionsHref(branch)}
                  target="_blank"
                  rel="noopener noreferrer"
                  aria-label={`Cómo llegar a ${branch.name}`}
                  className={cn(
                    "inline-flex min-h-11 w-26 shrink-0 items-center justify-center gap-1.5 rounded-[0.875rem] text-sm font-bold ring-[1.5px] ring-movo-muted/35",
                    BUTTON_FOCUS,
                  )}
                >
                  <Icon name="route" className="size-4" />
                  Llegar
                </a>
              </li>
            ))}
          </ul>
        </section>
      </main>

      <div className="fixed inset-x-0 bottom-0 z-30 border-t border-movo-muted/15 bg-movo-field px-4 pt-3 pb-5">
        <div className="mx-auto max-w-md">
          <a
            href={whatsappHref(main.branch_id)}
            target="_blank"
            rel="noopener noreferrer"
            className={cn(
              "flex h-[3.125rem] items-center justify-center gap-2 rounded-[0.875rem] bg-movo-accent font-bold text-movo-ink",
              BUTTON_FOCUS,
            )}
          >
            <Icon name="chat" className="size-5" />
            Reservar por WhatsApp
          </a>
        </div>
      </div>

      <ProfileViewBeacon slug={profile.slug} />
    </div>
  );
}

/** Tres datos de un vistazo: calificación, si está abierto ahora y cuántos beneficios hay. */
function Stats({
  branch,
  today,
  slug,
  benefits,
}: {
  branch: ProfileBranch;
  today: number;
  slug: string;
  benefits: number;
}) {
  const open = branch.open_now.open;
  // "Cerrado · abre a las 09:00" → "Cerrado" arriba y el resto debajo.
  const [state, ...rest] = describeOpenNow(branch.open_now, today).split(" · ");
  const cell = cn("min-w-0 py-3.5 text-xs text-movo-muted");
  return (
    <div
      className={cn(
        "mx-5 mt-[1.375rem] grid divide-x border-y [&>*:not(:first-child)]:pl-4",
        benefits > 0 ? "grid-cols-3" : "grid-cols-2",
        DIVIDER,
        "divide-movo-muted/15",
      )}
    >
      <div className={cell}>
        <StarRating
          average={branch.rating_avg}
          count={branch.rating_count}
          onDark
          className="mb-0.5 text-lg font-extrabold text-white"
        />
        <Link
          href={`/businesses/${slug}/reviews`}
          className={cn("block truncate rounded underline", BUTTON_FOCUS)}
        >
          Ver reseñas
        </Link>
      </div>
      <div className={cell}>
        <b className={cn("mb-0.5 block text-lg", open ? "text-movo-ok" : "text-movo-error")}>
          {state}
        </b>
        {rest.join(" · ") || "\u00a0"}
      </div>
      {benefits > 0 ? (
        <div className={cell}>
          <b className="mb-0.5 block text-lg text-white">{benefits}</b>
          {benefits === 1 ? "beneficio" : "beneficios"}
        </div>
      ) : null}
    </div>
  );
}

function HoursCard({ branch, title }: { branch: ProfileBranch; title: string | null }) {
  const groups = groupHours(branch.hours);
  const hasHours = branch.hours.length > 0;

  return (
    <div>
      {title ? <h3 className="mt-3 font-display text-base font-bold">{title}</h3> : null}
      {hasHours ? (
        <dl>
          {groups.map((group) => (
            <div
              key={group.label}
              className={cn("flex justify-between gap-4 border-b py-3 text-sm", DIVIDER)}
            >
              <dt className="font-semibold">{group.label}</dt>
              <dd className="text-right text-movo-muted">
                {group.ranges.length === 0 ? (
                  <span className="font-semibold text-movo-error">Cerrado</span>
                ) : (
                  group.ranges.map((range) => (
                    <span key={range} className="block">
                      {range}
                    </span>
                  ))
                )}
              </dd>
            </div>
          ))}
        </dl>
      ) : (
        <p className="py-3 text-sm text-movo-muted">El local todavía no cargó sus horarios.</p>
      )}
    </div>
  );
}
