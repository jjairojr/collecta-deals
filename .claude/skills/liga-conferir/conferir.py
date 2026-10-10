import csv,json,re,sys,collections
g,tcg=sys.argv[1],sys.argv[2]
def num(n): return re.sub(r'^0+(?=\d)','',re.sub(r'/.*','',(n or '').strip().upper()))
def norm(s): return re.sub(r'[^a-z0-9]','',(s or '').lower())
def brl(s): return float((s or '0').replace('.','').replace(',','.') or 0)
SETMAP={'151':'MEW','CRZ:GG':'CZGG','HIF:SV':'HIF','SWSH09':'BRS','SWSH10':'ASR','SWSH12':'SIT','SWSH12:TG':'SITTG','SM01':'SUM','SVI':'SV1','MCAP':'SVP','CRZ':'CRZ'}
def sig(s):
    s=(s or '').strip()
    m=re.search(r'\((\w+)\)\s*$',s)
    if m: return m.group(1)
    if '2021' in s: return 'CEL'
    if '2011' in s: return 'BLW'
    return SETMAP.get(s,s)
def lang_of(t):
    v=(t.get('variant') or '').upper()
    for l in ('JP','PT','EN'):
        if re.search(r'\b'+l+r'\b',v): return l
    return None
rows=[r for r in csv.reader(open(f'exp-{g}.csv',encoding='utf-8',errors='replace')) if r and r[0]==tcg]
liga=[]
for r in rows:
    q=int(r[12] or 0)
    if q<=0: continue
    liga.append({'edid':r[1],'sig':r[2],'cardid':r[3],'num':num(r[4]),'rawnum':r[4],'name':r[8] or r[9],'nameEN':r[9],'lang':r[10],'cond':r[11],'qty':q,'price':brl(r[14]),'used':0})
def base(n):
    p=(n or '').strip().upper().split('-')
    return '-'.join(p[:2]) if g=='onepiece' else num(n)
def cname(s):
    s=re.sub(r'\s*-\s*[A-Z]{1,4}\d{0,2}-\d{3}\S*','',s or '')
    s=re.sub(r'\((?:#)?[A-Z]{0,4}\d{0,2}-?\d{3}[^)]*\)','',s)
    return norm(s)
by=collections.defaultdict(list); bynum=collections.defaultdict(list)
for l in liga:
    l['base']=base(l['rawnum']); l['cn']=cname(l['nameEN']) or cname(l['name'])
    by[(l['sig'].upper().replace('-',''),l['base'])].append(l); bynum[l['base']].append(l)
T=json.load(open(f'trades-{g}.json'))['trades']
hold=[t for t in T if t.get('status')=='holding' and t.get('qty') and t.get('set') not in ('ACESSORIO','SEALED')]
import collections as _c
hold.sort(key=lambda t:(0 if t.get('ligaListed') else 1, 0 if (t.get('askBRL') or 0)>0 else 1))
hn=_c.Counter((cname(t['name']),lang_of(t)) for t in hold)
res=collections.Counter(); out=collections.defaultdict(list); updates=[]
def pick(c,t,n,lg,ask):
    if lg: c=[l for l in c if l['lang']==lg] or c
    if len(c)>1:
        cn=cname(t['name']); c2=[l for l in c if l['cn']==cn]
        if c2: c=c2
    if len(c)>1 and g=='onepiece':
        full=n.strip().upper()
        c2=[l for l in c if l['rawnum'].upper()==full]
        if c2: c=c2
    if len(c)>1:
        c2=[l for l in c if abs(l['price']-ask)<0.005] or [l for l in c if abs(l['price']-float(t.get('ligaPriceBRL') or 0))<0.005]
        if c2: c=c2
    if len(c)>1:
        c2=[l for l in c if not l['used']]
        if c2: c=c2
    return c
