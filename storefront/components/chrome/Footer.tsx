import Link from "next/link";
import Container from "@/components/ui/Container";
import TrackedLink from "@/components/chrome/TrackedLink";
import WhatsAppLink from "@/components/chrome/WhatsAppLink";
import { STORE, STORE_ADDRESS_LINE, STORE_MAPS_URL } from "@/lib/store";

export default function Footer() {
  const whatsapp = process.env.NEXT_PUBLIC_WHATSAPP;
  const whatsappHref = whatsapp ? `https://wa.me/${whatsapp}` : "https://wa.me/";

  return (
    <footer className="mt-16 border-t-[6px] border-outline bg-brand">
      <Container className="flex flex-col items-start gap-6 py-10 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <span className="font-display logo-shadow text-3xl leading-none text-white">
            COLLECTA
          </span>
          <p className="mt-3 max-w-xs text-sm leading-relaxed text-white">
            Loja de cartas TCG em Goiânia. Singles e selados conferidos carta por
            carta, envio para todo o Brasil.
          </p>
          <address className="mt-3 max-w-xs text-sm not-italic leading-relaxed text-white">
            <TrackedLink
              href={STORE_MAPS_URL}
              track={{ kind: "directions", origin: "footer" }}
              className="underline-offset-4 hover:underline"
            >
              {STORE_ADDRESS_LINE}
            </TrackedLink>
            <br />
            Todos os dias, {STORE.opens} às {STORE.closes} · {STORE.phone}
          </address>
        </div>
        <nav className="flex max-w-md flex-wrap items-center gap-x-7 gap-y-1">
          <Link
            href="/loja"
            className="inline-block py-1.5 text-sm font-bold text-white underline-offset-4 hover:underline"
          >
            Loja física
          </Link>
          <Link
            href="/singles"
            className="inline-block py-1.5 text-sm font-bold text-white underline-offset-4 hover:underline"
          >
            Singles
          </Link>
          <Link
            href="/selado"
            className="inline-block py-1.5 text-sm font-bold text-white underline-offset-4 hover:underline"
          >
            Produto selado
          </Link>
          <Link
            href="/acessorios"
            className="inline-block py-1.5 text-sm font-bold text-white underline-offset-4 hover:underline"
          >
            Acessórios
          </Link>
          <Link
            href="/politica-de-devolucao"
            className="inline-block py-1.5 text-sm font-bold text-white underline-offset-4 hover:underline"
          >
            Devoluções
          </Link>
          <TrackedLink
            href={STORE.instagram}
            track={{ kind: "social", network: "instagram", origin: "footer" }}
            className="inline-block py-1.5 text-sm font-bold text-white underline-offset-4 hover:underline"
          >
            Instagram {STORE.instagramHandle}
          </TrackedLink>
          <WhatsAppLink
            href={whatsappHref}
            origin="footer"
            className="inline-block py-1.5 text-sm font-bold text-white underline-offset-4 hover:underline"
          >
            Fale no WhatsApp
          </WhatsAppLink>
        </nav>
      </Container>
      <div className="border-t-[3px] border-outline py-4 text-center font-pixel text-[8px] text-outline">
        © 2026 COLLECTA · GAME OVER? NUNCA. CONTINUE (9)
      </div>
    </footer>
  );
}
