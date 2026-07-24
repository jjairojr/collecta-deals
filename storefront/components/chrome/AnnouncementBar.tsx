import Container from "@/components/ui/Container";

// Blue marquee strip. Pixel copy is accent-free (Press Start 2P has no accents).
const NOTES = [
  "FRETE GRATIS ACIMA DE R$ 250",
  "ENVIO EM 24H",
  "+18.400 CARTAS NO ESTOQUE",
];

export default function AnnouncementBar() {
  return (
    <div className="overflow-hidden border-b-4 border-outline bg-royal py-2 text-brand-soft">
      <Container className="flex items-center gap-5">
        <span className="flex items-center gap-x-5 whitespace-nowrap font-pixel text-[9px] leading-none">
          <span className="flex items-center gap-2">
            <span
              className="inline-block h-2 w-2 rounded-full bg-brand-soft animate-blink"
              aria-hidden
            />
            {NOTES[0]}
          </span>
          <span className="hidden sm:inline">{NOTES[1]}</span>
          <span className="hidden md:inline">{NOTES[2]}</span>
        </span>
        <span className="ml-auto hidden whitespace-nowrap font-pixel text-[9px] leading-none text-white sm:block">
          P1 · CONVIDADO
        </span>
      </Container>
    </div>
  );
}
