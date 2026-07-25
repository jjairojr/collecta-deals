/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  images: {
    remotePatterns: [
      { protocol: "https", hostname: "product-images.tcgplayer.com" },
    ],
  },
};

export default nextConfig;
