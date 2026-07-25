export const siteURL =
  process.env.NEXT_PUBLIC_SITE_URL ?? "https://collectatcg.com.br";

export function absoluteURL(path: string): string {
  return new URL(path, siteURL).toString();
}
