import { useState } from "react";
import { cardImageURL, tcgProductImageURL } from "../api";

export default function CardArt({
  set,
  number,
  name,
  productID,
  className = "",
}: {
  set: string;
  number: string;
  name?: string;
  productID?: number;
  className?: string;
}) {
  const sources: string[] = [];
  if (productID) {
    sources.push(tcgProductImageURL(productID));
  }
  if (set) {
    sources.push(cardImageURL(set, number));
  }
  const [sourceIdx, setSourceIdx] = useState(0);
  if (sourceIdx >= sources.length) {
    return (
      <div
        className={`flex items-center justify-center bg-slate-800 text-center ${className}`}
      >
        <span className="px-2 font-mono text-[11px] text-slate-500">{number}</span>
      </div>
    );
  }
  return (
    <img
      src={sources[sourceIdx]}
      alt={name ?? number}
      loading="lazy"
      onError={() => setSourceIdx((i) => i + 1)}
      className={`bg-slate-800 object-cover ${className}`}
    />
  );
}
