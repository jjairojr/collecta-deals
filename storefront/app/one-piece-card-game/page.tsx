import { notFound } from "next/navigation";
import type { Metadata } from "next";
import GameHubPage, { hubMetadata } from "@/components/hub/GameHubPage";
import { hubFor } from "@/lib/game-hubs";

export const revalidate = 300;

const hub = hubFor("onepiece");

export async function generateMetadata(): Promise<Metadata> {
  return hub ? hubMetadata(hub) : {};
}

export default function OnePieceCardGamePage() {
  if (!hub) {
    notFound();
  }
  return <GameHubPage hub={hub} />;
}
