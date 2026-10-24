export default function DocSection({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section className="sticker rounded-[14px] bg-surface p-6 sm:p-7 [--sh:6px]">
      <h2 className="font-display text-2xl font-bold leading-tight text-white">
        {title}
      </h2>
      <div className="mt-3 space-y-3 text-sm leading-relaxed text-[#c9c9d1]">
        {children}
      </div>
    </section>
  );
}
