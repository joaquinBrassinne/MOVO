import { CategoryIcon } from "@/components/explorar/icons";

interface Props {
  images: string[];
  name: string;
  categorySlug: string;
}

/** Fotos del local, una al lado de otra con desplazamiento que encaja. Sin fotos, el ícono de la categoría. */
export function Gallery({ images, name, categorySlug }: Props) {
  if (images.length === 0) {
    return (
      <div className="grid h-72 place-items-center bg-gradient-to-br from-movo-card to-movo-page text-movo-accent">
        <CategoryIcon slug={categorySlug} className="size-20" />
      </div>
    );
  }

  return (
    <ul
      aria-label={`Fotos de ${name}`}
      className="no-scrollbar flex h-72 snap-x snap-mandatory overflow-x-auto bg-movo-card"
    >
      {images.map((src, index) => (
        <li key={src} className="relative h-full w-full shrink-0 snap-center">
          {/* Las URLs son prefirmadas y de otro origen: next/image no las optimiza. */}
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={src}
            alt={`${name}, foto ${index + 1} de ${images.length}`}
            loading={index === 0 ? "eager" : "lazy"}
            className="size-full object-cover"
          />
        </li>
      ))}
    </ul>
  );
}
