import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  /* config options here */
  experimental: {
    // ppr: "incremental",
  },
  images: {
    remotePatterns: [
      {
        protocol: "https",
        hostname: "img10.360buyimg.com",
        pathname: "/**",
      },
    ],
  },
};

export default nextConfig;
