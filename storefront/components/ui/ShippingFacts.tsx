import Link from "next/link";
import { brl } from "@/lib/format";
import { freeShippingMinLabel } from "@/lib/free-shipping";
import { SHIPPING } from "@/lib/store";

// Frete e prazo em uma linha, perto do topo das páginas de entrada (/selado/<jogo>,
// /singles/<jogo>, /pokemon-tcg), que é onde chega quem vem de um buscador ou de uma
// IA. O PAC grátis só aparece quando o servidor informa o limite
// (freeShippingMinBRL > 0), o mesmo que o checkout aplica.
export default function ShippingFacts({ freeShippingMinBRL }: { freeShippingMinBRL: number }) {
  const handling = `${SHIPPING.handlingDays[1]} ${SHIPPING.handlingDays[1] === 1 ? "dia útil" : "dias úteis"}`;
  const facts = [
    `Frete pelo CEP, a partir de uns ${brl(SHIPPING.minBRL * 100)}`,
    ...(freeShippingMinBRL > 0 ? [`PAC grátis a partir de ${freeShippingMinLabel(freeShippingMinBRL)} em produtos`] : []),
    `Postagem em até ${handling}, entrega em ${SHIPPING.transitDays[0]} a ${SHIPPING.transitDays[1]} dias úteis`,
    "Retirada grátis em Goiânia",
  ];
  return (
    <p className="mt-4 max-w-3xl rounded-[10px] border-2 border-outline bg-surface px-4 py-3 text-[13px] leading-relaxed text-white">
      {facts.join(" · ")}.{" "}
      <Link href="/frete" className="text-brand-soft underline underline-offset-2">
        Frete e prazos
      </Link>
    </p>
  );
}
