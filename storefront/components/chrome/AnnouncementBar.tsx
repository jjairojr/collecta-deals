import Container from "@/components/ui/Container";
import { pixelText } from "@/lib/format";
import { freeShippingMinLabel } from "@/lib/free-shipping";

const NOTES = [
  "A LOJA DE CARTAS MAIS RÁPIDA DO BRASIL",
  "CARTA ORIGINAL E CONFERIDA",
  "ENVIO EM 24H",
  // "+18.400 CARTAS NO ESTOQUE",
];

function NoteRow({ notes, hidden }: { notes: string[]; hidden?: boolean }) {
  return (
    <span
      aria-hidden={hidden}
      className="flex shrink-0 items-center gap-5 pr-5 font-pixel text-[9px] leading-none"
    >
      {notes.map((n) => (
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

// `extra` are the phrases from the dashboard's "Vitrine" screen that are live
// today; they lead the bar. The free shipping line only shows when the backend
// reports the rule (freeShippingMinBRL), so the bar never promises what the
// checkout does not give.
export default function AnnouncementBar({
  extra = [],
  freeShippingMinBRL = 0,
}: {
  extra?: string[];
  freeShippingMinBRL?: number;
}) {
  const notes = [
    ...extra,
    ...(freeShippingMinBRL > 0 ? [`FRETE GRATIS NO PAC ACIMA DE ${freeShippingMinLabel(freeShippingMinBRL)}`] : []),
    ...NOTES,
  ];
  return (
    <div className="overflow-hidden border-b-4 border-outline bg-royal py-2 text-brand-soft">
      <div className="flex w-max animate-marquee sm:hidden">
        <NoteRow notes={notes} />
        <NoteRow notes={notes} hidden />
      </div>
      <Container className="hidden items-center gap-5 sm:flex">
        <span className="flex items-center gap-x-5 whitespace-nowrap font-pixel text-[9px] leading-none">
          <span className="flex items-center gap-2">
            <span
              className="inline-block h-2 w-2 rounded-full bg-brand-soft animate-blink"
              aria-hidden
            />
            {pixelText(notes[0])}
          </span>
          <span>{pixelText(notes[1])}</span>
          <span className="hidden md:inline">{pixelText(notes[2])}</span>
        </span>
        <span className="ml-auto hidden whitespace-nowrap font-pixel text-[9px] leading-none text-white sm:block">
          P1 · CONVIDADO
        </span>
      </Container>
    </div>
  );
}
