(async (URL_PRODUTO, QTD) => {
  const api = "https://cartinha.com.br/api/v2/front";
  const headers = { "app-token": "wapstore", Accept: "application/json", "Content-Type": "application/json" };
  const call = (path, method = "GET", body) =>
    fetch(api + path, { method, headers, credentials: "include", body: body ? JSON.stringify(body) : undefined })
      .then((r) => r.json());

  const path = new URL(URL_PRODUTO).pathname;
  const { conteudo: p } = await call(`/url/product/detail?url=${encodeURIComponent(path)}`);
  if (!p?.id) return console.error("❌ produto não encontrado");
  console.log(`${p.nome}\nstatus: ${p.status} | estoque: ${p.estoque} | R$ ${p.precos.por}`);
  if (p.status !== "disponivel" || p.estoque < 1) return console.error("❌ indisponível");

  const qtd = Math.min(QTD, p.estoque);
  const item = { ...p.carrinho.itens[0], quantidade: String(qtd) };
  const cart = await call("/checkout/cart");
  const jaTem = (cart.itens || []).some((i) => Number(i.hash.idProduto) === Number(p.id));
  const res = jaTem
    ? await call("/checkout/cart", "PUT", { ...item, tipo: "produto" })
    : await call("/checkout/cart", "POST", { ...p.carrinho, itens: [item] });

  console.log(`✅ carrinho: ${res.quantidadeTotal} un | subtotal R$ ${res.subtotal?.valor}`);
  console.log("recarregue a página ou vá em https://cartinha.com.br/checkout/carrinho");
})("https://cartinha.com.br/unitario-one-piece-tcg-the-worlds-strongest-warriors-booster-display-op-17.html", 3);
