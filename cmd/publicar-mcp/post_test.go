package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"opdeals/internal/adtestq"
)

const (
	postCarousel = "17900000000000001"
	postReel     = "17900000000000002"
	postMusic    = "17900000000000003"
	postStory    = "17900000000000004"
	postOther    = "17900000000000005"
	postImage    = "17900000000000006"
)

// newPostFixture is the ad fixture with an anuncios.env pointing at the fake
// Meta and an account with posts, reels, a carousel, a story and a post with
// licensed music.
func newPostFixture(t *testing.T) (*fixture, *fakeMeta, *executor) {
	t.Helper()
	f, meta, ex := newAdFixture(t)
	base := strings.TrimSuffix(ex.g.cfg.BaseURL, "/v27.0")
	env := filepath.Join(t.TempDir(), "anuncios.env")
	content := "META_TOKEN=token-meta\nMETA_CONTA=act_1\nMETA_PAGINA=page\nMETA_INSTAGRAM_ID=ig\nMETA_URL=" + base + "\n"
	if err := os.WriteFile(env, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	f.app.cfg.AdsEnv = env
	ok := map[string]any{"eligible_to_boost": true}
	meta.media = map[string]map[string]any{
		postCarousel: {"id": postCarousel, "username": "tcgcollecta", "media_type": "CAROUSEL_ALBUM", "media_product_type": "FEED", "permalink": "https://www.instagram.com/p/AAA111/", "caption": "Graduadas PSA 10 chegaram", "media_url": base + "/cdn/a.jpg", "boost_eligibility_info": ok},
		postReel:     {"id": postReel, "username": "tcgcollecta", "media_type": "VIDEO", "media_product_type": "REELS", "permalink": "https://www.instagram.com/reel/BBB222/", "caption": "Abrindo o Gem Pack", "thumbnail_url": base + "/cdn/b.jpg", "boost_eligibility_info": ok},
		postMusic:    {"id": postMusic, "username": "tcgcollecta", "media_type": "VIDEO", "media_product_type": "REELS", "permalink": "https://www.instagram.com/reel/CCC333/", "boost_eligibility_info": map[string]any{"eligible_to_boost": false, "reason": "música com licença"}},
		postStory:    {"id": postStory, "username": "tcgcollecta", "media_type": "IMAGE", "media_product_type": "STORY", "permalink": "https://www.instagram.com/stories/tcgcollecta/1/", "boost_eligibility_info": ok},
		postOther:    {"id": postOther, "username": "outra_loja", "media_type": "IMAGE", "media_product_type": "FEED", "permalink": "https://www.instagram.com/p/EEE555/", "boost_eligibility_info": ok},
		postImage:    {"id": postImage, "username": "tcgcollecta", "media_type": "IMAGE", "media_product_type": "FEED", "permalink": "https://www.instagram.com/p/FFF666/", "caption": "Troca de cartas domingo", "media_url": base + "/cdn/f.jpg", "boost_eligibility_info": ok},
	}
	return f, meta, ex
}

func existingPostArgs(posts ...string) map[string]any {
	cs := make([]map[string]any, len(posts))
	for i, p := range posts {
		cs[i] = map[string]any{"post": p, "gancho": "post " + string(rune('A'+i))}
	}
	return proposeArgs(cs, 100)
}

func TestProposeExistingPosts(t *testing.T) {
	f, _, _ := newPostFixture(t)
	// Um carrossel pelo link e um reel pelo id, sem texto nem título.
	out, errText := callTool[proposeOutput](t, f, "propor_teste", existingPostArgs("https://www.instagram.com/p/AAA111/?igsh=x", postReel))
	if errText != "" {
		t.Fatal(errText)
	}
	list, _ := f.ads.List("")
	if len(list) != 1 || list[0].ID != out.ID || len(list[0].Creatives) != 2 {
		t.Fatalf("fila = %+v", list)
	}
	for i, want := range []struct{ id, link string }{{postCarousel, "https://www.instagram.com/p/AAA111/"}, {postReel, "https://www.instagram.com/reel/BBB222/"}} {
		c := list[0].Creatives[i]
		if c.Kind != "post" || c.MediaID != want.id || c.Permalink != want.link || !strings.Contains(c.Text, want.link) {
			t.Fatalf("criativo %d = %+v", i+1, c)
		}
		if _, err := os.Stat(c.Path); err != nil {
			t.Fatalf("criativo %d sem prévia: %v", i+1, err)
		}
	}
	p := f.pushed()
	if len(p) != 1 || p[0].filename != "01-capa.jpg" || !strings.Contains(p[0].message, "https://www.instagram.com/reel/BBB222/") {
		t.Fatalf("aviso ao dono = %+v", p)
	}
}

func TestProposeRefusesBadPosts(t *testing.T) {
	f, _, _ := newPostFixture(t)
	cases := []struct {
		name string
		args map[string]any
		want string
	}{
		{"música com licença", existingPostArgs(postCarousel, postMusic), "música com licença"},
		{"story", existingPostArgs(postCarousel, postStory), "story"},
		{"outra conta", existingPostArgs(postCarousel, postOther), "@outra_loja"},
		{"mesmo post duas vezes", existingPostArgs(postCarousel, "https://www.instagram.com/p/AAA111/"), "mesmo post"},
		{"link que não é do Instagram", existingPostArgs(postCarousel, "https://www.collectatcg.com.br/p/x"), "link do post"},
		{"post que não está na conta", existingPostArgs(postCarousel, "https://www.instagram.com/p/ZZZ999/"), "não achei"},
	}
	for _, c := range cases {
		if _, errText := callTool[proposeOutput](t, f, "propor_teste", c.args); !strings.Contains(errText, c.want) {
			t.Errorf("%s: devia recusar com %q: %q", c.name, c.want, errText)
		}
	}
	withText := existingPostArgs(postCarousel, postReel)
	withText["criativos"].([]map[string]any)[0]["texto"] = "texto novo"
	if _, errText := callTool[proposeOutput](t, f, "propor_teste", withText); !strings.Contains(errText, "legenda do post") {
		t.Errorf("texto com post devia recusar: %q", errText)
	}
	if list, _ := f.ads.List(""); len(list) != 0 || len(f.pushed()) != 0 {
		t.Fatalf("nada devia ser criado nem pedido: %d testes, %d avisos", len(list), len(f.pushed()))
	}
}

// Um teste pode misturar post existente e arquivo; e o mínimo continua 2.
func TestProposeMixesPostAndFileKeepsMinimum(t *testing.T) {
	f, _, _ := newPostFixture(t)
	file := f.creatives(t)[2]
	args := proposeArgs([]map[string]any{{"post": postCarousel, "gancho": "post"}, file}, 100)
	if _, errText := callTool[proposeOutput](t, f, "propor_teste", args); errText != "" {
		t.Fatal(errText)
	}
	if _, errText := callTool[proposeOutput](t, f, "propor_teste", existingPostArgs(postCarousel)); !strings.Contains(errText, "de 2 a 5 criativos") {
		t.Fatalf("um só post devia recusar: %q", errText)
	}
}

func TestExistingPostGoesLive(t *testing.T) {
	f, meta, ex := newPostFixture(t)
	if _, errText := callTool[proposeOutput](t, f, "propor_teste", existingPostArgs(postCarousel, postReel)); errText != "" {
		t.Fatal(errText)
	}
	approveAds(t, f)
	if err := ex.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if c := meta.find("act_1/campaigns"); len(c) != 1 || c[0].form["objective"] != "OUTCOME_TRAFFIC" {
		t.Fatalf("campanha = %+v", c)
	}
	if s := meta.find("act_1/adsets"); len(s) != 1 || s[0].form["optimization_goal"] != "PROFILE_VISIT" || s[0].form["destination_type"] != "INSTAGRAM_PROFILE" {
		t.Fatalf("conjunto = %+v", s)
	}
	crs := meta.find("act_1/adcreatives")
	if len(crs) != 2 {
		t.Fatalf("criativos = %+v", crs)
	}
	for i, id := range []string{postCarousel, postReel} {
		form := crs[i].form
		if form["source_instagram_media_id"] != id || form["object_id"] != "page" || form["instagram_user_id"] != "ig" || form["object_story_spec"] != "" || !strings.Contains(form["call_to_action"], "VIEW_INSTAGRAM_PROFILE") {
			t.Fatalf("criativo %d = %+v", i+1, form)
		}
	}
	running, _ := f.ads.List(adtestq.StatusRunning)
	if len(running) != 1 || len(running[0].Meta.Ads) != 2 || len(running[0].Meta.Refused) != 0 {
		t.Fatalf("no ar = %+v", running)
	}
}

func TestRefusedPostIsLeftOutAndReported(t *testing.T) {
	f, meta, ex := newPostFixture(t)
	if _, errText := callTool[proposeOutput](t, f, "propor_teste", existingPostArgs(postCarousel, postReel, postImage)); errText != "" {
		t.Fatal(errText)
	}
	approveAds(t, f)
	meta.failMedia = postImage // o Meta recusa o terceiro na hora de criar
	if err := ex.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	running, _ := f.ads.List(adtestq.StatusRunning)
	if len(running) != 1 || len(running[0].Meta.Ads) != 2 || len(running[0].Meta.Refused) != 1 || running[0].Meta.Refused[0].Creative != 2 {
		t.Fatalf("devia subir com 2 e 1 recusado: %+v", running)
	}
	var live string
	for _, p := range f.pushed() {
		if p.title == "Teste de anúncio no ar" {
			live = p.body
		}
	}
	if !strings.Contains(live, "o Meta recusou") || !strings.Contains(live, "música deste post tem licença") || !strings.Contains(live, "post C") {
		t.Fatalf("aviso de no ar = %q", live)
	}
	status, _ := callTool[adStatusOutput](t, f, "testes_anuncio", nil)
	if len(status.Testes) != 1 || len(status.Testes[0].Recusados) != 1 || !strings.Contains(status.Testes[0].Recusados[0], "FFF666") {
		t.Fatalf("testes_anuncio = %+v", status.Testes)
	}
}

func TestRefusedPostBelowMinimumFailsWithoutSpending(t *testing.T) {
	f, meta, ex := newPostFixture(t)
	if _, errText := callTool[proposeOutput](t, f, "propor_teste", existingPostArgs(postCarousel, postReel)); errText != "" {
		t.Fatal(errText)
	}
	approveAds(t, f)
	meta.failMedia = postReel
	_ = ex.run(context.Background())
	failed, _ := f.ads.List(adtestq.StatusFailed)
	if len(failed) != 1 || !strings.Contains(failed[0].Error, "o mínimo é 2") || !strings.Contains(failed[0].Error, "licença") {
		t.Fatalf("devia falhar: %+v", failed)
	}
	if meta.statusPosts("ACTIVE") != 0 || meta.statusPosts("PAUSED") == 0 {
		t.Fatalf("nada podia ficar ativo: ativos=%d pausados=%d", meta.statusPosts("ACTIVE"), meta.statusPosts("PAUSED"))
	}
}

// O Meta pode mudar de ideia depois da aprovação (o post foi editado com música):
// a conferência é refeita na hora de subir.
func TestPostCheckedAgainAtLaunch(t *testing.T) {
	f, meta, ex := newPostFixture(t)
	if _, errText := callTool[proposeOutput](t, f, "propor_teste", existingPostArgs(postCarousel, postReel, postImage)); errText != "" {
		t.Fatal(errText)
	}
	approveAds(t, f)
	meta.mu.Lock()
	meta.media[postReel]["boost_eligibility_info"] = map[string]any{"eligible_to_boost": false}
	meta.mu.Unlock()
	if err := ex.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	running, _ := f.ads.List(adtestq.StatusRunning)
	if len(running) != 1 || len(running[0].Meta.Refused) != 1 || !strings.Contains(running[0].Meta.Refused[0].Reason, "não deixa impulsionar") {
		t.Fatalf("o reel devia ficar de fora: %+v", running)
	}
	for _, c := range meta.find("act_1/adcreatives") {
		if c.form["source_instagram_media_id"] == postReel {
			t.Fatalf("não devia criar o criativo do reel recusado")
		}
	}
}

func TestStatusShowsResultsPerPost(t *testing.T) {
	f, meta, ex := newPostFixture(t)
	if _, errText := callTool[proposeOutput](t, f, "propor_teste", existingPostArgs(postCarousel, postReel)); errText != "" {
		t.Fatal(errText)
	}
	approveAds(t, f)
	if err := ex.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	running, _ := f.ads.List(adtestq.StatusRunning)
	ads := running[0].Meta.Ads
	meta.mu.Lock()
	meta.insights = []insightRow{{AdID: ads[0].ID, Spend: "12", Clicks: "40"}, {AdID: ads[1].ID, Spend: "8", Clicks: "0"}}
	meta.mu.Unlock()
	if err := ex.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	status, _ := callTool[adStatusOutput](t, f, "testes_anuncio", nil)
	got := status.Testes[0].Anuncios
	if len(got) != 2 || got[0].Post != "https://www.instagram.com/p/AAA111/" || got[0].Gasto != "R$ 12.00" || got[0].Resultado != "40 visitas ao perfil, R$ 0.30 cada" || got[1].Resultado != "0 visitas ao perfil" {
		t.Fatalf("anúncios = %+v", got)
	}
}
