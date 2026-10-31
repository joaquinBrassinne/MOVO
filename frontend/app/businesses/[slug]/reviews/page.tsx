import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";

import { Logo } from "@/components/Logo";
import { BranchReviewsView } from "@/components/reviews/list/BranchReviewsView";
import { ArrowLeftIcon } from "@/components/socios/icons";
import { cn } from "@/lib/cn";
import { getProfile } from "@/lib/server/profile";

export async function generateMetadata({
  params,
}: PageProps<"/businesses/[slug]/reviews">): Promise<Metadata> {
  const { slug } = await params;
  const cached = await getProfile(slug);
  if (!cached) return { title: "Local no encontrado — MOVO", robots: { index: false } };
  return {
    title: `Reseñas de ${cached.profile.name} | MOVO`,
    alternates: { canonical: `/businesses/${cached.profile.slug}/reviews` },
  };
}

/** The reviews of one branch: `?branch=<branch_id>`, or the oldest branch when missing or unknown. */
export default async function ReviewsPage({
  params,
  searchParams,
}: PageProps<"/businesses/[slug]/reviews">) {
  const [{ slug }, { branch: branchParam }] = await Promise.all([params, searchParams]);
  const cached = await getProfile(slug);
  if (!cached) notFound();
  const { profile } = cached;

  const requested = Number(Array.isArray(branchParam) ? branchParam[0] : branchParam);
  const branch = profile.branches.find((b) => b.branch_id === requested) ?? profile.branches[0];
  const several = profile.branches.length > 1;

  return (
    <div className="min-h-dvh bg-movo-cream font-sans text-movo-ink">
      <header className="bg-movo-page text-white">
        <div className="mx-auto flex max-w-2xl items-center justify-between gap-3 px-4 py-4">
          <Link
            href={`/businesses/${profile.slug}`}
            aria-label={`Volver a ${profile.name}`}
            className="grid size-11 shrink-0 place-items-center rounded-full bg-white/10 outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
          >
            <ArrowLeftIcon />
          </Link>
          <Link href="/" aria-label="MOVO, ir al inicio" className="outline-none">
            <Logo className="h-8" />
          </Link>
          <span className="size-11 shrink-0" aria-hidden="true" />
        </div>
      </header>

      <main className="mx-auto max-w-2xl px-4 pt-6 pb-16">
        <h1 className="font-display text-3xl leading-tight font-extrabold">Reseñas</h1>
        <p className="mt-1 text-movo-slate">
          {profile.name}
          {several ? ` · ${branch.name}` : ""}
        </p>

        {several ? (
          <nav aria-label="Sedes" className="mt-4 flex flex-wrap gap-2">
            {profile.branches.map((b) => (
              <Link
                key={b.branch_id}
                href={`/businesses/${profile.slug}/reviews?branch=${b.branch_id}`}
                aria-current={b.branch_id === branch.branch_id ? "page" : undefined}
                className={cn(
                  "inline-flex h-11 items-center rounded-full px-4 text-sm font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink",
                  b.branch_id === branch.branch_id
                    ? "bg-movo-page text-white"
                    : "bg-white text-movo-ink shadow-sm",
                )}
              >
                {b.name}
              </Link>
            ))}
          </nav>
        ) : null}

        <div className="mt-5">
          {/* Keyed by branch so the chip, the cursor and the list restart on a branch change. */}
          <BranchReviewsView key={branch.branch_id} branchId={branch.branch_id} />
        </div>
      </main>
    </div>
  );
}
