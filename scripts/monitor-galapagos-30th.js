(() => {
  const CATEGORY = 'PRE-30TH';
  const CATALOG = 'B2B-Hobby';
  const PLG = 'Hobby-Low';
  const INTERVAL_MS = 3 * 60 * 1000;
  const ARM_CART = true;
  const BEEP_HZ = 880;
  const BEEP_SEC = 0.25;
  const CYCLE_SEC = 1.2;
  const VOLUME = 0.35;

  const rc = require('ccRestClient');
  const ps = require('pubsub');
  const $ = require('jquery');
  const seen = new Map();
  const baseTitle = document.title;
  let ticks = 0;
  let audio = null;
  let source = null;
  let titleTimer = null;

  const hasStock = p => p.x_hasStock === true || p.x_hasStock === 'true';

  const targetQty = (p) => {
    const limit = Number(p.x_purchaseLimit) || 0;
    const mult = Number(p.x_quantidadeDoMltiplo) || 1;
    const min = Number(p.x_minQtdB2B) || 1;
    if (!limit) return 0;
    const q = Math.floor(limit / mult) * mult;
    return q >= min ? q : 0;
  };

  const stepsDown = (p) => {
    const mult = Number(p.x_quantidadeDoMltiplo) || 1;
    const min = Number(p.x_minQtdB2B) || 1;
    const out = [];
    for (let q = targetQty(p); q >= min; q -= mult) out.push(q);
    return out;
  };

  const buildLoopBuffer = (ctx) => {
    const buf = ctx.createBuffer(1, Math.round(ctx.sampleRate * CYCLE_SEC), ctx.sampleRate);
    const data = buf.getChannelData(0);
    const tone = Math.round(ctx.sampleRate * BEEP_SEC);
    const fade = Math.round(ctx.sampleRate * 0.008);
    for (let i = 0; i < tone; i++) {
      let env = VOLUME;
      if (i < fade) env *= i / fade;
      else if (i > tone - fade) env *= (tone - i) / fade;
      data[i] = Math.sin((2 * Math.PI * BEEP_HZ * i) / ctx.sampleRate) * env;
    }
    return buf;
  };

  const alarmOn = () => source !== null || titleTimer !== null;

  const startAlarm = (label) => {
    if (alarmOn()) return;
    try {
      audio = audio || new (window.AudioContext || window.webkitAudioContext)();
      if (audio.state === 'suspended') audio.resume();
      source = audio.createBufferSource();
      source.buffer = buildLoopBuffer(audio);
      source.loop = true;
      source.connect(audio.destination);
      source.start();
    } catch (e) {
      source = null;
      console.warn('[30th] audio indisponivel, alarme sera apenas visual', e);
    }
    let flip = false;
    titleTimer = setInterval(() => {
      flip = !flip;
      document.title = flip ? '>>> RESTOQUE ' + label : '!!! ' + label;
    }, 700);
    document.title = '>>> RESTOQUE ' + label;
  };

  const ack = (label) => {
    if (!alarmOn()) return;
    if (source) {
      try { source.stop(); } catch (e) { /* noop */ }
      source.disconnect();
      source = null;
    }
    clearInterval(titleTimer);
    titleTimer = null;
    document.title = label ? 'RESTOQUE ' + label : baseTitle;
    console.log('[30th] alarme silenciado — monitor continua rodando');
  };

  const readCart = () => new Promise((resolve) => {
    rc.request('getCartItems', {}, d => resolve(d), () => resolve(null));
  });

  const cartQty = (cart, sku) => {
    const items = (cart && cart.items) || [];
    return items
      .filter(i => i.productId === sku || i.catRefId === sku)
      .reduce((a, i) => a + (Number(i.quantity) || 0), 0);
  };

  const addToCart = async (p) => {
    const sku = p.repositoryId;
    const before = cartQty(await readCart(), sku);
    for (const qty of stepsDown(p)) {
      const prod = Object.assign({}, p, { orderQuantity: qty, stockState: 'IN_STOCK' });
      try {
        $.Topic(ps.topicNames.CART_ADD).publishWith(prod, [{ message: 'success' }]);
      } catch (e) {
        return { sku, alvo: qty, obtido: 0, erro: 'publish falhou: ' + e.message };
      }
      await new Promise(r => setTimeout(r, 2500));
      const now = cartQty(await readCart(), sku);
      if (now > before) return { sku, alvo: targetQty(p), obtido: now - before, tentativa: qty };
    }
    return { sku, alvo: targetQty(p), obtido: 0, erro: 'nenhuma quantidade aceita' };
  };

  const announce = async (list) => {
    const label = list.map(p => p.sku).join(',');
    const msg = list.map(p => `${p.sku} — ${p.name}`).join('\n');
    console.log('%c### RESTOQUE ###\n' + msg, 'color:#0a0;font-size:16px;font-weight:bold');
    startAlarm(label);
    if (window.Notification && Notification.permission === 'granted') {
      const n = new Notification('Galápagos 30th — restoque!', { body: msg, requireInteraction: true });
      n.onclick = () => { window.focus(); ack(label); };
    }
    window.__g30ack = () => ack(label);
    ['pointerdown', 'keydown'].forEach(ev => {
      window.addEventListener(ev, () => ack(label), { capture: true, once: true });
    });

    if (!ARM_CART) return;
    const report = [];
    for (const p of list) {
      if (!p.product) continue;
      report.push(await addToCart(p.product));
    }
    if (report.length) {
      console.log('%c### CARRINHO ###', 'color:#06c;font-size:14px;font-weight:bold');
      console.table(report);
      const falhou = report.filter(r => !r.obtido);
      const parcial = report.filter(r => r.obtido && r.obtido < r.alvo);
      if (falhou.length) console.warn('[30th] NAO entraram no carrinho: ' + falhou.map(r => r.sku).join(', '));
      if (parcial.length) console.warn('[30th] entraram PARCIAIS: ' + parcial.map(r => `${r.sku} ${r.obtido}/${r.alvo}`).join(', '));
      console.log('[30th] carrinho montado — CONFIRA e finalize manualmente. O script nao fecha pedido.');
    }
  };

  const fetchStock = () => new Promise((resolve, reject) => {
    rc.request(
      'listProducts',
      { categoryId: CATEGORY, catalogId: CATALOG, includeChildren: true, limit: 100, offset: 0, storePriceListGroupId: PLG },
      d => resolve(d.items || []),
      e => reject(e),
    );
  });

  const tick = async () => {
    ticks++;
    let items;
    try {
      items = await fetchStock();
    } catch (e) {
      console.warn('[30th] falha na consulta (sessão expirada? recarregue a aba)', e);
      return;
    }
    const flipped = [];
    for (const p of items) {
      const sku = p.repositoryId;
      const has = hasStock(p);
      if (seen.get(sku) === false && has) flipped.push({ sku, name: p.displayName.trim(), product: p });
      seen.set(sku, has);
    }
    const inStock = items.filter(hasStock).map(p => p.repositoryId);
    console.log(`[30th #${ticks}] ${new Date().toLocaleTimeString('pt-BR')} — ${items.length} SKUs, com estoque: ${inStock.length ? inStock.join(', ') : 'nenhum'}`);
    if (flipped.length) await announce(flipped);
  };

  const plan = async () => {
    const items = await fetchStock();
    console.table(items.map(p => ({
      sku: p.repositoryId,
      alvo: targetQty(p),
      cota: p.x_purchaseLimit,
      min: p.x_minQtdB2B,
      mult: p.x_quantidadeDoMltiplo,
      estoque: p.x_hasStock,
    })));
  };

  if (window.__g30) clearInterval(window.__g30);
  if (window.Notification && Notification.permission === 'default') Notification.requestPermission();
  window.__g30ack = () => ack(null);
  window.__g30test = () => announce([{ sku: 'TESTE', name: 'simulacao de alarme' }]);
  window.__g30plan = plan;
  tick();
  window.__g30 = setInterval(tick, INTERVAL_MS);
  console.log(`[30th] monitor ativo — checando a cada ${INTERVAL_MS / 60000} min. ARM_CART=${ARM_CART}
  ver plano de quantidades: window.__g30plan()
  silenciar alarme: window.__g30ack()
  testar alarme: window.__g30test()
  parar monitor: clearInterval(window.__g30)`);
})();
