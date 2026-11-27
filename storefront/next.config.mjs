// Only the real product photos (uploaded to the backend, ~500 KB JPGs) go
// through Vercel's image optimizer, as WebP in five widths. Everything else
// keeps `unoptimized`, so usage stays small: on Hobby the plan includes 5K
// transformations a month and images fail past that; immutable photo URLs and a
// 31-day cache keep each photo to a handful of transformations.
// The backend writes photo URLs with its own public host; accept that one and,
// if different, the host the site reads the catalog from.
const photoHosts = (() => {
  const hosts = new Set(["collecta-deals-production.up.railway.app"]);
  try {
    if (process.env.STOREFRONT_API) {
      hosts.add(new URL(process.env.STOREFRONT_API).hostname);
    }
  } catch {
    // keep the default host
  }
  return [...hosts];
})();

/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  images: {
    formats: ["image/webp"],
    minimumCacheTTL: 2678400,
    deviceSizes: [640, 828, 1080],
    imageSizes: [256, 384],
    remotePatterns: photoHosts.map((hostname) => ({ protocol: "https", hostname, pathname: "/api/item-photos/**" })),
  },
  async redirects() {
    return [{ source: "/jogo/:jogo", destination: "/singles/:jogo", permanent: true }];
  },
};

export default nextConfig;
