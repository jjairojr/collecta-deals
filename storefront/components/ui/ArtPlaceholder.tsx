"use client";

import { useEffect, useState } from "react";

type Hue = "brand" | "royal" | "soft";
type Angle = "90" | "45" | "135";

const bg: Record<Hue, string> = {
  brand: "bg-brand",
  royal: "bg-royal",
  soft: "bg-brand-soft",
};

const labelColor: Record<Hue, string> = {
  brand: "text-white/70",
  royal: "text-brand-soft/80",
  soft: "text-on-soft/70",
};

const stripes: Record<Angle, string> = {
  "90": "art-stripes-90",
  "45": "art-stripes-45",
  "135": "art-stripes-135",
};

// The striped brand-color stand-in for product photography. When `imageURL` is
// set (real photo), the stripes/label are replaced — and if that photo fails to
// load, the stripes come back instead of a broken-image icon. Aspect ratio is
// owned by the caller so the same block serves 5:7 cards and 4:3 boxes.
export default function ArtPlaceholder({
  hue = "royal",
  angle = "90",
  label,
  imageURL,
  alt = "",
  priority = false,
}: {
  hue?: Hue;
  angle?: Angle;
  label: string;
  imageURL?: string;
  alt?: string;
  priority?: boolean;
}) {
  const [failed, setFailed] = useState(false);
  useEffect(() => {
    setFailed(false);
  }, [imageURL]);

  if (imageURL && !failed) {
    return (
      // eslint-disable-next-line @next/next/no-img-element
      <img
        src={imageURL}
        alt={alt}
        loading={priority ? "eager" : "lazy"}
        onError={() => setFailed(true)}
        className="h-full w-full object-cover"
      />
    );
  }
  return (
    <div className={`relative h-full w-full ${bg[hue]}`}>
      <div className={`absolute inset-0 ${stripes[angle]}`} aria-hidden />
      <div className="absolute inset-0 grid place-items-center p-3">
        <span
          className={`whitespace-pre-line text-center font-pixel text-[8px] leading-relaxed ${labelColor[hue]}`}
        >
          {label}
        </span>
      </div>
    </div>
  );
}
