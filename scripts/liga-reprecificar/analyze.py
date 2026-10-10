import json,csv,sys,re,collections,glob,datetime
g=sys.argv[1]
S=sys.argv[2] if len(sys.argv)>2 else '.'
XLANG='--xlang' in sys.argv
MARGIN=next((float(a.split('=')[1]) for a in sys.argv if a.startswith('--min-margin=')),None)
TODAY=datetime.date.today().isoformat()
short={'pokemon':'pkm','riftbound':'rft','onepiece':'op'}[g]
rows=json.load(open(f'{S}/floors-{g}.json'))
COLLECTA=866280
FEE=0.0499
LANG={'2':'EN','6':'JP','8':'PT','11':'PT/EN'}
prev={}
for f in sorted(glob.glob(f'{short}-reprecificado-*.csv')):
    if f.endswith(f'{TODAY}.csv'): continue
    for r in csv.DictReader(open(f)):
        if not r.get('edid'): continue
        prev[r['id']]=r
        prev[(r['set'],r['num'],r['idioma'])]=r
def lang_hint(v):
    v=(v or '').upper()
    for l in ('JP','PT','EN'):
        if re.search(r'\b'+l+r'\b',v): return l
    return None
def net(p): return p*(1-FEE)
out=[]; stats=collections.Counter(); notlisted=[]; multi=[]
for r in rows:
    t=r['trade']; offers=r['offers'] or []
    ours=[o for o in offers if o['storeId']==COLLECTA]
    if not ours:
        stats['sem_oferta_collecta']+=1
        if t.get('ligaListed'): notlisted.append(t)
        continue
    hint=lang_hint(t.get('variant'))
    if len(ours)>1:
        cand=[o for o in ours if LANG.get(o['language'])==hint] if hint else []
        if len(cand)==1: ours=cand
        else:
            multi.append((t['name'],t['number'],t.get('variant'),[(o['language'],o['extras'],o['priceBRL'],o['qty']) for o in ours]))
            ours=[min(ours,key=lambda o:o['priceBRL'])]
    o=ours[0]
    price=o['priceBRL'] if o['priceKnown'] else (t.get('ligaPriceBRL') or t.get('askBRL') or 0)
    cost=(t.get('costBRL') or 0)/max(t.get('qty') or 1,1)
    lang=LANG.get(o['language'],o['language'])
    base=[c for c in offers if c['storeId']!=COLLECTA and c['priceKnown'] and c['condition']=='2' and c['extras']==o['extras']]
    same=sorted([c for c in base if c['language']==o['language']],key=lambda c:c['priceBRL'])
    xl=XLANG and lang in ('EN','JP')
    pt=sorted([c for c in base if c['language']=='8'],key=lambda c:c['priceBRL'])
    comp=sorted(same+pt,key=lambda c:c['priceBRL']) if xl else same
    idr=prev.get(t['id']) or prev.get((t.get('set'),t['number'],lang)) or {}
    rec=dict(acao='',set=t.get('set'),num=t['number'],nome=t['name'],variante=t.get('variant') or '',idioma=lang,extras=o['extras'],
             qtd=t['qty'],liga_qty=o['qty'],custo_un=round(cost,2),preco_antigo=price,preco_novo='',piso='',piso_idioma='',piso_qty='',ofertas=len(comp),rank='',
             piso_mesma_lingua=same[0]['priceBRL'] if same else '',ofertas_mesma_lingua=len(same),piso_pt=pt[0]['priceBRL'] if pt else '',piso_pt_qty=pt[0]['qty'] if pt else '',ofertas_pt=len(pt),motivo='',
             id=t['id'],edid=idr.get('edid',''),cardid=idr.get('cardid',''),url=r['url'])
    if not comp:
        rec['acao']='sozinho'
    else:
        floor=comp[0]['priceBRL']; rec['piso']=floor; rec['piso_qty']=comp[0]['qty']; rec['piso_idioma']=LANG.get(comp[0]['language'],comp[0]['language'])
        rec['rank']=1+sum(1 for c in comp if c['priceBRL']<price)
        if price<=floor:
            rec['acao']='menor'
            gap=round(floor-0.01-price,2)
            if gap>=0.05 and len(comp)>=3: rec['motivo']=f'subir +{gap:.2f} ainda menor'
        else:
            target=round(floor-0.01,2); rec['preco_novo']=target
            if net(target)<cost:
                rec['acao']='abaixo_custo'; rec['motivo']=f'liquido {net(target):.2f} < custo {cost:.2f}'
                if MARGIN is not None and lang=='EN':
                    minp=round(cost*(1+MARGIN)/(1-FEE)+0.005,2)
                    if minp<price:
                        rec['acao']='abaixar_margem'; rec['preco_novo']=minp
                        rec['motivo']=f'piso {floor:.2f} daria prejuizo; preco minimo com {MARGIN:.0%} liquido sobre custo {cost:.2f}'
                    elif net(price)>=cost:
                        rec['acao']='segurar'; rec['preco_novo']=''
                        rec['motivo']=f'ja abaixo de {MARGIN:.0%} de margem (liquido {net(price):.2f} vs custo {cost:.2f}); piso {floor:.2f} daria prejuizo'
                    else:
                        rec['acao']='prejuizo_atual'; rec['preco_novo']=minp
                        rec['motivo']=f'preco atual ja da prejuizo (liquido {net(price):.2f} < custo {cost:.2f}); subir para {minp:.2f}'
            else:
                rec['acao']='abaixar'
    if hint and hint!=lang and rec['acao']!='duplicado':
        stats[rec['acao']]-=0
        rec['motivo']=f"ledger diz {hint}, oferta na Liga esta em {lang} ({rec['acao']} -> {rec['preco_novo'] or '-'}); conferir cadastro antes de mexer"
        rec['acao']='conferir_idioma'; rec['preco_novo']=''
    stats[rec['acao']]+=1
    out.append(rec)
