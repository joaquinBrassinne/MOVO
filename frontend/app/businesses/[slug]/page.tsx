import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { ProfilePage } from "@/components/locales/ProfilePage";
import { jsonLd } from "@/lib/jsonld";
import { getProfile } from "@/lib/server/profile";

export async function generateMetadata({
  params,
}: PageProps<"/businesses/[slug]">): Promise<Metadata> {
  const { slug } = await params;
  const cached = await getProfile(slug);
  if (!cached) return { title: "Local no encontrado — MOVO", robots: { index: false } };
  const { profile } = cached;

  const place = profile.branches[0];
  const description =
    profile.description.trim() ||
    `${profile.category.name} en ${place.city}. Mirá los servicios, los horarios y reservá por WhatsApp.`;
  return {
    title: `${profile.name} — ${profile.category.name} en ${place.city} | MOVO`,
    description,
    alternates: { canonical: `/businesses/${profile.slug}` },
    openGraph: {
      type: "website",
      title: `${profile.name} — MOVO`,
      description,
      locale: "es_AR",
    },
  };
}

export default async function LocalPage({ params }: PageProps<"/businesses/[slug]">) {
  const { slug } = await params;
  const cached = await getProfile(slug);
  if (!cached) notFound();
  const { profile, today } = cached;

  const place = profile.branches[0];
  // Datos estructurados para buscadores: nombre, dirección y ubicación. Sin teléfono: el
  // contacto va por el botón de WhatsApp, que cuenta el clic.
  const structured = {
    "@context": "https://schema.org",
    "@type": "LocalBusiness",
    name: profile.name,
    description: profile.description || undefined,
    address: {
      "@type": "PostalAddress",
      streetAddress: place.address,
      addressLocality: place.city,
      addressCountry: "AR",
    },
    geo: { "@type": "GeoCoordinates", latitude: place.lat, longitude: place.lng },
  };

  return (
    <>
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: jsonLd(structured) }} />
      <ProfilePage profile={profile} today={today} />
    </>
  );
}
