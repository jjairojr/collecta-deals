import { ImageResponse } from "next/og";

export const size = { width: 1200, height: 630 };
export const contentType = "image/png";
export const alt = "Collecta — Loja de cartas TCG";

export default function OpenGraphImage() {
  return new ImageResponse(
    (
      <div
        style={{
          width: "100%",
          height: "100%",
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          justifyContent: "center",
          backgroundColor: "#F6559B",
        }}
      >
        <div
          style={{
            fontSize: 148,
            fontWeight: 800,
            color: "#ffffff",
            letterSpacing: -6,
            textShadow: "8px 8px 0 #141416",
          }}
        >
          COLLECTA
        </div>
        <div style={{ fontSize: 42, marginTop: 24, color: "#FDC4E5" }}>
          Cartas Pokémon · One Piece · Riftbound
        </div>
        <div
          style={{
            fontSize: 30,
            marginTop: 28,
            color: "#141416",
            backgroundColor: "#FDC4E5",
            padding: "12px 28px",
            borderRadius: 14,
          }}
        >
          Singles e selados · pedido pelo WhatsApp
        </div>
      </div>
    ),
    size,
  );
}
