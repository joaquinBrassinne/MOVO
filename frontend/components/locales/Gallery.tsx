import { CategoryIcon } from "@/components/explorar/icons";

interface Props {
  images: string[];
  name: string;
  categorySlug: string;
}

/**
 * Fotos del local, una al lado de otra con desplazamiento que encaja. Sin fotos, el ícono de la categoría.
 *
 * La foto se ve entera (`object-contain`): con `object-cover` una foto cuadrada o vertical, en una
 * pantalla ancha, se recortaba a una franja y se agrandaba hasta pixelarse. Los lados que sobran se
 * rellenan con la misma foto desenfocada, que además es decorativa y no se anuncia.
 */
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
        <li key={src} className="relative h-full w-full shrink-0 snap-center overflow-hidden">
          {/* Las URLs son prefirmadas y de otro origen: next/image no las optimiza. */}
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={src}
            alt=""
            aria-hidden="true"
            loading={index === 0 ? "eager" : "lazy"}
            className="absolute inset-0 size-full scale-125 object-cover opacity-70 blur-2xl"
          />
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={src}
            alt={`${name}, foto ${index + 1} de ${images.length}`}
            loading={index === 0 ? "eager" : "lazy"}
            className="relative size-full object-contain"
          />
        </li>
      ))}
    </ul>
  );
}
