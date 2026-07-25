import Container from "@/components/ui/Container";
import { pixelText } from "@/lib/format";

const NOTES = [
  // "FRETE GRATIS ACIMA DE R$ 250",
  "A LOJA DE CARTAS MAIS RÁPIDA DO BRASIL",
  "CARTA CERTIFICADA E CONFERIDA",
  "ENVIO EM 24H",
  // "+18.400 CARTAS NO ESTOQUE",
];

function NoteRow({ hidden }: { hidden?: boolean }) {
  return (
    <span
      aria-hidden={hidden}
      className="flex shrink-0 items-center gap-5 pr-5 font-pixel text-[9px] leading-none"
    >
      {NOTES.map((n) => (
        <span key={n} className="flex items-center gap-2 whitespace-nowrap">
          <span
            className="inline-block h-2 w-2 rounded-full bg-brand-soft animate-blink"
            aria-hidden
          />
          {pixelText(n)}
        </span>
      ))}
    </span>
  );
}

export default function AnnouncementBar() {
  return (
    <div className="overflow-hidden border-b-4 border-outline bg-royal py-2 text-brand-soft">
      <div className="flex w-max animate-marquee sm:hidden">
        <NoteRow />
        <NoteRow hidden />
      </div>
      <Container className="hidden items-center gap-5 sm:flex">
        <span className="flex items-center gap-x-5 whitespace-nowrap font-pixel text-[9px] leading-none">
          <span className="flex items-center gap-2">
            <span
              className="inline-block h-2 w-2 rounded-full bg-brand-soft animate-blink"
              aria-hidden
            />
            {pixelText(NOTES[0])}
          </span>
          <span>{pixelText(NOTES[1])}</span>
          <span className="hidden md:inline">{pixelText(NOTES[2])}</span>
        </span>
        <span className="ml-auto hidden whitespace-nowrap font-pixel text-[9px] leading-none text-white sm:block">
          P1 · CONVIDADO
        </span>
      </Container>
    </div>
  );
}
