import type { NextConfig } from "next";

import { securityHeaders } from "./lib/securityHeaders.ts";

const nextConfig: NextConfig = {
  /* config options here */
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
