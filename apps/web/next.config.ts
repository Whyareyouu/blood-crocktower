import type { NextConfig } from "next";
import { fileURLToPath } from "node:url";

const apiUrl = process.env.API_INTERNAL_URL || "http://127.0.0.1:8080";

const config: NextConfig = {
  output: "standalone",
  outputFileTracingRoot: fileURLToPath(new URL("../../", import.meta.url)),
  poweredByHeader: false,
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${apiUrl}/api/:path*` }];
  },
};

export default config;
