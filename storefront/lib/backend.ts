import "server-only";

// Server-side bridge to the Go API's checkout endpoints. The shared token never
// reaches the browser: only these route handlers add it.
export async function checkoutFetch(
  path: string,
  init: RequestInit = {},
): Promise<Response> {
  const base = process.env.STOREFRONT_API;
  const token = process.env.STOREFRONT_TOKEN;
  if (!base || !token) {
    return Response.json({ error: "checkout desligado" }, { status: 503 });
  }
  const headers = new Headers(init.headers);
  headers.set("X-Storefront-Token", token);
  headers.set("Content-Type", "application/json");
  try {
    const res = await fetch(`${base}${path}`, { ...init, headers, cache: "no-store" });
    const body = await res.text();
    return new Response(body, {
      status: res.status,
      headers: {
        "Content-Type": res.headers.get("Content-Type") ?? "application/json",
        "Cache-Control": "no-store",
      },
    });
  } catch {
    return Response.json({ error: "backend indisponível" }, { status: 502 });
  }
}
