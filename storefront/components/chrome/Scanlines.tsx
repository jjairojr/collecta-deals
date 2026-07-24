// Full-screen CRT scanline overlay (pointer-events none, multiply blend).
// Purely decorative, so hidden from assistive tech.
export default function Scanlines() {
  return <div className="scanlines" aria-hidden />;
}
