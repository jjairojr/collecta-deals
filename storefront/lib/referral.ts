import { brl } from "@/lib/format";

export const REFERRAL_ACTIVATION_CARDS = 10;
export const REFERRAL_REWARD_CENTS = 1000;

export function buildReferralMessage(code: string, link: string): string {
  return [
    "Tô usando o Collecta pra organizar minha coleção de cartas (Pokémon, One Piece, Riftbound, Lorcana).",
    `Cria sua conta com meu código ${code}, cadastra ${REFERRAL_ACTIVATION_CARDS} cartas e ganha ${brl(REFERRAL_REWARD_CENTS)} na loja Collecta.`,
    "",
    link,
  ].join("\n");
}

export function referralWhatsAppURL(code: string, link: string): string {
  return `https://wa.me/?text=${encodeURIComponent(buildReferralMessage(code, link))}`;
}
