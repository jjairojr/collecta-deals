import type { Metadata, Viewport } from "next";
import { Baloo_2, DM_Sans, Press_Start_2P } from "next/font/google";
import "./globals.css";
import { CartProvider } from "@/lib/cart";
import { loadCatalog } from "@/lib/catalog";
import AnnouncementBar from "@/components/chrome/AnnouncementBar";
import Header from "@/components/chrome/Header";
import TabBar from "@/components/chrome/TabBar";
import Footer from "@/components/chrome/Footer";
import Scanlines from "@/components/chrome/Scanlines";

const baloo = Baloo_2({
  subsets: ["latin"],
  weight: ["600", "700", "800"],
  variable: "--font-baloo",
});
const dmSans = DM_Sans({
  subsets: ["latin"],
  weight: ["400", "500", "700"],
  variable: "--font-dm-sans",
});
const pressStart = Press_Start_2P({
  subsets: ["latin"],
  weight: "400",
  variable: "--font-press-start",
});

const siteURL = process.env.NEXT_PUBLIC_SITE_URL ?? "https://collecta.vercel.app";

export const metadata: Metadata = {
  metadataBase: new URL(siteURL),
  title: "Collecta — Marketplace de cartas",
  description:
    "Singles avulsas e produto selado de Pokémon, One Piece e Riftbound. Conferido carta por carta, enviado do Brasil.",
  openGraph: {
    title: "Collecta — Marketplace de cartas",
    description:
      "Singles e selados de Pokémon, One Piece e Riftbound. Monte seu pedido e finalize no WhatsApp.",
    type: "website",
    locale: "pt_BR",
    siteName: "Collecta",
  },
  twitter: {
    card: "summary_large_image",
    title: "Collecta — Marketplace de cartas",
    description: "Singles e selados. Peça pelo WhatsApp.",
  },
};

export const viewport: Viewport = {
  themeColor: "#141416",
};

export default async function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const { navGames } = await loadCatalog();
  const games = navGames.map((g) => ({ id: g.id, name: g.name }));
  return (
    <html
      lang="pt-BR"
      className={`${baloo.variable} ${dmSans.variable} ${pressStart.variable}`}
    >
      <body>
        <CartProvider>
          <Scanlines />
          <AnnouncementBar />
          <Header />
          <TabBar games={games} />
          <main className="min-h-[60vh]">{children}</main>
          <Footer />
        </CartProvider>
      </body>
    </html>
  );
}
