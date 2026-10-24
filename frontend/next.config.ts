import type { NextConfig } from "next";

const API_URL = process.env.API_URL ?? "http://localhost:8080";

const nextConfig: NextConfig = {
  /* config options here */
  cacheComponents: true,
  partialPrefetching: true,
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
  // El navegador llama a la API Go por el mismo origen, con el access token en la
  // cabecera Authorization. La lista es cerrada a propósito: /api/v1/auth/* NO se
  // expone, porque ahí viaja el refresh token y solo debe verlo app/api/session
  // (cookie httpOnly). Todo lo que no esté acá devuelve 404.
  async rewrites() {
    return [
      { source: "/api/v1/me/:path*", destination: `${API_URL}/api/v1/me/:path*` },
      { source: "/api/v1/admin/:path*", destination: `${API_URL}/api/v1/admin/:path*` },
      { source: "/api/v1/plans", destination: `${API_URL}/api/v1/plans` },
      { source: "/api/v1/categories", destination: `${API_URL}/api/v1/categories` },
    ];
  },
};

export default nextConfig;
