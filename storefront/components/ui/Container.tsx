// Page gutter + max width from the handoff: 1360px, 28px side padding. `narrow`
// switches to the cart screen's 1100px.
export default function Container({
  children,
  narrow,
  className = "",
}: {
  children: React.ReactNode;
  narrow?: boolean;
  className?: string;
}) {
  const max = narrow ? "max-w-[1100px]" : "max-w-[1360px]";
  return (
    <div className={`mx-auto w-full ${max} px-5 sm:px-7 ${className}`}>
      {children}
    </div>
  );
}
