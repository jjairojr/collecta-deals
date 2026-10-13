import AuthShell from "@/components/account/AuthShell";
import RegisterForm from "@/components/account/RegisterForm";
import { safeNext } from "@/components/account/safeNext";

export const metadata = {
  title: "Criar conta",
  robots: { index: false },
};

export default async function CriarPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const sp = await searchParams;
  const ref = typeof sp.ref === "string" ? sp.ref.trim().toUpperCase().slice(0, 6) : "";
  return (
    <AuthShell eyebrow="NEW PLAYER" title="Criar conta Collecta">
      <RegisterForm next={safeNext(sp.next)} initialCode={ref} />
    </AuthShell>
  );
}
