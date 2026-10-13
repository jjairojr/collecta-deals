import Container from "@/components/ui/Container";

export default function AuthShell({
  eyebrow,
  title,
  children,
}: {
  eyebrow: string;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <Container narrow className="py-12">
      <div className="sticker sticker-5 mx-auto max-w-md rounded-[18px] bg-surface p-6 sm:p-8 [--sh:8px]">
        <div className="font-pixel text-[9px] text-brand">{eyebrow}</div>
        <h1 className="mt-3 font-display text-3xl font-bold leading-tight text-white sm:text-[40px]">
          {title}
        </h1>
        <div className="mt-5">{children}</div>
      </div>
    </Container>
  );
}
