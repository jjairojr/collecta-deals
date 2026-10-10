function __fill(dry){
  const T = JSON.parse(localStorage.__T || '[]');
  const rows = [...document.querySelectorAll('[name^="h_id["]')].map(el => {
    const i = el.name.match(/\[(\d+)\]/)[1];
    const q = n => document.querySelector(`[name="${n}[${i}]"]`);
    return {edid: q('h_ide_edicao')?.value, cardid: q('h_ide_carta')?.value, idioma: q('txt_idioma')?.value, preco: q('txt_preco'), num: q('h_numero_carta')?.value, tr: el.closest('tr')};
  });
  const L = {EN:'2', JP:'6', PT:'8', 'PT/EN':'11'};
  const brl = s => parseFloat((s||'0').replace(/\./g,'').replace(',','.'));
  const fmt = v => v.toFixed(2).replace('.', ',');
  let done = 0, already = 0, miss = [];
  for (const [nome, num, idioma, antigo, novo, edid, cardid] of T) {
    const cand = rows.filter(r => r.idioma === L[idioma] && (edid ? (r.edid == edid && r.cardid == cardid) : num && r.num ? (r.num === num || r.num.startsWith(num + '-')) : (r.tr?.innerText || '').includes(nome)));
    if (!cand.length) continue;
    if (cand.some(r => Math.abs(brl(r.preco.value) - novo) < 0.005)) { already++; continue; }
    const atOld = cand.filter(r => Math.abs(brl(r.preco.value) - antigo) < 0.005);
    if (atOld.length !== 1) { miss.push([nome, num, cand.map(r => r.preco.value)]); continue; }
    done++;
    if (dry) continue;
    const pr = atOld[0].preco;
    pr.focus(); pr.value = fmt(novo);
    pr.dispatchEvent(new Event('input', {bubbles:true})); pr.dispatchEvent(new Event('change', {bubbles:true})); pr.blur();
  }
  const msg = (document.body.innerText.match(/[^\n]*alterados com sucesso[^\n]*/) || [''])[0];
  return JSON.stringify({page: new URL(location.href).searchParams.get('page'), rows: rows.length, done, already, miss, msg});
}
