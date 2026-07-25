import type { Metadata, Viewport } from "next";
import { Baloo_2, DM_Sans, Press_Start_2P } from "next/font/google";
import { GoogleAnalytics } from "@next/third-parties/google";
import "./globals.css";
import { CartProvider } from "@/lib/cart";
import { loadCatalog } from "@/lib/catalog";
import { organizationJsonLd, webSiteJsonLd } from "@/lib/seo";
import { siteURL } from "@/lib/site";
import AnnouncementBar from "@/components/chrome/AnnouncementBar";
import Header from "@/components/chrome/Header";
import TabBar from "@/components/chrome/TabBar";
import Footer from "@/components/chrome/Footer";
import Scanlines from "@/components/chrome/Scanlines";
import CartToast from "@/components/cart/CartToast";
import JsonLd from "@/components/seo/JsonLd";

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

export const metadata: Metadata = {
  metadataBase: new URL(siteURL),
  title: {
    default: "Collecta — Cartas Pokémon, One Piece e Riftbound · Singles e Selados",
    template: "%s — Collecta",
  },
  description:
    "Loja de cartas TCG do Brasil: singles avulsas e produto selado de Pokémon, One Piece e Riftbound. Cartas conferidas uma a uma, pedido pelo WhatsApp e envio para todo o Brasil.",
  applicationName: "Collecta",
  openGraph: {
    title: "Collecta — Loja de cartas TCG",
    description:
      "Singles e selados de Pokémon, One Piece e Riftbound. Monte seu pedido e finalize no WhatsApp.",
    type: "website",
    locale: "pt_BR",
    siteName: "Collecta",
    url: siteURL,
  },
  twitter: {
    card: "summary_large_image",
    title: "Collecta — Loja de cartas TCG",
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
        <JsonLd data={organizationJsonLd()} />
        <JsonLd data={webSiteJsonLd()} />
        <CartProvider>
          <Scanlines />
          <AnnouncementBar />
          <Header />
          <TabBar games={games} />
          <main className="min-h-[60vh]">{children}</main>
          <Footer />
          <CartToast />
        </CartProvider>
      </body>
      {process.env.NEXT_PUBLIC_GA_ID && (
        <GoogleAnalytics gaId={process.env.NEXT_PUBLIC_GA_ID} />
      )}
    </html>
  );
}
