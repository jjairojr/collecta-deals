import AuthShell from "@/components/account/AuthShell";
import LoginForm from "@/components/account/LoginForm";
import { safeNext } from "@/components/account/safeNext";

export const metadata = {
  title: "Entrar",
  robots: { index: false },
};

export default async function EntrarPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const sp = await searchParams;
  return (
    <AuthShell eyebrow="PLAYER 1 · LOGIN" title="Entrar na sua conta">
      <LoginForm next={safeNext(sp.next)} />
    </AuthShell>
  );
}
