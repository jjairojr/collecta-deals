export function safeNext(raw: string | string[] | undefined, fallback = "/conta"): string {
  const v = typeof raw === "string" ? raw : "";
  return v.startsWith("/") && !v.startsWith("//") ? v : fallback;
}
