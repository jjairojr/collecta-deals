import { redirect } from "next/navigation";
import AccountView from "@/components/account/AccountView";
import { referralMe } from "@/lib/collecta";
import { sessionUser } from "@/lib/session";

export const metadata = {
  title: "Minha conta",
  robots: { index: false },
};

export default async function ContaPage() {
  const session = await sessionUser();
  if (!session) {
    redirect("/conta/entrar?next=/conta");
  }
  const referral = await referralMe(session.token).catch(() => null);
  return <AccountView user={session.user} referral={referral} />;
}
