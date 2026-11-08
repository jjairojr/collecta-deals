const EXTERNAL_PHOTO_HOSTS = ["repositorio.sbrauble.com", "http2.mlstatic.com"];

function backendHost(): string | undefined {
  const base = process.env.STOREFRONT_API;
  if (!base) {
    return undefined;
  }
  try {
    return new URL(base).host;
  } catch {
    return undefined;
  }
}

const ITEM_PHOTO_PATH = "/api/item-photos/";

export function isProxiablePhoto(url: string): boolean {
  let parsed: URL;
  try {
    parsed = new URL(url);
  } catch {
    return false;
  }
  return (parsed.protocol === "https:" || parsed.protocol === "http:") && EXTERNAL_PHOTO_HOSTS.includes(parsed.host);
}

function itemPhotoID(url: string): string | undefined {
  try {
    const parsed = new URL(url);
    if (parsed.host !== backendHost() || !parsed.pathname.startsWith(ITEM_PHOTO_PATH)) {
      return undefined;
    }
    const id = parsed.pathname.slice(ITEM_PHOTO_PATH.length);
    return isItemPhotoID(id) ? id : undefined;
  } catch {
    return undefined;
  }
}

export function isItemPhotoID(id: string): boolean {
  return /^[\w-]+\.(jpe?g|png|webp)$/i.test(id);
}

export function itemPhotoSource(id: string): string | undefined {
  const base = process.env.STOREFRONT_API;
  return base && isItemPhotoID(id) ? `${base}${ITEM_PHOTO_PATH}${id}` : undefined;
}

export function ownPhotoURL(url: string | undefined): string | undefined {
  if (!url) {
    return url;
  }
  const item = itemPhotoID(url);
  if (item) {
    return `/foto?item=${encodeURIComponent(item)}`;
  }
  return isProxiablePhoto(url) ? `/foto?u=${encodeURIComponent(url)}` : url;
}
