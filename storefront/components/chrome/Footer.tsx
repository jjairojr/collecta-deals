import Link from "next/link";
import Container from "@/components/ui/Container";
import WhatsAppLink from "@/components/chrome/WhatsAppLink";

export default function Footer() {
  const whatsapp = process.env.NEXT_PUBLIC_WHATSAPP;
  const whatsappHref = whatsapp ? `https://wa.me/${whatsapp}` : "https://wa.me/";

  return (
    <footer className="mt-16 border-t-[6px] border-outline bg-brand">
      <Container className="flex flex-col items-start gap-6 py-10 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <span className="font-display logo-shadow text-3xl leading-none text-white">
            COLLECTA
          </span>
          <p className="mt-3 max-w-xs text-sm leading-relaxed text-white">
            Loja de cartas do Brasil. Singles e selados — conferidos carta por
            carta.
          </p>
        </div>
        <nav className="flex flex-wrap items-center gap-x-7 gap-y-1">
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
            href="/politica-de-devolucao"
            className="inline-block py-1.5 text-sm font-bold text-white underline-offset-4 hover:underline"
          >
            Devoluções
          </Link>
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
