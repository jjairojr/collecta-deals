import { ImageResponse } from "next/og";
import { readFile } from "node:fs/promises";
import { join } from "node:path";

export const size = { width: 1200, height: 630 };
export const contentType = "image/png";
export const alt = "Collecta — Loja de cartas TCG";

const ink = "#0b0b0c";
const brand = "#f6559b";
const brandSoft = "#fdc4e5";
const royal = "#1355b3";
const royalLight = "#2f74d8";

export default async function OpenGraphImage() {
  const [baloo, pixel, dmSans, dmSansBold, mascot] = await Promise.all([
    readFile(join(process.cwd(), "app", "og", "baloo2-extrabold.ttf")),
    readFile(join(process.cwd(), "app", "og", "press-start-2p.ttf")),
    readFile(join(process.cwd(), "app", "og", "dm-sans-medium.ttf")),
    readFile(join(process.cwd(), "app", "og", "dm-sans-bold.ttf")),
    readFile(join(process.cwd(), "app", "og", "mascot.png")),
  ]);
  const mascotSrc = `data:image/png;base64,${mascot.toString("base64")}`;
  return new ImageResponse(
    (
      <div
        style={{
          width: "100%",
          height: "100%",
          display: "flex",
          flexDirection: "column",
          backgroundColor: royal,
          fontFamily: "DM Sans",
        }}
      >
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            backgroundColor: ink,
            color: "#ffffff",
            fontFamily: "Press Start 2P",
            fontSize: 13,
            letterSpacing: 1,
            padding: "18px 0",
            width: "100%",
          }}
        >
          A LOJA DE CARTAS MAIS RAPIDA DO BRASIL · ENVIO EM 24H
        </div>
        <div style={{ position: "relative", display: "flex", flex: 1 }}>
          {Array.from({ length: 12 }, (_, i) => (
            <div
              key={i}
              style={{
                position: "absolute",
                left: 96 + i * 96,
                top: 0,
                width: 2,
                height: "100%",
                backgroundColor: "rgba(255,255,255,.08)",
              }}
            />
          ))}
          <div
            style={{
              position: "absolute",
              inset: 0,
              backgroundImage:
                "radial-gradient(circle at 78% 45%, rgba(246,85,155,.6) 0%, rgba(246,85,155,0) 60%)",
            }}
          />
          <div
            style={{
              display: "flex",
              flexDirection: "column",
              justifyContent: "center",
              paddingLeft: 72,
              width: 690,
            }}
          >
            <div style={{ display: "flex" }}>
              <div
                style={{
                  backgroundColor: brandSoft,
                  color: ink,
                  fontFamily: "Press Start 2P",
                  fontSize: 15,
                  padding: "14px 18px",
                  border: `4px solid ${ink}`,
                  borderRadius: 8,
                  boxShadow: `4px 4px 0 ${ink}`,
                }}
              >
                PLAYER 1 · READY
              </div>
            </div>
            <div
              style={{
                position: "relative",
                display: "flex",
                marginTop: 18,
                height: 138,
              }}
            >
              {[
                { x: 9, y: 9, color: ink },
                { x: 5, y: 5, color: brand },
                { x: 0, y: 0, color: "#ffffff" },
              ].map((layer) => (
                <div
                  key={layer.color}
                  style={{
                    position: "absolute",
                    left: layer.x,
                    top: layer.y,
                    fontFamily: "Baloo 2",
                    fontSize: 128,
                    fontWeight: 800,
                    color: layer.color,
                    lineHeight: 1,
                  }}
                >
                  COLLECTA
                </div>
              ))}
            </div>
            <div
              style={{
                display: "flex",
                flexDirection: "column",
                marginTop: 14,
                fontSize: 27,
                lineHeight: 1.5,
                color: brandSoft,
              }}
            >
              <div style={{ display: "flex" }}>
                <span>{"Singles avulsas e produto selado de\u00A0"}</span>
                <span style={{ color: "#ffffff", fontWeight: 700 }}>
                  Pokémon
                </span>
                <span>,</span>
              </div>
              <div style={{ display: "flex" }}>
                <span style={{ color: "#ffffff", fontWeight: 700 }}>
                  One Piece
                </span>
                <span>{"\u00A0e\u00A0"}</span>
                <span style={{ color: "#ffffff", fontWeight: 700 }}>
                  Riftbound
                </span>
                <span>{". Conferido carta por carta."}</span>
              </div>
            </div>
            <div style={{ display: "flex", marginTop: 32 }}>
              <div
                style={{
                  backgroundColor: brand,
                  color: "#ffffff",
                  fontFamily: "Press Start 2P",
                  fontSize: 14,
                  padding: "17px 24px",
                  border: `4px solid ${ink}`,
                  borderRadius: 10,
                  boxShadow: `6px 6px 0 ${ink}`,
                }}
              >
                EXPLORAR SINGLES
              </div>
              <div
                style={{
                  marginLeft: 22,
                  backgroundColor: brandSoft,
                  color: ink,
                  fontFamily: "Press Start 2P",
                  fontSize: 14,
                  padding: "17px 24px",
                  border: `4px solid ${ink}`,
                  borderRadius: 10,
                  boxShadow: `6px 6px 0 ${ink}`,
                }}
              >
                VER SELADOS
              </div>
            </div>
          </div>
          <div
            style={{
              position: "relative",
              display: "flex",
              width: 386,
              height: 386,
              alignSelf: "center",
              marginLeft: "auto",
              marginRight: 84,
            }}
          >
            <div
              style={{
                position: "absolute",
                left: 26,
                top: 26,
                width: 360,
                height: 360,
                borderRadius: 999,
                backgroundColor: brand,
              }}
            />
            <div
              style={{
                display: "flex",
                position: "absolute",
                left: 0,
                top: 0,
                width: 360,
                height: 360,
                borderRadius: 999,
                border: `6px solid ${ink}`,
                backgroundColor: royalLight,
                overflow: "hidden",
              }}
            >
              <img
                src={mascotSrc}
                width={348}
                height={362}
                style={{ width: "100%", height: "100%", objectFit: "cover" }}
              />
            </div>
            <div
              style={{
                position: "absolute",
                bottom: 0,
                right: -6,
                transform: "rotate(-4deg)",
                backgroundColor: "#ffffff",
                color: ink,
                fontFamily: "Press Start 2P",
                fontSize: 13,
                padding: "12px 14px",
                border: `4px solid ${ink}`,
                borderRadius: 8,
                boxShadow: `4px 4px 0 ${ink}`,
              }}
            >
              PEGA ESSA!
            </div>
          </div>
        </div>
      </div>
    ),
    {
      ...size,
      fonts: [
        { name: "Baloo 2", data: baloo, weight: 800, style: "normal" },
        { name: "Press Start 2P", data: pixel, weight: 400, style: "normal" },
        { name: "DM Sans", data: dmSans, weight: 500, style: "normal" },
        { name: "DM Sans", data: dmSansBold, weight: 700, style: "normal" },
      ],
    },
  );
}
