import { useEffect, useState } from "react";
import { ImageOff } from "lucide-react";
import { cardImageURL, tcgProductImageURL } from "../api";

export default function CardArt({
  set,
  number,
  name,
  productID,
  imageURL,
  className = "",
}: {
  set: string;
  number: string;
  name?: string;
  productID?: number;
  imageURL?: string;
  className?: string;
}) {
  const sources: string[] = [];
  if (imageURL) {
    sources.push(imageURL);
  }
  if (productID) {
    sources.push(tcgProductImageURL(productID));
  }
  if (set) {
    sources.push(cardImageURL(set, number));
  }
  const [sourceIdx, setSourceIdx] = useState(0);
  // Reset to the first source whenever the inputs change (e.g. the seller edits
  // the image URL live), so a prior onError fallthrough doesn't stick.
  useEffect(() => {
    setSourceIdx(0);
  }, [imageURL, productID, set, number]);
  if (sourceIdx >= sources.length) {
    return (
      <div
        className={`relative flex flex-col items-center justify-center overflow-hidden bg-surface text-center ${className}`}
      >
        <div aria-hidden className="art-stripes-45 pointer-events-none absolute inset-0 opacity-70" />
        <ImageOff aria-hidden className="relative h-auto w-1/3 max-w-[34px] text-brand-soft/60" />
        <span className="relative mt-1 max-w-full truncate px-1 font-mono text-[10px] leading-tight text-slate-400">
          {number}
        </span>
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