for t in hold:
    n=t.get('number') or ''
    if not n:
        m=re.search(r'\(#?([^)]+)\)',t['name']); n=m.group(1) if m else ''
    lg=lang_of(t); ask=float(t.get('askBRL') or 0)
    c=pick(bynum.get(base(n),[]) if g=='onepiece' else by.get((sig(t['set']).upper().replace('-',''),base(n)),[]),t,n,lg,ask)
    dupnote=None
    if c and all(l['used'] for l in c):
        dupnote=c[0]; c=[]
    if not c and g!='onepiece':
        c=pick([l for l in bynum.get(base(n),[]) if l['cn']==cname(t['name']) and not l['used']],t,n,lg,ask)
        if not c and not ask and lg and hn[(cname(t['name']),lg)]==1:
            cn=cname(t['name']); c=[l for l in liga if l['cn']==cn and l['lang']==lg and not l['used']]
            if len(c)!=1: c=[]
        if c: out['casado_por_nome'].append(f"{t['name'][:30]} {t['set']} {n} -> Liga {c[0]['sig']} {c[0]['rawnum']} {c[0]['nameEN'][:25]}")
    if not c and g!='onepiece':
        lp=float(t.get('ligaPriceBRL') or 0)
        c=[l for l in bynum.get(base(n),[]) if (not lg or l['lang']==lg) and l["price"]>0 and (abs(l["price"]-ask)<0.005 or abs(l["price"]-lp)<0.005) and not l["used"]]
        if not c:
            cn=cname(t['name'])
            c=[l for l in liga if l['cn']==cn and (not lg or l['lang']==lg) and l["price"]>0 and (abs(l["price"]-ask)<0.005 or abs(l["price"]-lp)<0.005) and not l["used"]]
        if c: out['casado_por_nome'].append(f"{t['name'][:30]} {t['set']} {n} {t.get('variant') or ''} ask {ask} -> Liga {c[0]['sig']} {c[0]['rawnum']} {c[0]['nameEN'][:25]} {c[0]['lang']} R${c[0]['price']}")
    if len(c)==1 and c[0]['used']:
        dupnote=c[0]; c=[]
    if not c and dupnote:
        out['dash_duplicado'].append(f"{t['name'][:30]} {t['set']} {n} {t.get('variant') or ''} q{t['qty']} ask {ask} -> Liga {dupnote['sig']} {dupnote['rawnum']} {dupnote['lang']} q{dupnote['qty']} ja casada com outro registro")
    if g=='onepiece' and len(c)==1 and c[0]['cn']!=cname(t['name']) and c[0]['rawnum'].upper()!=n.strip().upper():
        out['nome_diverge'].append(f"{t['name'][:34]} {n} <-> Liga {c[0]['rawnum']} {c[0]['nameEN'][:34]}")
    tag=f"{t['name'][:32]:<33}{t['set'][:6]:<7}{n[:12]:<13}{(t.get('variant') or '')[:10]:<11}"
    if len(c)>1:
        res['ambiguo']+=1; out['ambiguo'].append(tag+' | '+'; '.join(f"{l['sig']} {l['rawnum']} {l['lang']} q{l['qty']} R${l['price']}" for l in c)); continue
    listed=bool(t.get('ligaListed'))
    if not c:
        if not ask and not listed:
            res['sem_preco_fora']+=1; continue
        if listed:
            res['flag_mentindo_sumiu']+=1; out['flag_mentindo_sumiu'].append(tag+f" dash q{t['qty']} ask {ask} | flag q{t.get('ligaQty')} R${t.get('ligaPriceBRL')}")
            updates.append({'id':t['id'],'ligaListed':False})
        else:
            res['falta_cadastrar']+=1; out['falta_cadastrar'].append(tag+f" q{t['qty']} ask {ask}")
        continue
    l=c[0]; l['used']+=1
    same=(int(t.get('ligaQty') or 0)==l['qty'] and abs(float(t.get('ligaPriceBRL') or 0)-l['price'])<0.005)
    if not listed:
        res['flag_mentindo_cadastrado']+=1; out['flag_mentindo_cadastrado'].append(tag+f" | Liga {l['sig']} {l['rawnum']} {l['lang']} q{l['qty']} R${l['price']} (dash q{t['qty']} ask {ask})")
        updates.append({'id':t['id'],'ligaListed':True,'ligaQty':l['qty'],'ligaPriceBRL':l['price']})
    elif same:
        res['ok']+=1
    else:
        res['flag_desatualizada']+=1; out['flag_desatualizada'].append(tag+f" flag q{t.get('ligaQty')} R${t.get('ligaPriceBRL')} -> Liga q{l['qty']} R${l['price']} (dash q{t['qty']} ask {ask})")
        updates.append({'id':t['id'],'ligaListed':True,'ligaQty':l['qty'],'ligaPriceBRL':l['price']})
    if not ask and c:
        out['na_liga_sem_preco_dash'].append(tag+f" | Liga {l['sig']} {l['rawnum']} {l['lang']} q{l['qty']} R${l['price']}")
    if t['qty']!=l['qty'] or abs(ask-l['price'])>=0.005:
        out['dash_vs_liga'].append(tag+f" dash q{t['qty']} R${ask} | Liga q{l['qty']} R${l['price']}")
orph=[l for l in liga if not l['used']]
dup=[l for l in liga if l['used']>1]
print(f"=== {g.upper()}: dash com preço {len(hold)} | Liga com estoque {len(liga)} | {dict(res)} | Liga sem holding {len(orph)} | Liga usada 2x {len(dup)}")
for k in ['dash_duplicado','falta_cadastrar','nome_diverge','na_liga_sem_preco_dash','flag_desatualizada','flag_mentindo_sumiu','flag_mentindo_cadastrado','ambiguo','casado_por_nome','dash_vs_liga']:
    if out[k]:
        print(f"-- {k} ({len(out[k])})")
        for x in out[k]: print('   ',x)
if orph:
    print(f"-- Liga com estoque sem holding no dash ({len(orph)})")
    for l in orph: print(f"    {l['sig']:<6}{l['rawnum']:<12}{l['nameEN'][:32]:<33}{l['lang']:<4}q{l['qty']:<3}R${l['price']}")
if dup:
    print('-- Liga usada 2x:',[(l['sig'],l['rawnum'],l['nameEN'][:20]) for l in dup])
json.dump(updates,open(f'flag-updates-{g}.json','w'))
print('updates',len(updates))
