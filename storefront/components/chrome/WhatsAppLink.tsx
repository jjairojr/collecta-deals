"use client";

import { trackWhatsAppContact } from "@/lib/analytics";

export default function WhatsAppLink({
  href,
  origin,
  className,
  children,
}: {
  href: string;
  origin: string;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      onClick={() => trackWhatsAppContact(origin)}
      className={className}
    >
      {children}
    </a>
  );
}
