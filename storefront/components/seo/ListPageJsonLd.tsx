import JsonLd from "@/components/seo/JsonLd";
import { SINGLES_PAGE_SIZE } from "@/lib/paging";
import { breadcrumbJsonLd, itemListJsonLd } from "@/lib/seo";

// Breadcrumb + ItemList for one page of a card list, in the order page 1 already
// uses (catalog order), with positions continuing across pages.
export default function ListPageJsonLd({
  name,
  crumbs,
  slugs,
  page = 1,
}: {
  name: string;
  crumbs: { name: string; path?: string }[];
  slugs: string[];
  page?: number;
}) {
  const start = (page - 1) * SINGLES_PAGE_SIZE;
  const paths = slugs.slice(start, start + SINGLES_PAGE_SIZE).map((s) => `/carta/${s}`);
  const title = page > 1 ? `${name} — página ${page}` : name;
  return (
    <>
      <JsonLd data={breadcrumbJsonLd(crumbs)} />
      {paths.length > 0 && <JsonLd data={itemListJsonLd(title, paths, start + 1)} />}
    </>
  );
}
