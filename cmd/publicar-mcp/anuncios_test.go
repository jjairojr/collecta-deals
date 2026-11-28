package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"opdeals/internal/adtestq"
	"opdeals/internal/publishq"
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
		msgs, merr := f.ads.AvailableFor("mensagens")
		if lerr != nil || aerr != nil || merr != nil {
			writeErr(errors.Join(lerr, aerr, merr))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"items": all, "availableCents": avail, "availableMessagesCents": msgs})
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
			Summary    string `json:"summary"`
			SpentCents *int64 `json:"spentCents"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		out, err = f.ads.FinishedSpent(id, in.Summary, in.SpentCents)
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
	inFeed   bool
	// media are the Instagram posts the fake account has, by id; failMedia makes
	// the creative of that post fail like Meta refusing licensed music.
	media     map[string]map[string]any
	failMedia string
	// objects são as campanhas, conjuntos e anúncios criados, para a busca por
	// nome da subida idempotente; o status acompanha os POST de status.
	objects []*fakeObject
}

type fakeObject struct {
	id, kind, name, parent, status, end string
}

func (m *fakeMeta) object(id string) *fakeObject {
	for _, o := range m.objects {
		if o.id == id {
			return o
		}
	}
	return nil
}

// listing é a resposta das leituras de lista (campanhas por nome, conjuntos da
// campanha, anúncios do conjunto), ou nil se o caminho não é uma delas.
func (m *fakeMeta) listing(path string, q url.Values) []map[string]string {
	parent, edge, ok := strings.Cut(path, "/")
	if !ok || (edge != "campaigns" && edge != "adsets" && edge != "ads") {
		return nil
	}
	kind := map[string]string{"campaigns": "campaign", "adsets": "adset", "ads": "ad"}[edge]
	want := ""
	if f := q.Get("filtering"); f != "" {
		var rules []map[string]string
		_ = json.Unmarshal([]byte(f), &rules)
		if len(rules) > 0 {
			want = rules[0]["value"]
		}
	}
	out := []map[string]string{}
	for _, o := range m.objects {
		if o.kind != kind || (kind != "campaign" && o.parent != parent) || (want != "" && !strings.Contains(o.name, want)) {
			continue
		}
		out = append(out, map[string]string{"id": o.id, "name": o.name, "effective_status": o.status, "end_time": o.end})
	}
	return out
}

func (m *fakeMeta) serve(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/cdn/") {
		_, _ = w.Write([]byte("jpg " + r.URL.Path))
		return
	}
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
	var listed []map[string]string
	if r.Method == http.MethodGet {
		listed = m.listing(path, r.URL.Query())
	}
	if r.Method == http.MethodPost {
		if o := m.object(path); o != nil && form["status"] != "" {
			o.status = form["status"]
		}
		created := fmt.Sprintf("id%d", n)
		switch {
		case path == "act_1/campaigns":
			m.objects = append(m.objects, &fakeObject{id: created, kind: "campaign", name: form["name"], status: form["status"]})
		case path == "act_1/adsets":
			end := ""
			if t, err := time.Parse(time.RFC3339, form["end_time"]); err == nil {
				end = t.Format("2006-01-02T15:04:05-0700")
			}
			m.objects = append(m.objects, &fakeObject{id: created, kind: "adset", name: form["name"], parent: form["campaign_id"], status: form["status"], end: end})
		case path == "act_1/ads":
			m.objects = append(m.objects, &fakeObject{id: created, kind: "ad", name: form["name"], parent: form["adset_id"], status: form["status"]})
		}
	}
	media := m.media[path]
	var list []map[string]any
	if r.Method == http.MethodGet && strings.HasSuffix(path, "/media") {
		for id, x := range m.media {
			list = append(list, map[string]any{"id": id, "permalink": x["permalink"]})
		}
	}
	refuse := m.failMedia != "" && form["source_instagram_media_id"] == m.failMedia
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
	case listed != nil:
		_ = json.NewEncoder(w).Encode(map[string]any{"data": listed})
	case refuse:
		http.Error(w, `{"error":{"message":"Invalid parameter","error_user_msg":"A música deste post tem licença e não pode virar anúncio","code":100,"error_subcode":2490427}}`, http.StatusBadRequest)
	case r.Method == http.MethodGet && strings.HasSuffix(path, "/media"):
		_ = json.NewEncoder(w).Encode(map[string]any{"data": list})
	case r.Method == http.MethodGet && media != nil:
		_ = json.NewEncoder(w).Encode(media)
	case strings.HasSuffix(path, "/adimages"):
		_ = json.NewEncoder(w).Encode(map[string]any{"images": map[string]any{"x.png": map[string]string{"hash": fmt.Sprintf("hash%d", n)}}})
	case strings.HasSuffix(path, "/insights"):
		_ = json.NewEncoder(w).Encode(map[string]any{"data": insights})
	case path == "search":
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"id": "6003", "name": r.URL.Query().Get("q")}}})
	case r.Method == http.MethodGet && r.URL.Query().Get("fields") == "status_code,status":
		_ = json.NewEncoder(w).Encode(map[string]string{"status_code": "FINISHED", "status": "Finished: Media has been uploaded"})
	case r.Method == http.MethodGet && r.URL.Query().Get("fields") == "permalink,is_shared_to_feed":
		_ = json.NewEncoder(w).Encode(map[string]any{"permalink": "https://www.instagram.com/reel/teste123/", "is_shared_to_feed": m.inFeed})
	case r.Method == http.MethodGet && r.URL.Query().Get("fields") == "status":
		_ = json.NewEncoder(w).Encode(map[string]any{"status": map[string]string{"video_status": "ready"}})
	case r.Method == http.MethodPost && !strings.Contains(path, "/"):
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	default:
		_ = json.NewEncoder(w).Encode(map[string]string{"id": fmt.Sprintf("id%d", n)})
	}
}

// find devolve as chamadas que terminam em suffix, menos as leituras de lista
// (GET de campanhas, conjuntos ou anúncios), que não criam nada.
func (m *fakeMeta) find(suffix string) []metaCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []metaCall
	for _, c := range m.calls {
		if c.method == http.MethodGet && (strings.HasSuffix(c.path, "/campaigns") || strings.HasSuffix(c.path, "/adsets") || strings.HasSuffix(c.path, "/ads")) {
			continue
		}
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

type fakeHost struct {
	shared []string
	open   int
}

func (h *fakeHost) share(_ context.Context, path string) (string, func(), error) {
	h.shared = append(h.shared, path)
	h.open++
	return "https://tunel.test/v.mp4", func() { h.open-- }, nil
}

func TestExecutorWarnsWhenTrialLandsInFeed(t *testing.T) {
	f, meta, ex := newAdFixture(t)
	f.metaEnv(t)
	meta.inFeed = true
	video := f.file(t, "lista.mp4", "v")
	if _, errText := callTool[requestOutput](t, f, "pedir_aprovacao", trialArgs(video)); errText != "" {
		t.Fatal(errText)
	}
	f.now = f.now.Add(6 * time.Hour)
	if err := ex.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range f.pushed() {
		if p.title == "Reels de teste caiu na grade" {
			found = true
		}
	}
	if !found {
		t.Fatalf("devia avisar que caiu na grade: %+v", f.pushed())
	}
}

func newAdFixture(t *testing.T) (*fixture, *fakeMeta, *executor) {
	t.Helper()
	f := newFixture(t)
	meta := &fakeMeta{}
	srv := httptest.NewServer(http.HandlerFunc(meta.serve))
	t.Cleanup(srv.Close)
	mc := metaConfig{Token: "token-meta", Account: "act_1", PageID: "page", IGUserID: "ig", IGUsername: "tcgcollecta", PixelID: "px", BaseURL: srv.URL + "/v27.0", Trials: true}
	ex := newExecutor(f.app, mc, http.DefaultClient)
	ex.pollEvery = time.Millisecond
	ex.host = &fakeHost{}
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
	out, errText := callTool[proposeOutput](t, f, "propor_teste", proposeArgs(f.creatives(t), 300))
	if errText != "" {
		t.Fatal(errText)
	}
	if out.ID == "" || out.SobraSemana != "R$ 650,00" {
		t.Fatalf("saída = %+v", out)
	}
	pushes := f.pushed()
	if len(pushes) != 1 || pushes[0].title != "Aprovar teste de anúncio" || !strings.Contains(pushes[0].actions, "/api/aprovar-anuncio/"+out.ID) {
		t.Fatalf("push = %+v", pushes)
	}
	if pushes[0].filename != "01-capa.png" || !strings.Contains(pushes[0].message, "R$ 300,00 em 4 dias") {
		t.Fatalf("anexo ou texto = %+v", pushes[0])
	}
	list, _ := f.ads.List("")
	if len(list) != 1 || list[0].Creatives[0].Kind != "video" || list[0].Creatives[2].Kind != "imagem" || !strings.HasPrefix(list[0].Creatives[0].Path, filepath.Join(f.data, "anuncios")) {
		t.Fatalf("fila = %+v", list)
	}
	if _, err := os.Stat(list[0].Creatives[0].Thumb); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ { // 3 testes de R$ 300 ao mesmo tempo
		if _, errText = callTool[proposeOutput](t, f, "propor_teste", proposeArgs(f.creatives(t), 300)); errText != "" {
			t.Fatal(errText)
		}
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
		{"verba alta", func(_ *fixture, a map[string]any) { a["verba_reais"] = 400 }, "verba_reais vai"},
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
	// O gasto real vai junto, e é ele que conta no teto daqui em diante.
	if done[0].SpentCents == nil || *done[0].SpentCents != 5600 || adtestq.Held(done[0]) != 5600 {
		t.Fatalf("gasto gravado = %v", done[0].SpentCents)
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

func TestRemarketingAudience(t *testing.T) {
	_, _, ex := newAdFixture(t)
	if _, err := ex.targeting(context.Background(), "remarketing"); err == nil || !strings.Contains(err.Error(), "META_PUBLICO_REMARKETING") {
		t.Fatalf("sem públicos devia falhar: %v", err)
	}
	ex.g.cfg.Remarketing = []string{"111", "222"}
	ex.g.cfg.Buyers = []string{"999"}
	tg, err := ex.targeting(context.Background(), "remarketing")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(tg)
	s := string(data)
	for _, want := range []string{
		`"custom_audiences":[{"id":"111"},{"id":"222"}]`, `"excluded_custom_audiences":[{"id":"999"}]`,
		`"countries":["BR"]`, `"age_min":18`, `"age_max":65`, `"advantage_audience":0`,
		`"publisher_platforms":["facebook","instagram"]`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("falta %s em %s", want, s)
		}
	}
	if strings.Contains(s, "flexible_spec") {
		t.Errorf("remarketing não leva interesse: %s", s)
	}
}

func TestIDList(t *testing.T) {
	if got := idList(" 1, ,2 ,"); len(got) != 2 || got[0] != "1" || got[1] != "2" {
		t.Fatalf("idList = %v", got)
	}
	if got := idList(""); len(got) != 0 {
		t.Fatalf("idList vazio = %v", got)
	}
}

func TestAudiencesAgeAndPlacements(t *testing.T) {
	_, _, ex := newAdFixture(t)
	for _, audience := range []string{"brasil", "goiania"} { // remarketing tem teste próprio
		tg, err := ex.targeting(context.Background(), audience)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(tg)
		s := string(data)
		for _, want := range []string{
			`"age_min":18`, `"age_max":65`,
			`"publisher_platforms":["facebook","instagram"]`,
			`"facebook_positions":["feed","story","facebook_reels"]`,
			`"instagram_positions":["stream","story","reels"]`,
		} {
			if !strings.Contains(s, want) {
				t.Errorf("%s: falta %s em %s", audience, want, s)
			}
		}
		if strings.Contains(s, "audience_network") || strings.Contains(s, "messenger") {
			t.Errorf("%s: posicionamento a mais em %s", audience, s)
		}
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

func (f *fixture) metaEnv(t *testing.T) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "anuncios.env")
	if err := os.WriteFile(p, []byte("META_TOKEN=x\nMETA_INSTAGRAM_ID=ig\nMETA_REELS_TESTE=on\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.app.cfg.AdsEnv = p
}

func TestRequestTrialReelNeedsSwitch(t *testing.T) {
	f := newFixture(t)
	video := f.file(t, "lista.mp4", "v")
	off := filepath.Join(t.TempDir(), "anuncios.env")
	if err := os.WriteFile(off, []byte("META_TOKEN=x\nMETA_INSTAGRAM_ID=ig\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, env := range []string{filepath.Join(t.TempDir(), "nao-existe.env"), off} {
		f.app.cfg.AdsEnv = env
		if _, errText := callTool[requestOutput](t, f, "pedir_aprovacao", trialArgs(video)); !strings.Contains(errText, "desligado") {
			t.Fatalf("devia recusar com %s: %s", env, errText)
		}
	}
	if all, _ := f.store.List(""); len(all) != 0 {
		t.Fatalf("não devia criar pedido: %+v", all)
	}
}

func TestExecutorSkipsTrialsWhenOff(t *testing.T) {
	f, meta, ex := newAdFixture(t)
	f.metaEnv(t)
	video := f.file(t, "lista.mp4", "v")
	if _, errText := callTool[requestOutput](t, f, "pedir_aprovacao", trialArgs(video)); errText != "" {
		t.Fatal(errText)
	}
	ex.g.cfg.Trials = false
	f.now = f.now.Add(6 * time.Hour)
	if err := ex.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(meta.find("ig/media")) != 0 {
		t.Fatal("publicou reels de teste com a chave desligada")
	}
}

func trialArgs(path string) map[string]any {
	args := reelsArgs(path)
	args["formato"] = "teste"
	return args
}

func TestRequestTrialReelApprovesItselfAndSkipsPublicador(t *testing.T) {
	f := newFixture(t)
	f.metaEnv(t)
	video := f.file(t, "lista.mp4", "v")
	out, errText := callTool[requestOutput](t, f, "pedir_aprovacao", trialArgs(video))
	if errText != "" || !strings.Contains(out.Proximo, "API do Instagram") {
		t.Fatalf("saída errada: %+v %s", out, errText)
	}
	approved, _ := f.store.List(publishq.StatusApproved)
	if len(approved) != 1 || approved[0].Format != "teste" {
		t.Fatalf("reels de teste devia ficar aprovado: %+v", approved)
	}
	p := f.pushed()
	if len(p) != 1 || p[0].title != "Reels de teste agendado pela Collecta" || p[0].actions != "" {
		t.Fatalf("aviso errado: %+v", p)
	}
	f.now = f.now.Add(6 * time.Hour)
	ready, errText := callTool[approvedOutput](t, f, "aprovados", map[string]any{})
	if errText != "" || len(ready.Itens) != 0 || ready.Agendados != 0 {
		t.Fatalf("Publicador não devia ver o reels de teste: %+v %s", ready, errText)
	}
}

func TestExecutorPublishesTrialReel(t *testing.T) {
	f, meta, ex := newAdFixture(t)
	f.metaEnv(t)
	video := f.file(t, "lista.mp4", "video da lista")
	out, errText := callTool[requestOutput](t, f, "pedir_aprovacao", trialArgs(video))
	if errText != "" {
		t.Fatal(errText)
	}
	if err := ex.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(meta.find("ig/media")) != 0 {
		t.Fatal("publicou antes do horário")
	}
	f.now = f.now.Add(5*time.Hour + time.Minute)
	if err := ex.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	created := meta.find("ig/media")
	host := ex.host.(*fakeHost)
	if len(created) != 1 || created[0].form["media_type"] != "REELS" || created[0].form["video_url"] != "https://tunel.test/v.mp4" || created[0].form["trial_params"] != `{"graduation_strategy":"SS_PERFORMANCE"}` || !strings.Contains(created[0].form["caption"], "Faltam 10 dias") {
		t.Fatalf("container errado: %+v", created)
	}
	if len(host.shared) != 1 || filepath.Base(host.shared[0]) != filepath.Base(video) || host.open != 0 {
		t.Fatalf("link do vídeo: compartilhou %v, abertos %d", host.shared, host.open)
	}
	if pub := meta.find("ig/media_publish"); len(pub) != 1 || pub[0].form["creation_id"] != "id1" {
		t.Fatalf("publish errado: %+v", pub)
	}
	done, _ := f.store.List(publishq.StatusPublished)
	if len(done) != 1 || done[0].ID != out.ID || done[0].Link != "https://www.instagram.com/reel/teste123/" {
		t.Fatalf("não registrou publicado: %+v", done)
	}
	p := f.pushed()
	if last := p[len(p)-1]; last.title != "Reels de teste no ar" || !strings.Contains(last.body, "reel/teste123") {
		t.Fatalf("aviso errado: %+v", last)
	}
}

func TestExecutorTrialReelFailureRetries(t *testing.T) {
	f, meta, ex := newAdFixture(t)
	f.metaEnv(t)
	meta.failOn = "/media_publish"
	video := f.file(t, "lista.mp4", "v")
	if _, errText := callTool[requestOutput](t, f, "pedir_aprovacao", trialArgs(video)); errText != "" {
		t.Fatal(errText)
	}
	f.now = f.now.Add(6 * time.Hour)
	if err := ex.run(context.Background()); err == nil || !strings.Contains(err.Error(), "Objetivo indisponível") {
		t.Fatalf("devia falhar: %v", err)
	}
	approved, _ := f.store.List(publishq.StatusApproved)
	if len(approved) != 1 || approved[0].Attempts != 1 {
		t.Fatalf("falha passageira devia voltar na próxima rodada: %+v", approved)
	}
}

// "A Collecta compra cartas" (Jairo, 27/11): anúncio que abre conversa no
// WhatsApp da loja e otimiza conversas iniciadas (Click to WhatsApp).
func TestMessagesObjectiveOpensWhatsApp(t *testing.T) {
	f, meta, ex := newAdFixture(t)
	args := proposeArgs(f.creatives(t), 100)
	args["objetivo"] = "mensagens"
	args["publico"] = "brasil"
	if _, errText := callTool[proposeOutput](t, f, "propor_teste", args); errText != "" {
		t.Fatal(errText)
	}
	approveAds(t, f)
	if err := ex.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	campaigns := meta.find("act_1/campaigns")
	if len(campaigns) != 1 || campaigns[0].form["objective"] != "OUTCOME_ENGAGEMENT" {
		t.Fatalf("campanha = %+v", campaigns)
	}
	set := meta.find("act_1/adsets")[0].form
	if set["destination_type"] != "WHATSAPP" || set["optimization_goal"] != "CONVERSATIONS" || set["billing_event"] != "IMPRESSIONS" || !strings.Contains(set["promoted_object"], `"page_id":"page"`) || !strings.Contains(set["targeting"], `"countries":["BR"]`) {
		t.Fatalf("conjunto = %+v", set)
	}
	for i, c := range meta.find("act_1/adcreatives") {
		spec := c.form["object_story_spec"]
		if !strings.Contains(spec, `"type":"WHATSAPP_MESSAGE"`) || !strings.Contains(spec, `"app_destination":"WHATSAPP"`) {
			t.Fatalf("criativo %d sem botão do WhatsApp: %s", i, spec)
		}
	}
}

func TestMessagesMeasuredByConversations(t *testing.T) {
	var rows []insightRow
	if err := json.Unmarshal([]byte(`[{"ad_id":"a","spend":"30","clicks":"40","actions":[{"action_type":"onsite_conversion.messaging_conversation_started_7d","value":"6"}]},{"ad_id":"b","spend":"30","clicks":"50"}]`), &rows); err != nil {
		t.Fatal(err)
	}
	metric, count := measure("mensagens", rows)
	if metric != "conversas iniciadas" || count(rows[0]) != 6 || count(rows[1]) != 0 {
		t.Fatalf("métrica = %q %v %v", metric, count(rows[0]), count(rows[1]))
	}
}

// A verba de compra de cartas (mensagens) é separada da de vendas: um teste de
// mensagens passa mesmo com a semana de vendas cheia.
func TestMessagesUseOwnBudget(t *testing.T) {
	f, _, _ := newAdFixture(t)
	for _, b := range []int{300, 300, 300, 50} { // vendas cheia: R$ 950
		if _, errText := callTool[proposeOutput](t, f, "propor_teste", proposeArgs(f.creatives(t), b)); errText != "" {
			t.Fatal(errText)
		}
	}
	if _, errText := callTool[proposeOutput](t, f, "propor_teste", proposeArgs(f.creatives(t), 30)); !strings.Contains(errText, "na semana") {
		t.Fatalf("vendas devia estar cheia: %q", errText)
	}
	msg := proposeArgs(f.creatives(t), 150)
	msg["objetivo"] = "mensagens"
	if _, errText := callTool[proposeOutput](t, f, "propor_teste", msg); errText != "" {
		t.Fatalf("mensagens tem verba própria: %q", errText)
	}
	msg2 := proposeArgs(f.creatives(t), 150)
	msg2["objetivo"] = "mensagens"
	if _, errText := callTool[proposeOutput](t, f, "propor_teste", msg2); !strings.Contains(errText, "compra de cartas") {
		t.Fatalf("mensagens passou de R$ 250 na semana: %q", errText)
	}
	status, _ := callTool[adStatusOutput](t, f, "testes_anuncio", nil)
	if status.SobraMensagens != "R$ 100,00" {
		t.Fatalf("sobra de compra de cartas = %q", status.SobraMensagens)
	}
}

func init() {
	// Sem rede nos testes: "nao-existe" no endereço faz de conta que a página dá 404.
	checkSitePage = func(_ context.Context, _ *http.Client, link string) error {
		if strings.Contains(link, "nao-existe") {
			return fmt.Errorf("a página %s respondeu 404: confira o endereço do produto", link)
		}
		return nil
	}
}

// Teste do Gem Pack (02 a 08/12): um anúncio do Vol.2 e um do Vol.3, cada um
// levando à página da sua caixa; o criativo sem link usa o link do teste.
func TestSalesLinkPerCreative(t *testing.T) {
	f, meta, ex := newAdFixture(t)
	args := proposeArgs(f.creatives(t), 100)
	args["objetivo"] = "vendas"
	args["link"] = "https://www.collectatcg.com.br/selado/pokemon"
	cs := args["criativos"].([]map[string]any)
	cs[0]["link"] = "https://www.collectatcg.com.br/selado/pokemon-gem-pack-vol-02"
	cs[1]["link"] = "https://collectatcg.com.br/selado/pokemon-gem-pack-vol-03?utm=x"
	if _, errText := callTool[proposeOutput](t, f, "propor_teste", args); errText != "" {
		t.Fatal(errText)
	}
	list, _ := f.ads.List("")
	if len(list) != 1 || list[0].Creatives[0].Link != "https://www.collectatcg.com.br/selado/pokemon-gem-pack-vol-02" || list[0].Creatives[1].Link != "https://www.collectatcg.com.br/selado/pokemon-gem-pack-vol-03" || list[0].Creatives[2].Link != "" {
		t.Fatalf("links guardados = %+v", list[0].Creatives)
	}
	if push := f.pushed()[len(f.pushed())-1]; !strings.Contains(push.message, "leva para: https://www.collectatcg.com.br/selado/pokemon-gem-pack-vol-02") {
		t.Fatalf("o dono devia ver o link de cada criativo: %q", push.message)
	}
	approveAds(t, f)
	if err := ex.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	cr := meta.find("act_1/adcreatives")
	want := []string{"pokemon-gem-pack-vol-02", "pokemon-gem-pack-vol-03", "/selado/pokemon\""}
	for i, w := range want {
		if spec := cr[i].form["object_story_spec"]; !strings.Contains(spec, w) || !strings.Contains(spec, "SHOP_NOW") {
			t.Fatalf("criativo %d devia levar a %s: %s", i, w, spec)
		}
	}
}

func TestCreativeLinkRefused(t *testing.T) {
	f, _, _ := newAdFixture(t)
	args := proposeArgs(f.creatives(t), 100)
	args["criativos"].([]map[string]any)[0]["link"] = "https://www.collectatcg.com.br/selado/pokemon"
	if _, errText := callTool[proposeOutput](t, f, "propor_teste", args); !strings.Contains(errText, "só existe em vendas") {
		t.Fatalf("seguidores com link por criativo: %q", errText)
	}
	args = proposeArgs(f.creatives(t), 100)
	args["objetivo"] = "vendas"
	args["link"] = "https://www.collectatcg.com.br/selado/pokemon"
	args["criativos"].([]map[string]any)[1]["link"] = "https://www.collectatcg.com.br/selado/nao-existe"
	if _, errText := callTool[proposeOutput](t, f, "propor_teste", args); !strings.Contains(errText, "criativo 2") || !strings.Contains(errText, "404") {
		t.Fatalf("página que não existe devia ser recusada: %q", errText)
	}
	if list, _ := f.ads.List(""); len(list) != 0 {
		t.Fatal("criou o teste mesmo recusando")
	}
}

func TestLookalikeAndBuyersAudiences(t *testing.T) {
	_, _, ex := newAdFixture(t)
	for audience, env := range map[string]string{"semelhante": "META_PUBLICO_SEMELHANTE", "clientes": "META_PUBLICO_COMPRADORES"} {
		if _, err := ex.targeting(context.Background(), audience); err == nil || !strings.Contains(err.Error(), env) {
			t.Fatalf("%s sem id devia falhar citando %s: %v", audience, env, err)
		}
	}
	ex.g.cfg.Lookalike = []string{"555"}
	ex.g.cfg.Buyers = []string{"999"}
	cases := map[string][]string{
		"semelhante": {`"custom_audiences":[{"id":"555"}]`, `"excluded_custom_audiences":[{"id":"999"}]`},
		"clientes":   {`"custom_audiences":[{"id":"999"}]`},
	}
	for audience, wants := range cases {
		tg, err := ex.targeting(context.Background(), audience)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(tg)
		s := string(data)
		for _, want := range append(wants, `"countries":["BR"]`, `"age_min":18`, `"age_max":65`, `"advantage_audience":0`, `"publisher_platforms":["facebook","instagram"]`) {
			if !strings.Contains(s, want) {
				t.Errorf("%s: falta %s em %s", audience, want, s)
			}
		}
		if strings.Contains(s, "flexible_spec") {
			t.Errorf("%s não leva interesse: %s", audience, s)
		}
		if audience == "clientes" && strings.Contains(s, "excluded_custom_audiences") {
			t.Errorf("clientes não exclui ninguém: %s", s)
		}
	}
}

// Sem o id do público no anuncios.env, a propor_teste recusa antes de criar o
// teste: nada vai para a fila, nada é pedido ao dono, nada é gasto.
func TestProposeRefusesMissingCustomAudience(t *testing.T) {
	f, _, _ := newAdFixture(t)
	env := filepath.Join(t.TempDir(), "anuncios.env")
	if err := os.WriteFile(env, []byte("META_TOKEN=x\nMETA_PUBLICO_REMARKETING=\nMETA_PUBLICO_COMPRADORES=999\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.app.cfg.AdsEnv = env
	for audience, key := range map[string]string{"semelhante": "META_PUBLICO_SEMELHANTE", "remarketing": "META_PUBLICO_REMARKETING"} {
		args := proposeArgs(f.creatives(t), 100)
		args["publico"] = audience
		if _, errText := callTool[proposeOutput](t, f, "propor_teste", args); !strings.Contains(errText, key) || !strings.Contains(errText, "Nada foi criado") {
			t.Fatalf("%s devia recusar citando %s: %q", audience, key, errText)
		}
	}
	if list, _ := f.ads.List(""); len(list) != 0 || len(f.pushed()) != 0 {
		t.Fatalf("não devia criar teste nem avisar: %d testes, %d avisos", len(list), len(f.pushed()))
	}
	args := proposeArgs(f.creatives(t), 100)
	args["publico"] = "clientes"
	if _, errText := callTool[proposeOutput](t, f, "propor_teste", args); errText != "" {
		t.Fatalf("clientes com META_PUBLICO_COMPRADORES devia passar: %s", errText)
	}
	list, _ := f.ads.List("")
	if len(list) != 1 || list[0].Audience != "clientes" {
		t.Fatalf("fila = %+v", list)
	}
}

func TestProposeLookalikeGoesLive(t *testing.T) {
	f, meta, ex := newAdFixture(t)
	env := filepath.Join(t.TempDir(), "anuncios.env")
	if err := os.WriteFile(env, []byte("META_PUBLICO_SEMELHANTE=555\nMETA_PUBLICO_COMPRADORES=999\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.app.cfg.AdsEnv = env
	ex.g.cfg.Lookalike = []string{"555"}
	ex.g.cfg.Buyers = []string{"999"}
	args := proposeArgs(f.creatives(t), 100)
	args["publico"] = "semelhante"
	if _, errText := callTool[proposeOutput](t, f, "propor_teste", args); errText != "" {
		t.Fatal(errText)
	}
	if p := f.pushed(); len(p) != 1 || !strings.Contains(p[0].message, "Semelhante aos clientes") {
		t.Fatalf("aviso = %+v", p)
	}
	approveAds(t, f)
	if err := ex.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	set := meta.find("act_1/adsets")
	if len(set) != 1 || !strings.Contains(set[0].form["targeting"], `"custom_audiences":[{"id":"555"}]`) || !strings.Contains(set[0].form["targeting"], `"excluded_custom_audiences":[{"id":"999"}]`) {
		t.Fatalf("conjunto = %+v", set)
	}
}
