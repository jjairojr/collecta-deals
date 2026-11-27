// Only the real product photos (uploaded to the backend, ~500 KB JPGs) go
// through Vercel's image optimizer, as WebP in three widths. Everything else
// keeps `unoptimized`, so usage stays small: on Hobby the plan includes 5K
// transformations a month and images fail past that; immutable photo URLs and a
// 31-day cache keep each photo to a handful of transformations.
const apiHost = (() => {
  try {
    return new URL(process.env.STOREFRONT_API || "https://collecta-deals-production.up.railway.app").hostname;
  } catch {
    return "collecta-deals-production.up.railway.app";
  }
})();

/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  images: {
    formats: ["image/webp"],
    minimumCacheTTL: 2678400,
    deviceSizes: [640, 828, 1080],
    imageSizes: [256, 384],
    remotePatterns: [{ protocol: "https", hostname: apiHost, pathname: "/api/item-photos/**" }],
  },
  async redirects() {
    return [{ source: "/jogo/:jogo", destination: "/singles/:jogo", permanent: true }];
  },
};

export default nextConfig;
