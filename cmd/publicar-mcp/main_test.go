package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"opdeals/internal/publishq"
)

type ntfyCall struct {
	title    string
	message  string
	filename string
	actions  string
	priority string
	body     string
}

type fixture struct {
	app     *app
	session *mcp.ClientSession
	store   *publishq.Store
	media   string
	data    string
	now     time.Time
	mu      sync.Mutex
	pushes  []ntfyCall
}

func decodeHeader(v string) string {
	out, err := new(mime.WordDecoder).DecodeHeader(v)
	if err != nil {
		return v
	}
	return out
}

func (f *fixture) serveDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Admin-Token") != "segredo" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeErr := func(err error) {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, publishq.ErrNotFound):
			status = http.StatusNotFound
		case errors.Is(err, publishq.ErrNotApproved), errors.Is(err, publishq.ErrNotPending):
			status = http.StatusConflict
		case errors.Is(err, publishq.ErrInvalidInput):
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
	}
	id := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/publish/"), "/")[0]
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/api/publish":
		var in publishq.Item
		_ = json.NewDecoder(r.Body).Decode(&in)
		it, err := f.store.Add(in)
		if err != nil {
			writeErr(err)
			return
		}
		_ = json.NewEncoder(w).Encode(it)
	case r.Method == http.MethodGet && r.URL.Path == "/api/publish":
		all, err := f.store.List(r.URL.Query().Get("status"))
		if err != nil {
			writeErr(err)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string][]publishq.Item{"items": all})
	case strings.HasSuffix(r.URL.Path, "/published"):
		var in struct {
			Link string `json:"link"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		it, err := f.store.Published(id, in.Link)
		if err != nil {
			writeErr(err)
			return
		}
		_ = json.NewEncoder(w).Encode(it)
	case strings.HasSuffix(r.URL.Path, "/failed"):
		var in struct {
			Error     string `json:"error"`
			Permanent bool   `json:"permanent"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		it, err := f.store.Failed(id, in.Error, in.Permanent)
		if err != nil {
			writeErr(err)
			return
		}
		_ = json.NewEncoder(w).Encode(it)
	default:
		http.NotFound(w, r)
	}
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	root := t.TempDir()
	f := &fixture{
		media: filepath.Join(root, "store"),
		data:  filepath.Join(root, "publicar"),
		now:   time.Date(2026, 11, 6, 10, 0, 0, 0, saoPaulo),
	}
	if err := os.MkdirAll(filepath.Join(f.media, "motion", "out"), 0o755); err != nil {
		t.Fatal(err)
	}
	f.store = publishq.NewStoreWithClock(filepath.Join(root, "publishq.json"), func() time.Time { return f.now })
	dash := httptest.NewServer(http.HandlerFunc(f.serveDashboard))
	t.Cleanup(dash.Close)
	ntfy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.pushes = append(f.pushes, ntfyCall{
			title: decodeHeader(r.Header.Get("Title")), message: decodeHeader(r.Header.Get("Message")),
			filename: r.Header.Get("Filename"), actions: r.Header.Get("Actions"), priority: r.Header.Get("Priority"), body: string(body),
		})
		f.mu.Unlock()
	}))
	t.Cleanup(ntfy.Close)
	cfg := config{OpdealsURL: dash.URL, AdminToken: "segredo", NtfyURL: ntfy.URL + "/topico", DataDir: f.data, MediaRoots: []string{f.media}}
	f.app = newApp(cfg, http.DefaultClient, func() time.Time { return f.now })
	ctx := context.Background()
	serverSide, clientSide := mcp.NewInMemoryTransports()
	if _, err := f.app.server().Connect(ctx, serverSide, nil); err != nil {
		t.Fatal(err)
	}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "teste"}, nil).Connect(ctx, clientSide, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	f.session = session
	return f
}

func (f *fixture) file(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(f.media, "motion", "out", name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func (f *fixture) pushed() []ntfyCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]ntfyCall(nil), f.pushes...)
}

func callTool[T any](t *testing.T, f *fixture, name string, args map[string]any) (T, string) {
	t.Helper()
	var out T
	res, err := f.session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if res.IsError {
		text := ""
		if len(res.Content) > 0 {
			if tc, ok := res.Content[0].(*mcp.TextContent); ok {
				text = tc.Text
			}
		}
		return out, text
	}
	data, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out, ""
}

func reelsArgs(path string) map[string]any {
	return map[string]any{
		"canal": "instagram", "formato": "reels", "arquivos": []string{path},
		"legenda": "Faltam 10 dias pra inauguração 💙 #pokemontcg", "publicar_em": "2026-11-06T15:00", "motivo": "ação de hoje",
	}
}

