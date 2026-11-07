package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"opdeals/internal/adtestq"
)

func (f *fixture) serveAds(w http.ResponseWriter, r *http.Request) {
	writeErr := func(err error) {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, adtestq.ErrOverBudget), errors.Is(err, adtestq.ErrWrongStatus):
			status = http.StatusConflict
		case errors.Is(err, adtestq.ErrInvalidInput):
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/adtests/"), "/")
	id, action := parts[0], ""
	if len(parts) > 1 {
		action = parts[1]
	}
	var out adtestq.Test
	var err error
	switch {
	case r.URL.Path == "/api/adtests" && r.Method == http.MethodPost:
		var in adtestq.Test
		_ = json.NewDecoder(r.Body).Decode(&in)
		out, err = f.ads.Add(in)
	case r.URL.Path == "/api/adtests":
		all, lerr := f.ads.List(r.URL.Query().Get("status"))
		avail, aerr := f.ads.Available()
		if lerr != nil || aerr != nil {
			writeErr(errors.Join(lerr, aerr))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"items": all, "availableCents": avail})
		return
	case action == "started":
		var in struct {
			Meta   adtestq.Meta `json:"meta"`
			EndsAt time.Time    `json:"endsAt"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		out, err = f.ads.Started(id, in.Meta, in.EndsAt)
	case action == "meta":
		var in adtestq.Meta
		_ = json.NewDecoder(r.Body).Decode(&in)
		out, err = f.ads.SetMeta(id, in)
	case action == "finished":
		var in struct {
			Summary string `json:"summary"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		out, err = f.ads.Finished(id, in.Summary)
	case action == "failed":
		var in struct {
			Error string        `json:"error"`
			Meta  *adtestq.Meta `json:"meta"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		out, err = f.ads.Failed(id, in.Error, in.Meta)
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		writeErr(err)
		return
	}
	_ = json.NewEncoder(w).Encode(out)
}

type metaCall struct {
	method string
	path   string
	form   map[string]string
}

type fakeMeta struct {
	mu       sync.Mutex
	calls    []metaCall
	next     int
	failOn   string
	insights []insightRow
}

func (m *fakeMeta) serve(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v27.0/")
	form := map[string]string{}
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		_ = r.ParseMultipartForm(32 << 20)
	} else {
		_ = r.ParseForm()
	}
	for k, v := range r.Form {
		form[k] = v[0]
	}
	m.mu.Lock()
	m.calls = append(m.calls, metaCall{method: r.Method, path: path, form: form})
	m.next++
	n := m.next
	fail := m.failOn != "" && strings.HasSuffix(path, m.failOn)
	insights := m.insights
	m.mu.Unlock()
	if form["access_token"] != "token-meta" {
		http.Error(w, `{"error":{"message":"token"}}`, http.StatusUnauthorized)
		return
	}
	if fail {
		http.Error(w, `{"error":{"message":"Invalid parameter","error_user_msg":"Objetivo indisponível","code":100,"error_subcode":1815629}}`, http.StatusBadRequest)
		return
	}
	switch {
	case strings.HasSuffix(path, "/adimages"):
		_ = json.NewEncoder(w).Encode(map[string]any{"images": map[string]any{"x.png": map[string]string{"hash": fmt.Sprintf("hash%d", n)}}})
	case strings.HasSuffix(path, "/insights"):
		_ = json.NewEncoder(w).Encode(map[string]any{"data": insights})
	case path == "search":
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"id": "6003", "name": r.URL.Query().Get("q")}}})
	case r.Method == http.MethodGet && r.URL.Query().Get("fields") == "status":
		_ = json.NewEncoder(w).Encode(map[string]any{"status": map[string]string{"video_status": "ready"}})
	case r.Method == http.MethodPost && !strings.Contains(path, "/"):
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	default:
		_ = json.NewEncoder(w).Encode(map[string]string{"id": fmt.Sprintf("id%d", n)})
	}
}

func (m *fakeMeta) find(suffix string) []metaCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []metaCall
	for _, c := range m.calls {
		if strings.HasSuffix(c.path, suffix) {
			out = append(out, c)
		}
	}
	return out
}

func (m *fakeMeta) statusPosts(status string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, c := range m.calls {
		if c.method == http.MethodPost && !strings.Contains(c.path, "/") && c.form["status"] == status {
			n++
		}
	}
	return n
}

func newAdFixture(t *testing.T) (*fixture, *fakeMeta, *executor) {
	t.Helper()
	f := newFixture(t)
	meta := &fakeMeta{}
	srv := httptest.NewServer(http.HandlerFunc(meta.serve))
	t.Cleanup(srv.Close)
	mc := metaConfig{Token: "token-meta", Account: "act_1", PageID: "page", IGUserID: "ig", IGUsername: "tcgcollecta", PixelID: "px", BaseURL: srv.URL + "/v27.0"}
	ex := newExecutor(f.app, mc, http.DefaultClient)
	ex.pollEvery = time.Millisecond
	return f, meta, ex
}

func (f *fixture) creatives(t *testing.T) []map[string]any {
	t.Helper()
	video := f.file(t, "2026-11-06-produto-x-1.mp4", "video um")
	f.file(t, "2026-11-06-produto-x-1-previa-1.png", "capa um")
	video2 := f.file(t, "2026-11-06-contagem-y-2.mp4", "video dois")
	f.file(t, "2026-11-06-contagem-y-2-previa-1.png", "capa dois")
	img := f.file(t, "arte.png", "imagem")
	return []map[string]any{
		{"arquivo": video, "texto": "Reinado Delta na loja, R$ 389,50 no Pix", "gancho": "preço no Pix"},
		{"arquivo": video2, "texto": "Faltam 9 dias pra inauguração", "gancho": "contagem"},
		{"arquivo": img, "texto": "A gente compra coleção", "titulo": "Collecta TCG", "gancho": "compra de coleção"},
	}
}

func proposeArgs(cs []map[string]any, budget int) map[string]any {
	return map[string]any{"objetivo": "seguidores", "publico": "goiania", "criativos": cs, "verba_reais": budget, "dias": 4, "motivo": "achar o gancho que mais traz seguidor"}
}

func approveAds(t *testing.T, f *fixture) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(filepath.Dir(f.data), "adtests.json"))
	if err != nil {
		t.Fatal(err)
	}
	var all []adtestq.Test
	if err := json.Unmarshal(data, &all); err != nil {
		t.Fatal(err)
	}
	for _, x := range all {
		if x.Status == adtestq.StatusPending {
			if _, err := f.ads.Decide(x.ID, x.Token, true); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestProposeAdTestAsksOwner(t *testing.T) {
	f, _, _ := newAdFixture(t)
	out, errText := callTool[proposeOutput](t, f, "propor_teste", proposeArgs(f.creatives(t), 100))
	if errText != "" {
		t.Fatal(errText)
	}
	if out.ID == "" || out.SobraSemana != "R$ 50,00" {
		t.Fatalf("saída = %+v", out)
	}
	pushes := f.pushed()
	if len(pushes) != 1 || pushes[0].title != "Aprovar teste de anúncio" || !strings.Contains(pushes[0].actions, "/api/aprovar-anuncio/"+out.ID) {
		t.Fatalf("push = %+v", pushes)
	}
	if pushes[0].filename != "01-capa.png" || !strings.Contains(pushes[0].message, "R$ 100,00 em 4 dias") {
		t.Fatalf("anexo ou texto = %+v", pushes[0])
	}
	list, _ := f.ads.List("")
	if len(list) != 1 || list[0].Creatives[0].Kind != "video" || list[0].Creatives[2].Kind != "imagem" || !strings.HasPrefix(list[0].Creatives[0].Path, filepath.Join(f.data, "anuncios")) {
		t.Fatalf("fila = %+v", list)
	}
	if _, err := os.Stat(list[0].Creatives[0].Thumb); err != nil {
		t.Fatal(err)
	}
	if _, errText = callTool[proposeOutput](t, f, "propor_teste", proposeArgs(f.creatives(t), 60)); !strings.Contains(errText, "cabem só R$ 50,00") {
		t.Fatalf("teto: %q", errText)
	}
}

func TestProposeAdTestRefuses(t *testing.T) {
	cases := []struct {
		name   string
		change func(*fixture, map[string]any)
		want   string
	}{
		{"travessão", func(_ *fixture, a map[string]any) {
			a["criativos"].([]map[string]any)[0]["texto"] = "Chegou — corre"
		}, "travessão"},
		{"vídeo sem prévia", func(f *fixture, a map[string]any) {
			a["criativos"].([]map[string]any)[0]["arquivo"] = f.file(t, "solto.mp4", "x")
		}, "criar_peca"},
		{"um criativo só", func(_ *fixture, a map[string]any) {
			a["criativos"] = a["criativos"].([]map[string]any)[:1]
		}, "de 2 a 5"},
		{"vendas sem link", func(_ *fixture, a map[string]any) { a["objetivo"] = "vendas" }, "precisa ser uma página"},
		{"vendas fora do site", func(_ *fixture, a map[string]any) {
			a["objetivo"] = "vendas"
			a["link"] = "https://www.mercadolivre.com.br/x"
		}, "precisa ser uma página"},
		{"verba alta", func(_ *fixture, a map[string]any) { a["verba_reais"] = 200 }, "verba_reais vai"},
		{"dias demais", func(_ *fixture, a map[string]any) { a["dias"] = 10 }, "dias vai"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, _, _ := newAdFixture(t)
			args := proposeArgs(f.creatives(t), 50)
			c.change(f, args)
			if _, errText := callTool[proposeOutput](t, f, "propor_teste", args); !strings.Contains(errText, c.want) {
				t.Fatalf("erro = %q, queria %q", errText, c.want)
			}
			if list, _ := f.ads.List(""); len(list) != 0 {
				t.Fatal("criou o teste mesmo recusando")
			}
		})
	}
}

func TestExecutorLaunchesOnlyApprovedTests(t *testing.T) {
	f, meta, ex := newAdFixture(t)
	if _, errText := callTool[proposeOutput](t, f, "propor_teste", proposeArgs(f.creatives(t), 100)); errText != "" {
		t.Fatal(errText)
	}
	if err := ex.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(meta.find("/campaigns")) != 0 {
		t.Fatal("subiu sem aprovação")
	}
	approveAds(t, f)
	if err := ex.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	campaigns := meta.find("act_1/campaigns")
	if len(campaigns) != 1 || campaigns[0].form["objective"] != "OUTCOME_TRAFFIC" || campaigns[0].form["status"] != "PAUSED" || campaigns[0].form["special_ad_categories"] != "[]" {
		t.Fatalf("campanha = %+v", campaigns)
	}
	adsets := meta.find("act_1/adsets")
	if len(adsets) != 1 {
		t.Fatalf("conjuntos = %d", len(adsets))
	}
	set := adsets[0].form
	if set["lifetime_budget"] != "10000" || set["destination_type"] != "INSTAGRAM_PROFILE" || set["optimization_goal"] != "PROFILE_VISIT" || !strings.Contains(set["targeting"], `"radius":30`) || !strings.Contains(set["targeting"], `"advantage_audience":0`) {
		t.Fatalf("conjunto = %+v", set)
	}
	if len(meta.find("act_1/advideos")) != 2 || len(meta.find("act_1/adimages")) != 3 || len(meta.find("act_1/ads")) != 3 {
		t.Fatal("criativos não subiram todos")
	}
	creatives := meta.find("act_1/adcreatives")
	if !strings.Contains(creatives[0].form["object_story_spec"], `"instagram_user_id":"ig"`) || !strings.Contains(creatives[0].form["object_story_spec"], "VIEW_INSTAGRAM_PROFILE") || !strings.Contains(creatives[2].form["object_story_spec"], `"link_data"`) {
		t.Fatalf("criativos = %+v", creatives)
	}
	if meta.statusPosts("ACTIVE") != 5 {
		t.Fatalf("ativou %d objetos, queria 3 anúncios + conjunto + campanha", meta.statusPosts("ACTIVE"))
	}
	running, _ := f.ads.List(adtestq.StatusRunning)
	if len(running) != 1 || len(running[0].Meta.Ads) != 3 || running[0].EndsAt.Sub(running[0].StartedAt) < 96*time.Hour {
		t.Fatalf("no ar = %+v", running)
	}
	if last := f.pushed()[len(f.pushed())-1]; last.title != "Teste de anúncio no ar" {
		t.Fatalf("aviso = %+v", last)
	}
}

func TestExecutorPausesLosersKeepsTwoAndFinishes(t *testing.T) {
	f, meta, ex := newAdFixture(t)
	if _, errText := callTool[proposeOutput](t, f, "propor_teste", proposeArgs(f.creatives(t), 100)); errText != "" {
		t.Fatal(errText)
	}
	approveAds(t, f)
	if err := ex.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	running, _ := f.ads.List(adtestq.StatusRunning)
	ads := running[0].Meta.Ads
	meta.insights = []insightRow{
		{AdID: ads[0].ID, Spend: "20.00", Clicks: "20"},
		{AdID: ads[1].ID, Spend: "20.00", Clicks: "5"},
		{AdID: ads[2].ID, Spend: "16.00", Clicks: "0"},
	}
	if err := ex.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	running, _ = f.ads.List(adtestq.StatusRunning)
	got := running[0].Meta.Ads
	if got[0].Paused || got[1].Paused || !got[2].Paused || !strings.Contains(got[2].Reason, "nenhuma") {
		t.Fatalf("pausas = %+v", got)
	}
	if meta.statusPosts("PAUSED") != 1 {
		t.Fatalf("pausou %d no Meta", meta.statusPosts("PAUSED"))
	}
	f.now = running[0].EndsAt.Add(2 * time.Hour)
	if err := ex.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	done, _ := f.ads.List(adtestq.StatusDone)
	if len(done) != 1 || !strings.Contains(done[0].Summary, "Vencedor: 1. preço no Pix") || !strings.Contains(done[0].Summary, "R$ 56.00 gastos") {
		t.Fatalf("resumo = %+v", done)
	}
}

func TestExecutorFailureStopsCampaign(t *testing.T) {
	f, meta, ex := newAdFixture(t)
	meta.failOn = "act_1/adsets"
	if _, errText := callTool[proposeOutput](t, f, "propor_teste", proposeArgs(f.creatives(t), 100)); errText != "" {
		t.Fatal(errText)
	}
	approveAds(t, f)
	if err := ex.run(context.Background()); err == nil || !strings.Contains(err.Error(), "Objetivo indisponível") {
		t.Fatalf("erro = %v", err)
	}
	failed, _ := f.ads.List(adtestq.StatusFailed)
	if len(failed) != 1 || failed[0].Meta == nil || failed[0].Meta.CampaignID == "" || meta.statusPosts("PAUSED") != 1 || meta.statusPosts("ACTIVE") != 0 {
		t.Fatalf("falha = %+v pausadas=%d", failed, meta.statusPosts("PAUSED"))
	}
	if last := f.pushed()[len(f.pushed())-1]; last.title != "Teste de anúncio não subiu" || last.priority != "high" {
		t.Fatalf("aviso = %+v", last)
	}
}

func TestExecutorRefusesFileChangedAfterApproval(t *testing.T) {
	f, meta, ex := newAdFixture(t)
	if _, errText := callTool[proposeOutput](t, f, "propor_teste", proposeArgs(f.creatives(t), 100)); errText != "" {
		t.Fatal(errText)
	}
	approveAds(t, f)
	list, _ := f.ads.List(adtestq.StatusApproved)
	if err := os.WriteFile(list[0].Creatives[1].Path, []byte("trocado"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ex.run(context.Background()); err == nil || !strings.Contains(err.Error(), "mudou depois da aprovação") {
		t.Fatalf("erro = %v", err)
	}
	if len(meta.find("/campaigns")) != 0 {
		t.Fatal("criou campanha com arquivo trocado")
	}
}

func TestBrazilAudienceUsesInterests(t *testing.T) {
	_, _, ex := newAdFixture(t)
	tg, err := ex.targeting(context.Background(), "brasil")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(tg)
	if !strings.Contains(string(data), `"countries":["BR"]`) || !strings.Contains(string(data), `"name":"Pokémon"`) {
		t.Fatalf("público = %s", data)
	}
}

func TestReadEnvFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "anuncios.env")
	if err := os.WriteFile(p, []byte("# comentário\nMETA_TOKEN=\"abc\"\nMETA_CONTA=123\n\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env, err := readEnvFile(p)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := loadMetaConfig(func(k string) string { return env[k] })
	if err == nil || !strings.Contains(err.Error(), "META_") {
		t.Fatalf("faltando página/ig devia falhar: %v", err)
	}
	env["META_PAGINA"], env["META_INSTAGRAM_ID"] = "p", "i"
	cfg, err = loadMetaConfig(func(k string) string { return env[k] })
	if err != nil || cfg.Token != "abc" || cfg.Account != "act_123" || !strings.HasSuffix(cfg.BaseURL, "/v27.0") {
		t.Fatalf("cfg = %+v %v", cfg, err)
	}
}
