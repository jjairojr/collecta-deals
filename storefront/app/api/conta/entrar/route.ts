import { errorResponse, publicUser, readJSON, str, NO_STORE } from "@/lib/account-api";
import { login } from "@/lib/collecta";
import { setSessionCookie } from "@/lib/session";

export async function POST(req: Request): Promise<Response> {
  const body = await readJSON(req);
  if (!body) {
    return Response.json({ error: "dados inválidos" }, { status: 400 });
  }
  const email = str(body, "email").toLowerCase();
  const password = typeof body.password === "string" ? body.password : "";
  if (!email || !password) {
    return Response.json({ error: "Informe e-mail e senha" }, { status: 400 });
  }
  try {
    const { token, user } = await login(email, password);
    if (user.isGuest) {
      return Response.json({ error: "Conta convidada não entra pelo site" }, { status: 403 });
    }
    await setSessionCookie(token);
    return Response.json({ user: publicUser(user) }, { headers: NO_STORE });
  } catch (err) {
    return errorResponse(err);
  }
}
