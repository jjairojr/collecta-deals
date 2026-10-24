import csv,json,os,sys,urllib.request,datetime
g=sys.argv[1]; short={'pokemon':'pkm','riftbound':'rft'}[g]
B='https://collecta-deals-production.up.railway.app'
T=os.environ.get('ADMIN_TOKEN') or next(l.split('=',1)[1].strip() for l in open(os.path.join(os.path.dirname(os.path.abspath(__file__)),'..','deploy.env')) if l.startswith('ADMIN_TOKEN='))
AUTH={'X-Admin-Token':T}
rows=[r for r in csv.DictReader(open(f'{short}-reprecificado-{datetime.date.today().isoformat()}.csv')) if r['acao'] in ('abaixar','abaixar_margem','prejuizo_atual','subir')]
cur={t['id']:t for t in json.load(urllib.request.urlopen(urllib.request.Request(f'{B}/api/trades?game={g}',headers=AUTH)))['trades']}
listings=[];liga=[]
for r in rows:
    t=cur.get(r['id'])
    if not t: print('sem trade',r['id'],r['nome']); continue
    p=float(r['preco_novo'])
    listings.append({'id':r['id'],'askBRL':p,'listed':t.get('listed',False),'imageURL':t.get('imageURL',''),'featured':t.get('featured',False)})
    liga.append({'id':r['id'],'ligaListed':True,'ligaQty':int(r['liga_qty']),'ligaPriceBRL':p})
for path,items in (('listings',listings),('liga',liga)):
    req=urllib.request.Request(f'{B}/api/trades/{path}?game={g}',data=json.dumps({'items':items}).encode(),headers={'Content-Type':'application/json',**AUTH},method='POST')
    print(path,urllib.request.urlopen(req).read()[:200])