func approveAll(t *testing.T, f *fixture) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(filepath.Dir(f.data), "publishq.json"))
	if err != nil {
		t.Fatal(err)
	}
	var all []publishq.Item
	if err := json.Unmarshal(data, &all); err != nil {
		t.Fatal(err)
	}
	for _, it := range all {
		if it.Status == publishq.StatusPending {
			if _, err := f.store.Decide(it.ID, it.Token, true); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestLoadConfig(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	if _, err := loadConfig(env(map[string]string{"NTFY_TOPIC": "x"})); err == nil {
		t.Fatal("sem token devia falhar")
	}
	if _, err := loadConfig(env(map[string]string{"OPDEALS_ADMIN_TOKEN": "t"})); err == nil {
		t.Fatal("sem ntfy devia falhar")
	}
	cfg, err := loadConfig(env(map[string]string{"OPDEALS_ADMIN_TOKEN": "t", "NTFY_TOPIC": "abc", "PUBLICAR_MIDIA": "/a/store:/b/out/"}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.NtfyURL != "https://ntfy.sh/abc" || strings.Join(cfg.MediaRoots, ",") != "/a/store,/b/out" || cfg.DataDir != "/Users/Shared/collecta-bot/publicar" {
		t.Fatalf("config errada: %+v", cfg)
	}
}

func TestRequestSendsPreviewWithApproveButtons(t *testing.T) {
	f := newFixture(t)
	video := f.file(t, "loja.mp4", "video")
	out, errText := callTool[requestOutput](t, f, "pedir_aprovacao", reelsArgs(video))
	if errText != "" {
		t.Fatal(errText)
	}
	if out.ID == "" || out.Quando != "06/11 15:00" || !out.Previa {
		t.Fatalf("saída errada: %+v", out)
	}
	p := f.pushed()
	if len(p) != 1 || p[0].filename != "loja.mp4" || p[0].body != "video" || p[0].title != "Aprovar post da Collecta" {
		t.Fatalf("push errado: %+v", p)
	}
	if !strings.Contains(p[0].message, "Faltam 10 dias pra inauguração 💙") || !strings.Contains(p[0].message, "Por quê: ação de hoje") {
		t.Fatalf("mensagem errada: %q", p[0].message)
	}
	if !strings.Contains(p[0].actions, "/api/aprovar/"+out.ID+"?acao=aprovar&t=") || !strings.Contains(p[0].actions, "acao=rejeitar") || !strings.Contains(p[0].actions, "method=POST") {
		t.Fatalf("botões errados: %q", p[0].actions)
	}
	list, _ := f.store.List(publishq.StatusPending)
	if len(list) != 1 || list[0].Files[0].SHA256 == "" || list[0].Note != "ação de hoje" {
		t.Fatalf("pedido não ficou pendente: %+v", list)
	}
}

func TestRequestBigVideoGoesWithoutAttachment(t *testing.T) {
	f := newFixture(t)
	big := f.file(t, "grande.mp4", "")
	if err := os.Truncate(big, attachMax+1); err != nil {
		t.Fatal(err)
	}
	out, errText := callTool[requestOutput](t, f, "pedir_aprovacao", reelsArgs(big))
	if errText != "" || out.Previa {
		t.Fatalf("vídeo grande não devia ir anexado: %+v %s", out, errText)
	}
	p := f.pushed()
	if len(p) != 1 || p[0].filename != "" || !strings.Contains(p[0].body, "grande.mp4") {
		t.Fatalf("push errado: %+v", p[0])
	}
}

func TestRequestRefuses(t *testing.T) {
	f := newFixture(t)
	video := f.file(t, "ok.mp4", "v")
	image := f.file(t, "ok.png", "i")
	outside := filepath.Join(t.TempDir(), "fora.mp4")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(f.media, "motion", "out", "atalho.mp4")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	with := func(k string, v any) map[string]any {
		a := reelsArgs(video)
		a[k] = v
		return a
	}
	cases := []struct {
		name string
		args map[string]any
		want string
	}{
		{"fora da pasta", with("arquivos", []string{outside}), "fora das pastas"},
		{"atalho para fora", with("arquivos", []string{link}), "fora das pastas"},
		{"caminho relativo", with("arquivos", []string{"motion/out/ok.mp4"}), "completo"},
		{"não existe", with("arquivos", []string{filepath.Join(f.media, "nada.mp4")}), "não existe"},
		{"reels com imagem", with("arquivos", []string{image}), "vídeo"},
		{"reels com 2 arquivos", with("arquivos", []string{video, video}), "exatamente 1"},
		{"travessão", with("legenda", "Chegou — corre"), "travessão"},
		{"legenda vazia", with("legenda", "  "), "vazia"},
		{"horário passado", with("publicar_em", "2026-11-05T15:00"), "já passou"},
		{"horário longe", with("publicar_em", "2026-12-06T15:00"), "14 dias"},
		{"horário inválido", with("publicar_em", "amanhã"), "inválido"},
		{"story", with("formato", "story"), "não existe"},
		{"reels no tiktok", with("canal", "tiktok"), "não existe"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, errText := callTool[requestOutput](t, f, "pedir_aprovacao", c.args)
			if !strings.Contains(errText, c.want) {
				t.Fatalf("erro %q não contém %q", errText, c.want)
			}
		})
	}
	carousel := map[string]any{"canal": "instagram", "formato": "carrossel", "arquivos": []string{image}, "legenda": "x", "publicar_em": "2026-11-06T15:00"}
	if _, errText := callTool[requestOutput](t, f, "pedir_aprovacao", carousel); !strings.Contains(errText, "de 2 a 10") {
		t.Fatalf("carrossel com 1 arquivo: %q", errText)
	}
	if len(f.pushed()) != 0 {
		t.Fatal("pedido recusado não devia avisar o dono")
	}
	if all, _ := f.store.List(""); len(all) != 0 {
		t.Fatalf("pedido recusado não devia entrar na fila: %+v", all)
	}
}

func TestApprovedStagesFilesOnlyWhenDue(t *testing.T) {
	f := newFixture(t)
	video := f.file(t, "loja.mp4", "video")
	created, _ := callTool[requestOutput](t, f, "pedir_aprovacao", reelsArgs(video))
	before, _ := callTool[approvedOutput](t, f, "aprovados", nil)
	if len(before.Itens) != 0 {
		t.Fatalf("pendente não devia aparecer: %+v", before)
	}
	approveAll(t, f)
	early, _ := callTool[approvedOutput](t, f, "aprovados", nil)
	if len(early.Itens) != 0 || early.Agendados != 1 {
		t.Fatalf("antes do horário devia ficar agendado: %+v", early)
	}
	f.now = time.Date(2026, 11, 6, 15, 1, 0, 0, saoPaulo)
	ready, errText := callTool[approvedOutput](t, f, "aprovados", nil)
	if errText != "" || len(ready.Itens) != 1 {
		t.Fatalf("devia vir 1 pronto: %+v %s", ready, errText)
	}
	it := ready.Itens[0]
	wantDir := filepath.Join(f.data, "aprovados", created.ID)
	if it.ID != created.ID || it.Legenda != "Faltam 10 dias pra inauguração 💙 #pokemontcg" || len(it.Arquivos) != 1 || filepath.Dir(it.Arquivos[0]) != wantDir {
		t.Fatalf("item errado: %+v", it)
	}
	if got, err := os.ReadFile(it.Arquivos[0]); err != nil || string(got) != "video" {
		t.Fatalf("cópia errada: %q %v", got, err)
	}
}

func TestApprovedRefusesFileChangedAfterApproval(t *testing.T) {
	f := newFixture(t)
	video := f.file(t, "loja.mp4", "video")
	callTool[requestOutput](t, f, "pedir_aprovacao", reelsArgs(video))
	approveAll(t, f)
	f.file(t, "loja.mp4", "outro video")
	f.now = time.Date(2026, 11, 6, 16, 0, 0, 0, saoPaulo)
	out, errText := callTool[approvedOutput](t, f, "aprovados", nil)
	if errText != "" || len(out.Itens) != 0 || len(out.Recusados) != 1 || !strings.Contains(out.Recusados[0], "mudou depois da aprovação") {
		t.Fatalf("arquivo trocado devia ser recusado: %+v %s", out, errText)
	}
	if failed, _ := f.store.List(publishq.StatusFailed); len(failed) != 1 {
		t.Fatalf("devia falhar de vez: %+v", failed)
	}
	last := f.pushed()[len(f.pushed())-1]
	if last.priority != "high" || last.title != "Post não saiu" {
		t.Fatalf("dono devia ser avisado: %+v", last)
	}
}

func TestPublishedNeedsPostLinkAndCleansUp(t *testing.T) {
	f := newFixture(t)
	video := f.file(t, "loja.mp4", "video")
	created, _ := callTool[requestOutput](t, f, "pedir_aprovacao", reelsArgs(video))
	approveAll(t, f)
	f.now = time.Date(2026, 11, 6, 15, 5, 0, 0, saoPaulo)
	callTool[approvedOutput](t, f, "aprovados", nil)
	for _, bad := range []string{"https://www.tiktok.com/@collecta/video/1", "http://www.instagram.com/reel/x/", "https://www.instagram.com/", "https://evil.com/instagram.com/reel/x"} {
		if _, errText := callTool[okOutput](t, f, "publicado", map[string]any{"id": created.ID, "link": bad}); !strings.Contains(errText, "não é de um post") {
			t.Fatalf("link %q devia ser recusado: %q", bad, errText)
		}
	}
	out, errText := callTool[okOutput](t, f, "publicado", map[string]any{"id": created.ID, "link": "https://www.instagram.com/reel/DAbc123/"})
	if errText != "" || out.Status != publishq.StatusPublished {
		t.Fatalf("publicado falhou: %+v %s", out, errText)
	}
	if _, err := os.Stat(filepath.Join(f.data, "aprovados", created.ID)); !os.IsNotExist(err) {
		t.Fatal("pasta do aprovado devia sumir depois de publicar")
	}
	last := f.pushed()[len(f.pushed())-1]
	if last.title != "Post publicado" || !strings.Contains(last.body, "https://www.instagram.com/reel/DAbc123/") {
		t.Fatalf("aviso de publicado errado: %+v", last)
	}
	logData, _ := os.ReadFile(filepath.Join(f.data, "publicacoes.log"))
	if !strings.Contains(string(logData), "publicado "+created.ID) {
		t.Fatalf("log errado: %q", logData)
	}
	if _, errText := callTool[okOutput](t, f, "publicado", map[string]any{"id": created.ID, "link": "https://www.instagram.com/reel/DAbc123/"}); !strings.Contains(errText, "não está aprovada") {
		t.Fatalf("segunda vez devia recusar: %q", errText)
	}
}

func TestFailedRetriesThenGivesUp(t *testing.T) {
	f := newFixture(t)
	video := f.file(t, "tt.mp4", "video")
	args := reelsArgs(video)
	args["canal"], args["formato"] = "tiktok", "video"
	created, errText := callTool[requestOutput](t, f, "pedir_aprovacao", args)
	if errText != "" {
		t.Fatal(errText)
	}
	approveAll(t, f)
	f.now = time.Date(2026, 11, 6, 15, 5, 0, 0, saoPaulo)
	for i := 1; i < publishq.MaxAttempts; i++ {
		out, errText := callTool[okOutput](t, f, "falhou", map[string]any{"id": created.ID, "motivo": "envio travou", "permanente": false})
		if errText != "" || out.Status != publishq.StatusApproved {
			t.Fatalf("tentativa %d devia continuar aprovada: %+v %s", i, out, errText)
		}
		if last := f.pushed()[len(f.pushed())-1]; last.priority != "default" {
			t.Fatalf("falha passageira não devia ser alta: %+v", last)
		}
	}
	out, _ := callTool[okOutput](t, f, "falhou", map[string]any{"id": created.ID, "motivo": "envio travou", "permanente": false})
	if out.Status != publishq.StatusFailed {
		t.Fatalf("terceira falha devia encerrar: %+v", out)
	}
	if last := f.pushed()[len(f.pushed())-1]; last.priority != "high" {
		t.Fatalf("falha final devia ser alta: %+v", last)
	}
	if _, errText := callTool[okOutput](t, f, "falhou", map[string]any{"id": created.ID, "motivo": " "}); !strings.Contains(errText, "motivo") {
		t.Fatalf("sem motivo devia falhar: %q", errText)
	}
}

func TestStartReturnsRules(t *testing.T) {
	f := newFixture(t)
	out, errText := callTool[startOutput](t, f, "inicio", nil)
	if errText != "" || out.Regras != instructions || len(out.Versao) != 8 || out.Agora != "06/11/2026 10:00" {
		t.Fatalf("inicio errado: %+v %s", out, errText)
	}
	if got := f.session.InitializeResult().Instructions; got != serverInstructions || len(got) > 2000 {
		t.Fatalf("instruções do servidor com %d bytes", len(got))
	}
}

func TestStatusShowsRecentQueue(t *testing.T) {
	f := newFixture(t)
	video := f.file(t, "loja.mp4", "video")
	created, _ := callTool[requestOutput](t, f, "pedir_aprovacao", reelsArgs(video))
	out, errText := callTool[statusOutput](t, f, "situacao", nil)
	if errText != "" || len(out.Itens) != 1 || out.Itens[0].Situacao != "esperando o dono" || out.Itens[0].ID != created.ID {
		t.Fatalf("situação errada: %+v %s", out, errText)
	}
	approveAll(t, f)
	f.now = time.Date(2026, 11, 6, 15, 5, 0, 0, saoPaulo)
	callTool[approvedOutput](t, f, "aprovados", nil)
	callTool[okOutput](t, f, "publicado", map[string]any{"id": created.ID, "link": "https://www.instagram.com/reel/X1/"})
	out, _ = callTool[statusOutput](t, f, "situacao", nil)
	if out.Itens[0].Situacao != "publicada" || out.Itens[0].Link != "https://www.instagram.com/reel/X1/" {
		t.Fatalf("publicada errada: %+v", out.Itens[0])
	}
	f.now = f.now.Add(8 * 24 * time.Hour)
	if out, _ = callTool[statusOutput](t, f, "situacao", nil); len(out.Itens) != 0 {
		t.Fatalf("mais de 7 dias não devia aparecer: %+v", out)
	}
}
