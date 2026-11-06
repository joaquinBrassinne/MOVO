import type { NextConfig } from "next";

import { securityHeaders } from "./lib/securityHeaders.ts";

const nextConfig: NextConfig = {
  // Servidor mínimo para la imagen de producción (frontend/Dockerfile).
  output: "standalone",
  cacheComponents: true,
  partialPrefetching: true,
  headers: async () => securityHeaders(process.env.NODE_ENV === "production"),
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
};

export default nextConfig;
