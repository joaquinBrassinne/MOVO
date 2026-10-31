import { expect, test, type Page } from "@playwright/test";

// Home y Mapa contra una API simulada: así la prueba no depende de los locales que haya
// en la base (la búsqueda real la cubren los tests del backend).
test.use({
  geolocation: { latitude: -32.1738, longitude: -64.1146 },
  permissions: ["geolocation"],
});

const CATEGORIES = [
  { id: 1, slug: "gimnasios", name: "Gimnasios" },
  { id: 2, slug: "yoga", name: "Yoga" },
  { id: 8, slug: "trekking-outdoor", name: "Trekking y outdoor" },
];

const GYM = {
  branch_id: 10,
  business_name: "Alma Haus Training Club",
  business_slug: "alma-haus",
  branch_name: "Centro",
  address: "Av. San Martín 450",
  city: "Río Tercero",
  reference: "",
  lat: -32.1698,
  lng: -64.1126,
  distance_m: 482,
  rating_avg: 0,
  rating_count: 0,
  verified: true,
  featured: false,
  category_slug: "gimnasios",
  category_name: "Gimnasios",
};

const TREKKING = {
  ...GYM,
  branch_id: 11,
  business_name: "Sierras Trekking",
  business_slug: "sierras-trekking",
  distance_m: 1420,
  category_slug: "trekking-outdoor",
  category_name: "Trekking y outdoor",
};

async function mockApi(page: Page) {
  const searches: URL[] = [];
  await page.route("**/api/v1/categories", (route) =>
    route.fulfill({ json: { categories: CATEGORIES } }),
  );
  await page.route("**/api/v1/home/featured*", (route) => route.fulfill({ json: { results: [] } }));
  await page.route("**/api/v1/search*", (route) => {
    const url = new URL(route.request().url());
    searches.push(url);
    const results =
      url.searchParams.get("category") === "trekking-outdoor" ? [TREKKING] : [GYM, TREKKING];
    return route.fulfill({ json: { results, next_offset: null } });
  });
  return searches;
}

test.describe("Search views", () => {
  test(
    "Home lists nearby businesses and filters by the trekking category",
    { tag: ["@critical", "@e2e", "@search", "@SEARCH-E2E-001"] },
    async ({ page }) => {
      const searches = await mockApi(page);
      await page.goto("/");

      const nearby = page.getByRole("region", { name: "Cerca tuyo" });
      await expect(nearby.getByRole("link", { name: /Alma Haus Training Club/ })).toBeVisible();
      await expect(nearby.getByText("Gimnasios · a 500 m")).toBeVisible();
      // Sin locales Pro todavía: la sección de destacados no se muestra.
      await expect(page.getByRole("heading", { name: "Destacados cerca tuyo" })).toHaveCount(0);

      await page.getByRole("button", { name: "Trekking y outdoor" }).click();
      await expect(nearby.getByRole("link", { name: /Sierras Trekking/ })).toBeVisible();
      await expect(nearby.getByRole("link", { name: /Alma Haus/ })).toHaveCount(0);
      expect(searches.at(-1)?.searchParams.get("category")).toBe("trekking-outdoor");
    },
  );

  test(
    "Map shows a pin per business and the list in the bottom sheet",
    { tag: ["@high", "@e2e", "@search", "@SEARCH-E2E-002"] },
    async ({ page }) => {
      await mockApi(page);
      await page.goto("/map");

      await expect(page.locator(".leaflet-marker-icon")).toHaveCount(2);
      await expect(page.getByText("2 disponibles")).toBeVisible();
      await expect(page.getByRole("button", { name: "Trekking y outdoor" })).toBeVisible();

      await page.getByTitle("Sierras Trekking").click({ force: true });
      await expect(page.getByRole("button", { name: "Bajar la lista" })).toBeVisible();
      await expect(page.getByRole("link", { name: /Ver local/ }).first()).toHaveAttribute(
        "href",
        /\/businesses\//,
      );
    },
  );
});
