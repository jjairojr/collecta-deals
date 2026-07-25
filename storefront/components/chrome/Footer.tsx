import Link from "next/link";
import Container from "@/components/ui/Container";

interface FooterLink {
  label: string;
  href: string;
  external?: boolean;
}

export default function Footer() {
  const whatsapp = process.env.NEXT_PUBLIC_WHATSAPP;
  const whatsappHref = whatsapp ? `https://wa.me/${whatsapp}` : "https://wa.me/";

  const columns: { heading: string; links: FooterLink[] }[] = [
    {
      heading: "COMPRAR",
      links: [
        { label: "Singles", href: "/singles" },
        { label: "Produto selado", href: "/selado" },
        { label: "Pokémon", href: "/singles?jogo=pokemon" },
        { label: "One Piece", href: "/singles?jogo=onepiece" },
      ],
    },
    {
      heading: "AJUDA",
      links: [
        { label: "Como comprar", href: "/" },
        { label: "Envio e prazos", href: "/" },
        { label: "Condições das cartas", href: "/" },
        { label: "Fale no WhatsApp", href: whatsappHref, external: true },
      ],
    },
    {
      heading: "COLLECTA",
      links: [
        { label: "Sobre nós", href: "/" },
        { label: "Vender com a gente", href: whatsappHref, external: true },
        { label: "Instagram", href: "/" },
        { label: "Termos", href: "/" },
      ],
    },
  ];

  return (
    <footer className="mt-16 border-t-[6px] border-outline bg-brand">
      <Container className="py-12">
        <div className="grid gap-10 md:grid-cols-[1.2fr_.6fr_.6fr_.6fr]">
          <div>
            <span className="font-display logo-shadow text-4xl leading-none text-white">
              COLLECTA
            </span>
            <p className="mt-4 max-w-xs text-sm leading-relaxed text-white">
              Loja de cartas do Brasil. Singles e selados de Pokémon, One Piece
              e Riftbound — conferidos carta por carta.
            </p>
          </div>
          {columns.map((col) => (
            <div key={col.heading}>
              <h3 className="font-pixel text-[9px] text-outline">{col.heading}</h3>
              <ul className="mt-3 space-y-1">
                {col.links.map((l) => (
                  <li key={l.label}>
                    {l.external ? (
                      <a
                        href={l.href}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-block py-1.5 text-sm text-white underline-offset-4 hover:underline"
                      >
                        {l.label}
                      </a>
                    ) : (
                      <Link
                        href={l.href}
                        className="inline-block py-1.5 text-sm text-white underline-offset-4 hover:underline"
                      >
                        {l.label}
                      </Link>
                    )}
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      </Container>
      <div className="border-t-[3px] border-outline py-4 text-center font-pixel text-[8px] text-outline">
        © 2026 COLLECTA · GAME OVER? NUNCA. CONTINUE (9)
      </div>
    </footer>
  );
}
