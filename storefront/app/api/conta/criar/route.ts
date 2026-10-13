import { errorResponse, publicUser, readJSON, str, NO_STORE } from "@/lib/account-api";
import { register } from "@/lib/collecta";
import { setSessionCookie } from "@/lib/session";

export async function POST(req: Request): Promise<Response> {
  const body = await readJSON(req);
  if (!body) {
    return Response.json({ error: "dados inválidos" }, { status: 400 });
  }
  const email = str(body, "email").toLowerCase();
  const password = typeof body.password === "string" ? body.password : "";
  const displayName = str(body, "displayName").slice(0, 40);
  const referralCode = str(body, "referralCode").toUpperCase();
  if (!email || password.length < 8) {
    return Response.json({ error: "Informe e-mail e uma senha com 8+ caracteres" }, { status: 400 });
  }
  try {
    const { token, user } = await register({
      email,
      password,
      displayName: displayName || undefined,
      referralCode: referralCode || undefined,
    });
    await setSessionCookie(token);
    return Response.json({ user: publicUser(user) }, { headers: NO_STORE });
  } catch (err) {
    return errorResponse(err);
  }
}
