export const SINGLES_PAGE_SIZE = 24;

export function pageCount(total: number): number {
  return Math.max(1, Math.ceil(total / SINGLES_PAGE_SIZE));
}
