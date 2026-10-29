import Link from "next/link";

import { Icon, VerifiedIcon } from "@/components/explorar/icons";
import { Logo } from "@/components/Logo";
import { StarRating } from "@/components/reviews/StarRating";
import { ArrowLeftIcon } from "@/components/socios/icons";
import { formatPrice } from "@/lib/business/price";
import { describeSchedule } from "@/lib/business/schedule";
import {
  describeOpenNow,
  groupHours,
  type Profile,
  type ProfileBranch,
} from "@/lib/search/profile";

import { Gallery } from "./Gallery";
import { ProfileBenefits } from "./ProfileBenefits";
import { ProfileViewBeacon } from "./ProfileViewBeacon";

/** El botón de WhatsApp pasa por la API, que cuenta el clic y redirige a la conversación. */
const whatsappHref = (branchId: number, serviceId?: number) =>
  `/api/v1/r/wa/${branchId}${serviceId ? `?service=${serviceId}` : ""}`;

const directionsHref = (branch: ProfileBranch) =>
  `https://www.google.com/maps/dir/?api=1&destination=${branch.lat},${branch.lng}`;

const BUTTON_FOCUS = "outline-none focus-visible:ring-2 focus-visible:ring-movo-ink";

/** La ficha pública del local, del lado del servidor. Los números salen de la API, nunca el WhatsApp. */
export function ProfilePage({ profile, today }: { profile: Profile; today: number }) {
  // Todavía no hay "sede principal" (SDD §13): la ficha usa la más antigua.
  const main = profile.branches[0];

  return (
    <div className="min-h-dvh bg-movo-cream pb-32 font-sans text-movo-ink">
      <div className="relative">
        <Gallery images={profile.images} name={profile.name} categorySlug={profile.category.slug} />
        <header className="absolute inset-x-0 top-0 flex items-center justify-between bg-gradient-to-b from-movo-page/80 to-transparent px-4 pt-4 pb-8">
          <Link
            href="/mapa"
            aria-label="Volver al mapa"
            className="grid size-11 place-items-center rounded-full bg-movo-cream/90 text-movo-ink outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
          >
            <ArrowLeftIcon />
          </Link>
          <Link href="/" aria-label="MOVO, ir al inicio" className="outline-none">
            <Logo className="h-8" />
          </Link>
          <span className="size-11" aria-hidden="true" />
        </header>
      </div>

      <main className="relative -mt-6 rounded-t-[2rem] bg-movo-cream px-4 pt-6">
        <p className="inline-flex rounded-full bg-white px-3 py-1.5 text-sm font-semibold shadow-sm">
          {profile.category.name}
        </p>

        <h1 className="mt-3 flex items-center gap-2 font-display text-3xl leading-tight font-extrabold">
          {profile.name}
          {profile.verified ? (
            <span title="Local verificado por MOVO" className="shrink-0 text-movo-gold">
              <VerifiedIcon className="size-7" />
              <span className="sr-only">Local verificado por MOVO</span>
            </span>
          ) : null}
        </h1>

        <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1">
          <StarRating
            average={main.rating_avg}
            count={main.rating_count}
            showCount
            starClassName="size-5"
          />
          <Link
            href={`/locales/${profile.slug}/reviews`}
            className="text-sm font-semibold underline outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
          >
            Ver todas las reseñas
          </Link>
        </div>

        <OpenStatus branch={main} today={today} />

        <div className="mt-5 flex gap-3">
          <a
            href={whatsappHref(main.branch_id)}
            target="_blank"
            rel="noopener noreferrer"
            className={`flex h-14 flex-1 items-center justify-center gap-2 rounded-full bg-movo-accent font-display text-base font-bold ${BUTTON_FOCUS}`}
          >
            <Icon name="chat" className="size-5" />
            Reservar por WhatsApp
          </a>
          <a
            href={directionsHref(main)}
            target="_blank"
            rel="noopener noreferrer"
            className={`flex h-14 items-center justify-center gap-2 rounded-full bg-white px-6 font-display text-base font-bold shadow-sm ${BUTTON_FOCUS}`}
          >
            <Icon name="route" className="size-5" />
            Llegar
          </a>
        </div>

        {main.external_booking_url ? (
          <a
            href={main.external_booking_url}
            target="_blank"
            rel="noopener noreferrer"
            className={`mt-3 flex h-12 items-center justify-center gap-2 rounded-full bg-movo-sand text-sm font-semibold ${BUTTON_FOCUS}`}
          >
            <Icon name="calendar" className="size-5" />
            Reservar cancha
          </a>
        ) : null}

        {profile.description ? (
          <section aria-labelledby="about" className="mt-8">
            <h2 id="about" className="font-display text-xl font-bold">
              Sobre el lugar
            </h2>
            <p className="mt-3 whitespace-pre-line text-movo-slate">{profile.description}</p>
          </section>
        ) : null}

        <ProfileBenefits businessName={profile.name} benefits={profile.benefits} />

        {profile.services.length > 0 ? (
          <section aria-labelledby="services" className="mt-8">
            <div className="flex items-baseline justify-between">
              <h2 id="services" className="font-display text-xl font-bold">
                Servicios
              </h2>
              <span className="text-sm font-semibold text-movo-slate">
                {profile.services.length}{" "}
                {profile.services.length === 1 ? "disponible" : "disponibles"}
              </span>
            </div>
            <ul className="mt-3 space-y-3">
              {profile.services.map((service) => (
                <li
                  key={service.id}
                  className="flex items-center justify-between gap-3 rounded-3xl bg-white p-4 shadow-sm"
                >
                  <div className="min-w-0">
                    <h3 className="font-display text-lg font-bold">{service.name}</h3>
                    {service.description ? (
                      <p className="mt-0.5 text-sm text-movo-slate">{service.description}</p>
                    ) : null}
                    <p className="mt-0.5 text-sm text-movo-slate">
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
                    className={`shrink-0 rounded-full bg-movo-sand px-4 py-2 text-sm font-bold ${BUTTON_FOCUS}`}
                  >
                    Consultar
                  </a>
                </li>
              ))}
            </ul>
          </section>
        ) : null}

        <section aria-labelledby="hours" className="mt-8">
          <h2 id="hours" className="font-display text-xl font-bold">
            Horarios
          </h2>
          <div className="mt-3 space-y-3">
            {profile.branches.map((branch) => (
              <HoursCard
                key={branch.branch_id}
                branch={branch}
                title={profile.branches.length > 1 ? branch.name : null}
              />
            ))}
          </div>
        </section>

        <section aria-labelledby="branches" className="mt-8">
          <h2 id="branches" className="font-display text-xl font-bold">
            {profile.branches.length > 1 ? "Sedes" : "Dónde estamos"}
          </h2>
          <ul className="mt-3 space-y-3">
            {profile.branches.map((branch) => (
              <li
                key={branch.branch_id}
                className="flex items-center justify-between gap-3 rounded-3xl bg-white p-4 shadow-sm"
              >
                <div className="min-w-0">
                  <h3 className="font-display text-lg font-bold">{branch.name}</h3>
                  <p className="text-sm text-movo-slate">
                    {branch.address}, {branch.city}
                  </p>
                  {branch.reference ? (
                    <p className="text-sm text-movo-slate">{branch.reference}</p>
                  ) : null}
                </div>
                <a
                  href={directionsHref(branch)}
                  target="_blank"
                  rel="noopener noreferrer"
                  aria-label={`Cómo llegar a ${branch.name}`}
                  className={`inline-flex shrink-0 items-center gap-1.5 rounded-full bg-movo-sand px-4 py-2 text-sm font-bold ${BUTTON_FOCUS}`}
                >
                  <Icon name="route" className="size-4" />
                  Llegar
                </a>
              </li>
            ))}
          </ul>
        </section>
      </main>

      <div className="fixed inset-x-0 bottom-0 z-10 border-t border-movo-sand bg-white px-4 py-3">
        <div className="mx-auto flex max-w-md items-center justify-between gap-3">
          <p className="min-w-0 leading-tight">
            <span className="block text-xs text-movo-slate">Atención directa</span>
            <span className="block truncate font-display text-lg font-extrabold">
              {profile.name}
            </span>
          </p>
          <a
            href={whatsappHref(main.branch_id)}
            target="_blank"
            rel="noopener noreferrer"
            className={`inline-flex h-14 shrink-0 items-center gap-2 rounded-full bg-movo-accent px-6 font-display font-bold ${BUTTON_FOCUS}`}
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

function OpenStatus({ branch, today }: { branch: ProfileBranch; today: number }) {
  const open = branch.open_now.open;
  return (
    <p className="mt-2 flex items-center gap-2 text-sm text-movo-slate">
      <span
        aria-hidden="true"
        className={`size-2.5 shrink-0 rounded-full ${open ? "bg-movo-ok" : "bg-movo-slate/50"}`}
      />
      <span className={open ? "font-semibold text-movo-ok" : undefined}>
        {describeOpenNow(branch.open_now, today)}
      </span>
    </p>
  );
}

function HoursCard({ branch, title }: { branch: ProfileBranch; title: string | null }) {
  const groups = groupHours(branch.hours);
  const hasHours = branch.hours.length > 0;

  return (
    <div className="rounded-3xl bg-white p-4 shadow-sm">
      {title ? <h3 className="mb-2 font-display text-base font-bold">{title}</h3> : null}
      {hasHours ? (
        <dl className="divide-y divide-movo-sand">
          {groups.map((group) => (
            <div key={group.label} className="flex justify-between gap-4 py-2.5 text-sm">
              <dt className="font-semibold">{group.label}</dt>
              <dd className="text-right text-movo-slate">
                {group.ranges.length === 0 ? (
                  <span className="font-semibold text-movo-danger">Cerrado</span>
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
        <p className="text-sm text-movo-slate">El local todavía no cargó sus horarios.</p>
      )}
    </div>
  );
}
