package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"opdeals/internal/tracking"
)

type listaSite struct {
	srv      *httptest.Server
	requests []string
}

type ligaFixtureCard struct {
	number string
	name   string
	before float64
	now    float64
	stores int
	sold   int
	image  bool
}

func newListaFixture(t *testing.T) (*fixture, *pecaSite, *listaSite) {
	t.Helper()
	f, site := newPecaFixture(t)
	photo := pngBytes(t, 100, 140)
	ls := &listaSite{}
	groups := map[int]string{
		3: `[
			{"groupId": 24831, "name": "ME06: Delta Reign", "abbreviation": "DLR", "isSupplemental": false, "publishedOn": "2026-11-06T00:00:00"},
			{"groupId": 24938, "name": "McDonald's Promos 2026", "abbreviation": "M26", "isSupplemental": true, "publishedOn": "2026-11-06T00:00:00"},
			{"groupId": 1423, "name": "Nintendo Promos", "abbreviation": "PR", "isSupplemental": false, "publishedOn": "2026-11-06T20:00:05.5949439Z"},
			{"groupId": 24900, "name": "Delta Reign Prerelease Cards", "abbreviation": "DLRP", "isSupplemental": false, "publishedOn": "2026-11-01T00:00:00"},
			{"groupId": 24700, "name": "ME05: Ascended Heroes", "abbreviation": "ASC", "isSupplemental": false, "publishedOn": "2026-09-26T00:00:00"}
		]`,
		71: `[
			{"groupId": 24740, "name": "Hyperia City", "abbreviation": "14", "isSupplemental": false, "publishedOn": "2026-10-16T00:00:00"},
			{"groupId": 24990, "name": "Winterspell", "abbreviation": "15", "isSupplemental": false, "publishedOn": "2026-12-12T00:00:00"}
		]`,
	}
	ls.srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ls.requests = append(ls.requests, r.URL.Path)
		var cat, group int
		switch {
		case strings.HasSuffix(r.URL.Path, ".png"), strings.HasSuffix(r.URL.Path, ".jpg"):
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(photo)
		case r.URL.Path == "/noticia":
			_, _ = w.Write([]byte("<html>ok</html>"))
		case r.URL.Path == "/bloqueia":
			w.WriteHeader(http.StatusForbidden)
		case strings.HasSuffix(r.URL.Path, "/groups"):
			if _, err := fmt.Sscanf(r.URL.Path, "/tcgplayer/%d/groups", &cat); err != nil || groups[cat] == "" {
				http.NotFound(w, r)
				return
			}
			_, _ = fmt.Fprintf(w, `{"success": true, "results": %s}`, groups[cat])
		case strings.HasSuffix(r.URL.Path, "/products"):
			if _, err := fmt.Sscanf(r.URL.Path, "/tcgplayer/%d/%d/products", &cat, &group); err != nil {
				http.NotFound(w, r)
				return
			}
			img := func(id int) string { return fmt.Sprintf("%s/product/%d_200w.jpg", ls.srv.URL, id) }
			_, _ = fmt.Fprintf(w, `{"success": true, "results": [
				{"productId": 1, "name": "Some Card", "imageUrl": "%s", "extendedData": [{"name": "Number"}]},
				{"productId": 2, "name": "Set Booster Box Case", "imageUrl": "%s", "extendedData": []},
				{"productId": 3, "name": "Set Booster Pack", "imageUrl": "%s", "extendedData": []},
				{"productId": %d, "name": "Set Booster Box", "imageUrl": "%s", "extendedData": []}
			]}`, img(1), img(2), img(3), group, img(group))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(ls.srv.Close)
	f.app.cfg.TCGCSVURL = ls.srv.URL
	f.app.cfg.LigaDir = filepath.Join(t.TempDir(), "liga")
	return f, site, ls
}

func writeLiga(t *testing.T, f *fixture, ls *listaSite, dir, set string, cards []ligaFixtureCard) {
	t.Helper()
	root := filepath.Join(f.app.cfg.LigaDir, dir)
	images := map[string]string{}
	day := func(date string, captured time.Time, middle bool, last bool) tracking.DaySnapshot {
		snap := tracking.DaySnapshot{Set: set, Date: date, CapturedAt: captured}
		for _, c := range cards {
			price := c.before
			if last {
				price = c.now
			}
			card := tracking.CardDay{Number: c.number, Name: c.name, LowBRL: price}
			for s := 0; s < c.stores; s++ {
				qty := 10
				if s == 0 && middle {
					qty = 10 - c.sold/2
				}
				if s == 0 && last {
					qty = 10 - c.sold
				}
				card.Stores = append(card.Stores, tracking.StoreQty{StoreID: s + 1, StoreName: fmt.Sprintf("Loja %d", s+1), Condition: "2", Language: "2", Quantity: qty, Known: true, PriceBRL: price, PriceKnown: true})
			}
			snap.Cards = append(snap.Cards, card)
		}
		return snap
	}
	store := tracking.NewStore(root, nil)
	for _, d := range []tracking.DaySnapshot{
		day("2026-10-30T06", time.Date(2026, 10, 30, 6, 0, 0, 0, saoPaulo), false, false),
		day("2026-11-03T06", time.Date(2026, 11, 3, 6, 0, 0, 0, saoPaulo), true, false),
		day("2026-11-06T06", time.Date(2026, 11, 6, 6, 0, 0, 0, saoPaulo), false, true),
	} {
		if err := store.SaveDay(d); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range cards {
		if c.image {
			images[set+"/"+c.number] = fmt.Sprintf("%s/carta/%s.jpg", ls.srv.URL, c.number)
		}
	}
	data, err := json.Marshal(images)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "images.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func listaArgs(tipo, jogo string) map[string]any {
	return map[string]any{
		"tipo": tipo, "jogo": jogo, "formato": "video", "etiqueta": "Pokémon · Liga", "gancho": "As cartas que mais subiram",
		"destaque": []string{"subiram"},
		"final":    map[string]any{"titulo": "Siga a Collecta", "oferta": "Envio pra todo o Brasil", "linhas": []string{"collectatcg.com.br"}, "cta": "Link na bio"},
	}
}

func lastLista(t *testing.T, f *fixture) listaProps {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(f.data, "pecas"))
	if err != nil || len(entries) == 0 {
		t.Fatalf("sem props gravadas: %v", err)
	}
	var newest os.DirEntry
	var at time.Time
	for _, e := range entries {
		info, err := e.Info()
		if err == nil && !info.ModTime().Before(at) {
			newest, at = e, info.ModTime()
		}
	}
	data, err := os.ReadFile(filepath.Join(f.data, "pecas", newest.Name()))
	if err != nil {
		t.Fatal(err)
	}
	var props listaProps
	if err := json.Unmarshal(data, &props); err != nil {
		t.Fatal(err)
	}
	return props
}

func TestListaReleasesOfTheMonth(t *testing.T) {
	f, _, ls := newListaFixture(t)
	args := listaArgs("lancamentos", "pokemon")
	args["produto"] = map[string]any{"link": "https://www.collectatcg.com.br/selado/reinado-box", "nome": "Booster Box Reinado Delta"}
	out, errText := callTool[listaOutput](t, f, "criar_lista", args)
	if errText != "" {
		t.Fatal(errText)
	}
	props := lastLista(t, f)
	if props.Tipo != "lancamentos" || props.Selo != "NOVEMBRO" || len(props.Itens) != 2 || !strings.Contains(props.Fonte, "TCGplayer") {
		t.Fatalf("props = %+v", props)
	}
	set, product := props.Itens[0], props.Itens[1]
	if set.Titulo != "ME06: Delta Reign" || set.Rotulo != "06/11" || set.Detalhe != "Já saiu nos EUA" || !strings.HasPrefix(set.Imagem, "agente/") || set.Proporcao <= 0 {
		t.Fatalf("set = %+v", set)
	}
	if product.Produto == nil || product.Produto.Preco != 389.5 || product.Rotulo != "Na Collecta" {
		t.Fatalf("produto = %+v", product)
	}
	if len(out.Precos) != 1 || out.Precos[0].Preco != 389.5 || out.Itens[0].Rotulo != "06/11" || out.Itens[1].Agora != 389.5 {
		t.Fatalf("saída = %+v", out)
	}
	if !contains(ls.requests, "/product/24831_in_1000x1000.jpg") {
		t.Fatalf("não pediu a foto grande da booster box: %v", ls.requests)
	}
	if !strings.Contains(out.Arquivo, "-lista-lancamentos-pokemon-") || !strings.HasSuffix(out.Arquivo, ".mp4") {
		t.Fatalf("arquivo = %s", out.Arquivo)
	}
}

func TestListaReleasesEmptyMonthShowsLastAndNext(t *testing.T) {
	f, _, _ := newListaFixture(t)
	out, errText := callTool[listaOutput](t, f, "criar_lista", listaArgs("lancamentos", "lorcana"))
	if errText != "" {
		t.Fatal(errText)
	}
	props := lastLista(t, f)
	if len(props.Itens) != 2 || props.Itens[0].Detalhe != "Último lançamento" || props.Itens[0].Rotulo != "16/10" || props.Itens[1].Detalhe != "Próximo lançamento" || props.Itens[1].Rotulo != "12/12" {
		t.Fatalf("itens = %+v", props.Itens)
	}
	if len(out.Avisos) != 1 || !strings.Contains(out.Avisos[0], "não tem set novo em novembro") {
		t.Fatalf("avisos = %v", out.Avisos)
	}
	args := listaArgs("lancamentos", "lorcana")
	args["mes"] = "2026-12"
	if _, errText = callTool[listaOutput](t, f, "criar_lista", args); errText != "" {
		t.Fatal(errText)
	}
	if props = lastLista(t, f); props.Selo != "DEZEMBRO" || len(props.Itens) != 1 || props.Itens[0].Titulo != "Winterspell" || props.Itens[0].Detalhe != "Lançamento EUA" {
		t.Fatalf("dezembro = %+v", props)
	}
}

func TestListaPriceRisers(t *testing.T) {
	f, _, ls := newListaFixture(t)
	writeLiga(t, f, ls, "tracking-pkm", "M2", []ligaFixtureCard{
		{number: "001", name: "Charizard ex (#001/080)", before: 20, now: 40, stores: 3, image: true},
		{number: "002", name: "Pikachu (#002/080)", before: 10, now: 12, stores: 4, image: true},
		{number: "003", name: "Barato (#003/080)", before: 5, now: 50, stores: 5, image: true},
		{number: "004", name: "Poucas Lojas (#004/080)", before: 30, now: 60, stores: 2, image: true},
		{number: "005", name: "Umbreon ex (#005/080)", before: 15, now: 21, stores: 3, image: true},
		{number: "006", name: "Sem Foto (#006/080)", before: 20, now: 80, stores: 3},
		{number: "007", name: "Caiu (#007/080)", before: 50, now: 45, stores: 3, image: true},
	})
	out, errText := callTool[listaOutput](t, f, "criar_lista", listaArgs("subiram", "pokemon"))
	if errText != "" {
		t.Fatal(errText)
	}
	props := lastLista(t, f)
	titles := []string{}
	for _, it := range props.Itens {
		titles = append(titles, it.Titulo)
	}
	if strings.Join(titles, ",") != "Charizard ex,Umbreon ex,Pikachu" {
		t.Fatalf("ordem = %v", titles)
	}
	first := props.Itens[0]
	if first.Antes != 20 || first.Agora != 40 || first.Alta != 100 || first.Rotulo != "#1" || first.Detalhe != "M2 · 001" {
		t.Fatalf("primeiro = %+v", first)
	}
	if props.Fonte != "Menor preço na Liga, 30/10 a 06/11" || out.Fonte != props.Fonte || out.Itens[1].Alta != 40 || out.Itens[1].Antes != 15 {
		t.Fatalf("fonte/saída = %q %+v", props.Fonte, out.Itens)
	}
}

func TestListaBestSellers(t *testing.T) {
	f, _, ls := newListaFixture(t)
	writeLiga(t, f, ls, "tracking", "OP-16", []ligaFixtureCard{
		{number: "OP16-056", name: "Mr.3 (OP16-056)", before: 40, now: 40, stores: 3, sold: 4, image: true},
		{number: "OP16-001", name: "Luffy (OP16-001)", before: 12, now: 12, stores: 3, sold: 2, image: true},
		{number: "OP16-002", name: "Comum (OP16-002)", before: 2, now: 2, stores: 3, sold: 6, image: true},
		{number: "OP16-003", name: "Zoro (OP16-003)", before: 20, now: 20, stores: 3, sold: 2, image: true},
		{number: "OP16-004", name: "Parada (OP16-004)", before: 90, now: 90, stores: 3, image: true},
	})
	args := listaArgs("em_alta", "onepiece")
	args["itens"] = 3
	out, errText := callTool[listaOutput](t, f, "criar_lista", args)
	if errText != "" {
		t.Fatal(errText)
	}
	props := lastLista(t, f)
	if len(props.Itens) != 3 || props.Itens[0].Titulo != "Mr.3" || props.Itens[0].Unidades != 4 || props.Itens[0].Agora != 40 || props.Itens[1].Titulo != "Zoro" || props.Itens[2].Titulo != "Luffy" {
		t.Fatalf("itens = %+v", props.Itens)
	}
	if props.Itens[0].Detalhe != "OP16-056" || !strings.HasPrefix(props.Fonte, "Vendas na Liga") || out.Itens[0].Unidades != 4 {
		t.Fatalf("detalhe/fonte = %+v %q", props.Itens[0], props.Fonte)
	}
}

func TestListaRefusesStaleLiga(t *testing.T) {
	f, site, ls := newListaFixture(t)
	writeLiga(t, f, ls, "tracking-rft", "UNL", []ligaFixtureCard{{number: "1", name: "Ahri (1)", before: 20, now: 40, stores: 3, image: true}})
	f.now = time.Date(2026, 11, 9, 10, 0, 0, 0, saoPaulo)
	_, errText := callTool[listaOutput](t, f, "criar_lista", listaArgs("subiram", "riftbound"))
	if !strings.Contains(errText, "parados desde 06/11 06h") || len(site.renders) != 0 {
		t.Fatalf("erro = %q", errText)
	}
}

func TestListaNews(t *testing.T) {
	f, _, ls := newListaFixture(t)
	args := listaArgs("noticias", "pokemon")
	args["manchetes"] = []map[string]any{
		{"texto": "Reinado Delta chega com 5 caixas especiais", "link": ls.srv.URL + "/noticia", "data": "2026-11-05", "set": "DLR"},
		{"texto": "Site bloqueia robô mas a notícia existe", "link": ls.srv.URL + "/bloqueia", "data": "2026-11-06"},
	}
	out, errText := callTool[listaOutput](t, f, "criar_lista", args)
	if errText != "" {
		t.Fatal(errText)
	}
	props := lastLista(t, f)
	if len(props.Itens) != 2 || props.Itens[0].Rotulo != "05/11" || props.Itens[0].Detalhe != "fonte: 127.0.0.1" || props.Selo != "NOTÍCIAS" {
		t.Fatalf("itens = %+v", props.Itens)
	}
	if out.Itens[1].Link != ls.srv.URL+"/bloqueia" {
		t.Fatalf("saída = %+v", out.Itens)
	}
}

func TestListaRefuses(t *testing.T) {
	cases := []struct {
		name string
		args func(ls *listaSite) map[string]any
		want string
	}{
		{"tipo", func(*listaSite) map[string]any { return listaArgs("ranking", "pokemon") }, "não existe"},
		{"jogo", func(*listaSite) map[string]any { return listaArgs("subiram", "digimon") }, "jogo"},
		{"magic sem liga", func(*listaSite) map[string]any { return listaArgs("subiram", "magic") }, "a Liga não tem dados de Magic"},
		{"itens demais", func(*listaSite) map[string]any {
			a := listaArgs("subiram", "pokemon")
			a["itens"] = 8
			return a
		}, "itens vai de 3 a 5"},
		{"manchete fora de noticias", func(ls *listaSite) map[string]any {
			a := listaArgs("subiram", "pokemon")
			a["manchetes"] = []map[string]any{{"texto": "Uma manchete qualquer", "link": ls.srv.URL + "/noticia", "data": "2026-11-05"}}
			return a
		}, "manchetes só valem"},
		{"sem manchete", func(*listaSite) map[string]any { return listaArgs("noticias", "pokemon") }, "de 1 a 3 manchetes"},
		{"manchete velha", func(ls *listaSite) map[string]any {
			a := listaArgs("noticias", "pokemon")
			a["manchetes"] = []map[string]any{{"texto": "Notícia de outubro", "link": ls.srv.URL + "/noticia", "data": "2026-10-20"}}
			return a
		}, "mais de 7 dias"},
		{"manchete com travessão", func(ls *listaSite) map[string]any {
			a := listaArgs("noticias", "pokemon")
			a["manchetes"] = []map[string]any{{"texto": "Reinado Delta — chegou", "link": ls.srv.URL + "/noticia", "data": "2026-11-05"}}
			return a
		}, "travessão"},
		{"link quebrado", func(ls *listaSite) map[string]any {
			a := listaArgs("noticias", "pokemon")
			a["manchetes"] = []map[string]any{{"texto": "Notícia que sumiu do site", "link": ls.srv.URL + "/sumiu", "data": "2026-11-05"}}
			return a
		}, "respondeu 404"},
		{"produto fora de lancamentos", func(*listaSite) map[string]any {
			a := listaArgs("em_alta", "pokemon")
			a["produto"] = map[string]any{"link": "https://www.collectatcg.com.br/selado/reinado-box"}
			return a
		}, "só valem no tipo lancamentos"},
		{"gancho longo", func(*listaSite) map[string]any {
			a := listaArgs("lancamentos", "pokemon")
			a["gancho"] = strings.Repeat("a", 41)
			return a
		}, "gancho com 41 letras"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, site, ls := newListaFixture(t)
			_, errText := callTool[listaOutput](t, f, "criar_lista", c.args(ls))
			if !strings.Contains(errText, c.want) {
				t.Fatalf("erro = %q, queria %q", errText, c.want)
			}
			if len(site.renders) != 0 {
				t.Fatal("renderizou mesmo recusando")
			}
		})
	}
}

