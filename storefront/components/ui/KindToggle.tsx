import Link from "next/link";

// Server-safe Singles/Selados switch for the sealed pages, mirroring the TIPO
// chips inside SinglesBrowse: the active kind is a static chip, the other a link.
const chip = "rounded-[8px] border-[3px] border-outline px-3 py-2 text-xs font-bold transition-colors";

export default function KindToggle({ singlesHref }: { singlesHref: string }) {
  return (
    <div className="mb-6 flex flex-wrap gap-2">
      <Link
        href={singlesHref}
        className={`${chip} bg-outline text-[#c9c9d1] hover:text-white`}
      >
        Singles
      </Link>
      <span aria-current="true" className={`${chip} bg-brand text-white`}>
        Selados
      </span>
    </div>
  );
}
