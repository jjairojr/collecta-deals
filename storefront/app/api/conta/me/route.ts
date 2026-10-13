import { publicUser, NO_STORE } from "@/lib/account-api";
import { sessionUser } from "@/lib/session";

export async function GET(): Promise<Response> {
  const session = await sessionUser();
  return Response.json({ user: session ? publicUser(session.user) : null }, { headers: NO_STORE });
}