func TestListaSharesDailyLimit(t *testing.T) {
	f, _, _ := newListaFixture(t)
	if err := os.MkdirAll(f.data, 0o755); err != nil {
		t.Fatal(err)
	}
	var lines strings.Builder
	for i := 0; i < rendersPerDay; i++ {
		lines.WriteString("2026-11-06 08:00 gerada x produto video /tmp/x.mp4\n")
	}
	if err := os.WriteFile(filepath.Join(f.data, "pecas.log"), []byte(lines.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, errText := callTool[listaOutput](t, f, "criar_lista", listaArgs("lancamentos", "pokemon")); !strings.Contains(errText, "limite do dia") {
		t.Fatalf("erro = %q", errText)
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func TestReleaseLabelShowsYearOutsideThisYear(t *testing.T) {
	today := time.Date(2026, 11, 7, 10, 0, 0, 0, saoPaulo)
	if got := releaseLabel(time.Date(2026, 11, 20, 0, 0, 0, 0, saoPaulo), today); got != "20/11" {
		t.Fatalf("mesmo ano = %q", got)
	}
	if got := releaseLabel(time.Date(2027, 2, 5, 0, 0, 0, 0, saoPaulo), today); got != "05/02/27" {
		t.Fatalf("outro ano = %q", got)
	}
}

func TestCardName(t *testing.T) {
	cases := map[string]string{
		"Mr.3(Galdino) (056)":          "Mr.3(Galdino)",
		"Mega Heracross ex (#004/080)": "Mega Heracross ex",
		"Smoker (SP) (OP10-030)":       "Smoker (SP)",
		"Pok&eacute; Pad (#198/217)":   "Poké Pad",
		"Luffy & Ace (ST30-001)":       "Luffy & Ace",
	}
	for raw, want := range cases {
		if got := cardName(raw); got != want {
			t.Errorf("cardName(%q) = %q, queria %q", raw, got, want)
		}
	}
}
