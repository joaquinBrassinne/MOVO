// Contrato de GET /search y GET /home/featured de la API Go (tarjeta liviana, sin datos internos).

export interface SearchResult {
  branch_id: number;
  business_name: string;
  business_slug: string;
  branch_name: string;
  address: string;
  city: string;
  reference: string;
  lat: number;
  lng: number;
  distance_m: number;
  rating_avg: number;
  rating_count: number;
  verified: boolean;
  featured: boolean;
  category_slug: string;
  category_name: string;
  /** La etiqueta del beneficio más alto que ofrece el local ("-15%", "-$2.000", "Beneficio"); `null` si no tiene. */
  benefit_label: string | null;
}

export interface SearchPage {
  results: SearchResult[];
  next_offset: number | null;
}

export interface FeaturedList {
  results: SearchResult[];
}

export interface Point {
  lat: number;
  lng: number;
}
