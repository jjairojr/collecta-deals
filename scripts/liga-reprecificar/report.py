import csv,sys,datetime,collections,os
TODAY=datetime.date.today().isoformat()
FEE=0.0499
ORDER={'subir':-1,'prejuizo_atual':0,'abaixar_margem':1,'segurar':2,'abaixo_custo':3,'abaixar':4,'conferir_idioma':5,'duplicado':6,'menor':7,'sozinho':8}
LABEL={'prejuizo_atual':'PREJUÍZO HOJE → subir','abaixar_margem':'ABAIXAR (margem 10%)','segurar':'segurar','abaixo_custo':'abaixo do custo (parado)','abaixar':'ABAIXAR','conferir_idioma':'conferir idioma','duplicado':'duplicado','menor':'menor','sozinho':'sozinho','subir':'SUBIR','igual':'igual','fora_do_idioma':'fora do idioma'}
def f(v):
    try: return f'{float(v):,.2f}'.replace(',','X').replace('.',',').replace('X','.')
    except (TypeError,ValueError): return '—'
def piso(p,q,n): return f'{f(p)} ({q}un/{n})' if p else '—'
out=[f'# Reprecificação Liga {TODAY}','',
     'Piso = concorrente mais barato NM no mesmo acabamento (unidades dele / nº de ofertas). Novo = piso usado − 0,01; Pokémon EN/JP com piso na própria língua abaixo de R$ 150 usa min(piso EN/JP, piso PT); acima de R$ 150 usa min(piso EN/JP, MEDIANA das ofertas PT NM) (regra do dono em 2026-10-10); PT só PT. EN com piso abaixo do custo: novo = mínimo com 10% líquido sobre o custo.',
     '',f'Custo: o ledger de prod foi recriado em 2026-10-07 a partir do export da Liga com custo 0; onde existe, o custo vem do CSV da rodada de 2026-09-30 (`custo_origem = csv-anterior`). Itens sem custo conhecido não têm trava de prejuízo.','']
for g,short in (('Pokémon','pkm'),('Riftbound','rft')):
    if not os.path.exists(f'{short}-reprecificado-{TODAY}.csv'): continue
    rows=list(csv.DictReader(open(f'{short}-reprecificado-{TODAY}.csv')))
    rows.sort(key=lambda r:(ORDER.get(r['acao'],9),-float(r['preco_antigo'] or 0)))
    st=collections.Counter(r['acao'] for r in rows)
    delta=sum((float(r['preco_novo'])-float(r['preco_antigo']))*int(r['qtd']) for r in rows if r['preco_novo'])
    semcusto=sum(1 for r in rows if r['acao'] in('abaixar',) and not r['custo_origem'])
    out+= [f'### {g} ({len(rows)})','',
           f"Ações: {', '.join(f'{LABEL[k]} {v}' for k,v in sorted(st.items(),key=lambda kv:ORDER.get(kv[0],9)))}. Variação de receita se tudo for gravado: R$ {f(delta)}. Descidas sem custo conhecido: {semcusto}.",'']
    out.append('| Set | Nº | Carta | Lng | Acab | Qtd | Custo | Atual | Rank | Piso mesma lng | Piso PT | Mediana PT | Regra | Novo | Margem | Ação |')
    out.append('|---|---|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---:|---:|---|')
    for r in rows:
        ex={'0':'Normal','2':'Foil','6':'Foil+Rev'}.get(r['extras'],r['extras'])
        cost=float(r['custo_un'] or 0); novo=float(r['preco_novo']) if r['preco_novo'] else None
        base=novo if novo else float(r['preco_antigo'] or 0)
        margem=f'{(base*(1-FEE)-cost)/cost:.0%}' if cost and base else ''
        custo=f(cost) if cost else '?'
        if r['custo_origem']=='csv-anterior': custo+='*'
        acao=LABEL.get(r['acao'],r['acao'])+(' · '+r['motivo'] if r['motivo'] else '')
        out.append(f"| {r['set']} | {r['num']} | {r['nome']} | {r['idioma']} | {ex} | {r['qtd']} | {custo} | {f(r['preco_antigo'])} | {r['rank'] or ''} | {piso(r['piso_mesma_lingua'],'' ,r['ofertas_mesma_lingua']) if r['piso_mesma_lingua'] else '—'} | {piso(r['piso_pt'],r['piso_pt_qty'],r['ofertas_pt']) if r['piso_pt'] else '—'} | {f(r.get('mediana_pt')) if r.get('mediana_pt') else '—'} | {r.get('regra','')} | {f(novo) if novo else ''} | {margem} | {acao} |")
    out.append('')
open(f'reprecificacao-{TODAY}.md','w').write('\n'.join(out))
print('\n'.join(l for l in out if l.startswith('###') or l.startswith('Ações')))
