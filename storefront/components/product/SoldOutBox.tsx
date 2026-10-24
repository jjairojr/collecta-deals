"use client";

import { trackWhatsAppContact } from "@/lib/analytics";
import { brl } from "@/lib/format";
import { absoluteURL } from "@/lib/site";
import { whatsappURL } from "@/lib/whatsapp";

export default function SoldOutBox({
  name,
  price,
  path,
}: {
  name: string;
  price: number;
  path: string;
}) {
  const href = whatsappURL(
    `Olá! Quero ser avisado quando ${name} voltar ao estoque: ${absoluteURL(path)}`,
  );
  return (
    <div
      className="sticker mt-6 rounded-[14px] bg-surface p-6"
      style={{ ["--sh" as string]: "6px" }}
    >
      <span className="bg-outline px-2.5 py-1.5 font-pixel text-[9px] text-brand-soft">
        ESGOTADO
      </span>
      <div className="mt-4 flex flex-wrap items-baseline gap-3">
        <span className="font-pixel text-[18px] text-faint line-through sm:text-[24px]">
          {brl(price)}
        </span>
        <span className="text-[13px] text-muted">último preço na loja</span>
      </div>
      <p className="mt-3 text-[14px] leading-relaxed text-muted">
        Acabou o nosso estoque. Chama no WhatsApp que a gente avisa assim que chegar de novo, ou procura outra
        versão logo abaixo.
      </p>
      <a
        href={href}
        target="_blank"
        rel="noopener noreferrer"
        onClick={() => trackWhatsAppContact("avise_me")}
        className="arcade-press sticker mt-5 flex min-w-[220px] items-center justify-center rounded-[10px] bg-brand px-6 py-3.5 font-pixel text-[11px] text-white [--sh:6px]"
      >
        AVISE-ME NO WHATSAPP
      </a>
    </div>
  );
}
