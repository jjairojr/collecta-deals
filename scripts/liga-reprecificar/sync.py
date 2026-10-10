import csv,json,sys,urllib.request,datetime
g=sys.argv[1]; short={'pokemon':'pkm','riftbound':'rft'}[g]
B='https://collecta-deals-production.up.railway.app'
rows=[r for r in csv.DictReader(open(f'{short}-reprecificado-{datetime.date.today().isoformat()}.csv')) if r['acao']=='abaixar']
cur={t['id']:t for t in json.load(urllib.request.urlopen(f'{B}/api/trades?game={g}'))['trades']}
listings=[];liga=[]
for r in rows:
    t=cur.get(r['id'])
    if not t: print('sem trade',r['id'],r['nome']); continue
    p=float(r['preco_novo'])
    listings.append({'id':r['id'],'askBRL':p,'listed':t.get('listed',False),'imageURL':t.get('imageURL',''),'featured':t.get('featured',False)})
    liga.append({'id':r['id'],'ligaListed':True,'ligaQty':int(r['liga_qty']),'ligaPriceBRL':p})
for path,items in (('listings',listings),('liga',liga)):
    req=urllib.request.Request(f'{B}/api/trades/{path}?game={g}',data=json.dumps({'items':items}).encode(),headers={'Content-Type':'application/json'},method='POST')
    print(path,urllib.request.urlopen(req).read()[:200])
