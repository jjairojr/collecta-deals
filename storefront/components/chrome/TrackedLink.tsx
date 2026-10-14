"use client";

import { trackGetDirections, trackShareReferral, trackSocialClick } from "@/lib/analytics";

type Track =
  | { kind: "directions"; origin: string }
  | { kind: "social"; network: string; origin: string }
  | { kind: "share_referral" };

export default function TrackedLink({
  href,
  track,
  className,
  children,
}: {
  href: string;
  track: Track;
  className?: string;
  children: React.ReactNode;
}) {
  const onClick = () => {
    if (track.kind === "directions") {
      trackGetDirections(track.origin);
    } else if (track.kind === "social") {
      trackSocialClick(track.network, track.origin);
    } else {
      trackShareReferral("whatsapp");
    }
  };
  return (
    <a href={href} target="_blank" rel="noopener noreferrer" onClick={onClick} className={className}>
      {children}
    </a>
  );
}