groups=collections.defaultdict(list)
for r in out: groups[(r['url'],r['idioma'],r['extras'],r['preco_antigo'])].append(r)
for g_,rs in groups.items():
    if len(rs)<2 or len({r['preco_novo'] for r in rs})<2: continue
    keep=max(rs,key=lambda r:(lang_hint(r['variante'])==r['idioma'],r['custo_un']))
    for r in rs:
        if r is not keep:
            stats[r['acao']]-=1; r['acao']='duplicado'; r['preco_novo']=''; r['motivo']=f"mesma oferta da Liga que a trade {keep['id']} (custo {keep['custo_un']}); alvo dela: {keep['preco_novo'] or 'segurar'}"; stats['duplicado']+=1
cols=list(out[0].keys())
w=csv.DictWriter(open(f'{short}-reprecificado-{TODAY}.csv','w'),fieldnames=cols); w.writeheader(); w.writerows(out)
apply=[[r['nome'],r['num'],r['idioma'],r['preco_antigo'],r['preco_novo'],r['edid'],r['cardid']] for r in out if r['acao'] in ('abaixar','abaixar_margem')]
json.dump(apply,open(f'{S}/apply-{short}.json','w'),ensure_ascii=False)
print(g,'xlang' if XLANG else 'mesma língua',dict(stats))
print('ligaListed sem oferta COLLECTA na página:',len(notlisted),[(t['name'],t['number']) for t in notlisted])
print('ambíguos (várias ofertas COLLECTA):',multi)
noid=[r for r in out if r['acao']=='abaixar' and not r['edid']]
print('abaixar sem edid/cardid:',len(noid),[(r['nome'],r['num']) for r in noid])
print()
for r in out:
    if r['acao'] in('abaixar','abaixo_custo','abaixar_margem','segurar','prejuizo_atual'):
        print(f"{r['acao']:<12} {r['set']:<6} {r['num']:<8} {r['nome'][:30]:<30} {r['idioma']:<5} ex={r['extras']} q={r['qtd']} rank={r['rank']:<3} {r['preco_antigo']:>8.2f} -> {r['preco_novo'] or '-':>8} piso={r['piso']} {r['piso_idioma']} ({r['piso_qty']}un/{r['ofertas']}of) msm={r['piso_mesma_lingua'] or '-'} custo={r['custo_un']} {r['motivo']}")
subir=[r for r in out if r['motivo'].startswith('subir')]
print('\nsubir possível:',len(subir))
for r in subir: print(f"  {r['set']:<6} {r['num']:<8} {r['nome'][:30]:<30} {r['idioma']:<5} {r['preco_antigo']:>8.2f} piso={r['piso']} {r['piso_idioma']} {r['motivo']}")
