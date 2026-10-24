/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  images: {
    unoptimized: true,
  },
  async redirects() {
    return [{ source: "/jogo/:jogo", destination: "/singles/:jogo", permanent: true }];
  },
};

export default nextConfig;
