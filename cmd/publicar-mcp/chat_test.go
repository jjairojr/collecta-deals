package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"opdeals/internal/adtestq"
	"opdeals/internal/publishq"
)

const otherAgent = "agent-fb4761f9-27ed-4da0-ad0b-6d27f4388143"

type chatDB struct {
	path  string
	clock time.Time
	n     int
}

func sqlite(t *testing.T, db, query string) {
	t.Helper()
	if out, err := exec.Command("/usr/bin/sqlite3", db, query).CombinedOutput(); err != nil {
		t.Fatalf("sqlite3: %v %s", err, out)
	}
}

// newChatDB monta um banco com o mesmo formato das tabelas do OpenBot e liga a aprovação pelo chat.
func newChatDB(t *testing.T, f *fixture) *chatDB {
	t.Helper()
	db := &chatDB{path: filepath.Join(t.TempDir(), "openbot.db"), clock: f.now.Add(time.Minute)}
	sqlite(t, db.path, `CREATE TABLE projection_threads (thread_id TEXT PRIMARY KEY, agent_id TEXT NOT NULL, title TEXT NOT NULL);
CREATE TABLE projection_thread_messages (thread_id TEXT NOT NULL, message_id TEXT NOT NULL, author TEXT NOT NULL, item_type TEXT, created_at TEXT NOT NULL, ordinal INTEGER NOT NULL, message_json TEXT NOT NULL, PRIMARY KEY(thread_id, message_id));
INSERT INTO projection_threads VALUES ('t-ceo','`+ceoCollecta+`','CEO'),('t-social','`+otherAgent+`','Social');`)
	f.app.cfg.OpenBotDB = db.path
	if err := os.MkdirAll(f.data, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.data, chatModeFile), []byte("# chats que valem\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return db
}

func (d *chatDB) add(t *testing.T, thread, author, itemType, source, text string) {
	t.Helper()
	d.n++
	d.clock = d.clock.Add(time.Minute)
	typ := "NULL"
	if itemType != "" {
		typ = "'" + itemType + "'"
	}
	esc := func(s string) string { return strings.ReplaceAll(s, "'", "''") }
	msg := fmt.Sprintf(`{"id":"m%d","author":"%s","source":"%s","text":"%s"}`, d.n, author, source, strings.ReplaceAll(text, `"`, `\"`))
	sqlite(t, d.path, fmt.Sprintf(`INSERT INTO projection_thread_messages VALUES ('%s','m%d','%s',%s,'%s',%d,'%s')`,
		thread, d.n, author, typ, d.clock.UTC().Format("2006-01-02T15:04:05.000Z"), d.n, esc(msg)))
}

func (d *chatDB) ceo(t *testing.T, text string) {
	d.add(t, "t-ceo", "assistant", "agentMessage", "", text)
}
func (d *chatDB) jairo(t *testing.T, text string) { d.add(t, "t-ceo", "user", "", "user", text) }

func postArgs(f *fixture, t *testing.T, name string) map[string]any {
	args := reelsArgs(f.file(t, name, "png"))
	args["formato"] = "post"
	return args
}

func decide(t *testing.T, f *fixture, code, action string, check bool) (chatDecideOutput, string) {
	return callTool[chatDecideOutput](t, f, "aprovar_pelo_chat", map[string]any{"codigo": code, "acao": action, "so_conferir": check})
}

func TestChatRequestGoesToChatNotPhone(t *testing.T) {
	f := newFixture(t)
	newChatDB(t, f)
	out, errText := callTool[requestOutput](t, f, "pedir_aprovacao", postArgs(f, t, "arte.png"))
	if errText != "" {
		t.Fatal(errText)
	}
	if len(f.pushed()) != 0 {
		t.Fatalf("nada devia ir ao celular: %+v", f.pushed())
	}
	if out.Codigo != codeOf(out.ID) || !strings.Contains(out.Proximo, "CEO Collecta") {
		t.Fatalf("saída errada: %+v", out)
	}
	info, err := os.Stat(filepath.Join(f.data, chatTicketDir, out.ID+".json"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("código de aprovação não ficou guardado só para o dono: %v %v", info, err)
	}
	w, errText := callTool[waitingOutput](t, f, "esperando_jairo", nil)
	if errText != "" || !w.Ligado || len(w.Itens) != 1 || w.Itens[0].Codigo != out.Codigo || !strings.Contains(w.Itens[0].Resumo, "Faltam 10 dias") {
		t.Fatalf("esperando_jairo errado: %+v %s", w, errText)
	}
}

func TestChatApprovalNeedsJairosOwnWords(t *testing.T) {
	f := newFixture(t)
	db := newChatDB(t, f)
	out, _ := callTool[requestOutput](t, f, "pedir_aprovacao", postArgs(f, t, "arte.png"))
	code := out.Codigo
	if _, errText := decide(t, f, code, "aprovar", false); !strings.Contains(errText, "não achei mensagem do Jairo") {
		t.Fatalf("sem frase devia recusar: %s", errText)
	}
	db.ceo(t, "Peça de hoje, código "+code+". Aprova?")
	db.add(t, "t-ceo", "agent", "agent-exchange", "", "aprovo "+code)
	db.add(t, "t-ceo", "user", "routine", "routine", "aprovo "+code)
	db.add(t, "t-social", "user", "", "user", "aprovo "+code)
	if _, errText := decide(t, f, code, "aprovar", false); !strings.Contains(errText, "não achei mensagem do Jairo") {
		t.Fatalf("agente, rotina ou outro chat não podem aprovar: %s", errText)
	}
	db.jairo(t, "aprovo")
	chk, errText := decide(t, f, code, "aprovar", true)
	if errText != "" || chk.Feito || chk.Frase != "aprovo" {
		t.Fatalf("conferir errado: %+v %s", chk, errText)
	}
	if pending, _ := f.store.List(publishq.StatusPending); len(pending) != 1 {
		t.Fatal("só conferir não podia aprovar")
	}
	if _, errText := decide(t, f, code, "rejeitar", false); !strings.Contains(errText, "escreveu aprovar") {
		t.Fatalf("não pode inverter a decisão dele: %s", errText)
	}
	got, errText := decide(t, f, code, "aprovar", false)
	if errText != "" || !got.Feito || got.Mensagem == "" {
		t.Fatalf("aprovar errado: %+v %s", got, errText)
	}
	if approved, _ := f.store.List(publishq.StatusApproved); len(approved) != 1 || approved[0].ID != out.ID {
		t.Fatalf("devia estar aprovado: %+v", approved)
	}
	if _, errText := decide(t, f, code, "aprovar", false); !strings.Contains(errText, "nenhum pedido esperando") {
		t.Fatalf("não pode decidir duas vezes: %s", errText)
	}
}

func TestChatApprovalRefusesUnclearPhrases(t *testing.T) {
	f := newFixture(t)
	db := newChatDB(t, f)
	a, _ := callTool[requestOutput](t, f, "pedir_aprovacao", postArgs(f, t, "a.png"))
	b, _ := callTool[requestOutput](t, f, "pedir_aprovacao", postArgs(f, t, "b.png"))
	db.ceo(t, fmt.Sprintf("Duas peças: %s e %s.", a.Codigo, b.Codigo))
	db.jairo(t, "aprovo")
	if _, errText := decide(t, f, a.Codigo, "aprovar", true); !strings.Contains(errText, "mais de um pedido") {
		t.Fatalf("resposta sem dizer qual devia recusar: %s", errText)
	}
	db.ceo(t, fmt.Sprintf("Qual? %s ou %s?", a.Codigo, b.Codigo))
	db.jairo(t, "aprovo a primeira mas não pode postar a segunda")
	if _, errText := decide(t, f, a.Codigo, "aprovar", true); !strings.Contains(errText, "não decide") {
		t.Fatalf("frase contraditória devia recusar: %s", errText)
	}
	db.ceo(t, "Só a "+a.Codigo+"?")
	db.jairo(t, "aprovo a "+a.Codigo+" mas não pode postar ainda")
	if _, errText := decide(t, f, a.Codigo, "aprovar", true); !strings.Contains(errText, "aprova e nega") {
		t.Fatalf("frase contraditória devia recusar: %s", errText)
	}
	db.ceo(t, fmt.Sprintf("Me diga pelo código: %s, %s.", a.Codigo, b.Codigo))
	db.jairo(t, "aprovo "+strings.ToLower(a.Codigo))
	if _, errText := decide(t, f, a.Codigo, "aprovar", false); errText != "" {
		t.Fatal(errText)
	}
	if _, errText := decide(t, f, b.Codigo, "aprovar", true); errText == "" {
		t.Fatalf("aprovar um código não aprova o outro: %s", errText)
	}
	db.ceo(t, "E a "+b.Codigo+"?")
	db.jairo(t, "essa não, rejeita")
	if _, errText := decide(t, f, b.Codigo, "rejeitar", false); errText != "" {
		t.Fatal(errText)
	}
	if rejected, _ := f.store.List(publishq.StatusRejected); len(rejected) != 1 || rejected[0].ID != b.ID {
		t.Fatalf("devia estar rejeitado: %+v", rejected)
	}
}

func TestChatApprovesBothWithAllWord(t *testing.T) {
	f := newFixture(t)
	db := newChatDB(t, f)
	a, _ := callTool[requestOutput](t, f, "pedir_aprovacao", postArgs(f, t, "a.png"))
	b, _ := callTool[requestOutput](t, f, "pedir_aprovacao", postArgs(f, t, "b.png"))
	db.ceo(t, fmt.Sprintf("Duas peças: %s e %s.", a.Codigo, b.Codigo))
	db.jairo(t, "Pode postar as duas")
	for _, c := range []string{a.Codigo, b.Codigo} {
		if _, errText := decide(t, f, c, "aprovar", false); errText != "" {
			t.Fatal(errText)
		}
	}
}

func TestChatAdNeedsBudgetInPhrase(t *testing.T) {
	f, _, _ := newAdFixture(t)
	db := newChatDB(t, f)
	out, errText := callTool[proposeOutput](t, f, "propor_teste", proposeArgs(f.creatives(t), 150))
	if errText != "" || out.Codigo == "" || len(f.pushed()) != 0 {
		t.Fatalf("proposta pelo chat errada: %+v %s %+v", out, errText, f.pushed())
	}
	db.ceo(t, "Teste de anúncio "+out.Codigo+", R$ 150 em 4 dias.")
	db.jairo(t, "aprovo")
	if _, errText := decide(t, f, out.Codigo, "aprovar", false); !strings.Contains(errText, "R$ 150") {
		t.Fatalf("anúncio sem valor devia recusar: %s", errText)
	}
	db.ceo(t, "Preciso do valor para o "+out.Codigo+".")
	db.jairo(t, "confirmo o teste com R$ 150,00")
	if _, errText := decide(t, f, out.Codigo, "aprovar", false); errText != "" {
		t.Fatal(errText)
	}
	if approved, _ := f.ads.List(adtestq.StatusApproved); len(approved) != 1 || approved[0].ID != out.ID {
		t.Fatalf("teste devia estar aprovado: %+v", approved)
	}
}

func TestChatOffKeepsPhone(t *testing.T) {
	f := newFixture(t)
	out, errText := callTool[requestOutput](t, f, "pedir_aprovacao", postArgs(f, t, "arte.png"))
	if errText != "" || out.Codigo != "" || len(f.pushed()) != 1 {
		t.Fatalf("sem o arquivo de ligar, segue o celular: %+v %s", out, errText)
	}
	if _, errText := decide(t, f, codeOf(out.ID), "aprovar", false); !strings.Contains(errText, "desligada") {
		t.Fatalf("aprovar_pelo_chat devia recusar com o chat desligado: %s", errText)
	}
}

// fakeSlideshow troca o node+ffmpeg por um arquivo e guarda as imagens recebidas.
func fakeSlideshow(got *[]string) slideshowFunc {
	return func(_ context.Context, output string, images []string) error {
		*got = append([]string(nil), images...)
		if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
			return err
		}
		return os.WriteFile(output, []byte("video de "+strings.Join(images, ",")), 0o644)
	}
}

func approvedCarousel(t *testing.T, f *fixture, db *chatDB) (requestOutput, []string) {
	t.Helper()
	slides := []string{f.file(t, "s1.jpg", "um"), f.file(t, "s2.jpg", "dois")}
	args := reelsArgs(slides[0])
	args["formato"], args["arquivos"] = "carrossel", slides
	out, errText := callTool[requestOutput](t, f, "pedir_aprovacao", args)
	if errText != "" {
		t.Fatal(errText)
	}
	db.ceo(t, "Carrossel "+out.Codigo+" para hoje.")
	db.jairo(t, "aprovo")
	if _, errText := decide(t, f, out.Codigo, "aprovar", false); errText != "" {
		t.Fatal(errText)
	}
	return out, slides
}

func tiktokArgs(code string) map[string]any {
	return map[string]any{"canal": "tiktok", "formato": "video", "arquivos": []string{}, "legenda": "outra legenda", "publicar_em": "2026-11-06T18:00", "do_instagram": code}
}

func TestTiktokOnlyTheSameApprovedPiece(t *testing.T) {
	f := newFixture(t)
	db := newChatDB(t, f)
	f.app.cfg.MotionDir = filepath.Join(f.media, "motion")
	var got []string
	f.app.slideshow = fakeSlideshow(&got)

	pending, _ := callTool[requestOutput](t, f, "pedir_aprovacao", postArgs(f, t, "nova.png"))
	if _, errText := callTool[requestOutput](t, f, "pedir_aprovacao", tiktokArgs(pending.Codigo)); !strings.Contains(errText, "não foi aprovado") {
		t.Fatalf("post sem aprovação do Jairo não pode ir ao TikTok: %s", errText)
	}
	if _, errText := callTool[requestOutput](t, f, "pedir_aprovacao", tiktokArgs("ABCDEF")); !strings.Contains(errText, "nenhum post") {
		t.Fatalf("código inventado devia recusar: %s", errText)
	}

	ig, slides := approvedCarousel(t, f, db)
	out, errText := callTool[requestOutput](t, f, "pedir_aprovacao", tiktokArgs(strings.ToLower(ig.Codigo)))
	if errText != "" {
		t.Fatal(errText)
	}
	if len(got) != 0 {
		t.Fatalf("com o modo foto não monta vídeo: %v", got)
	}
	approved, _ := f.store.List(publishq.StatusApproved)
	var tt, igItem publishq.Item
	for _, it := range approved {
		switch it.ID {
		case out.ID:
			tt = it
		case ig.ID:
			igItem = it
		}
	}
	if tt.Channel != "tiktok" || tt.Format != "foto" || tt.Caption != "Faltam 10 dias pra inauguração 💙 #pokemontcg" || !strings.Contains(tt.Note, ig.Codigo) || out.Arquivos != len(slides) {
		t.Fatalf("TikTok devia sair aprovado, em foto, com a legenda aprovada: %+v", tt)
	}
	for i := range tt.Files {
		if tt.Files[i].SHA256 != igItem.Files[i].SHA256 {
			t.Fatalf("as fotos do TikTok têm de ser os mesmos arquivos aprovados: %+v", tt.Files)
		}
	}
	if _, errText := callTool[requestOutput](t, f, "pedir_aprovacao", tiktokArgs(ig.Codigo)); !strings.Contains(errText, "já foi mandado") {
		t.Fatalf("a mesma peça não vai duas vezes ao TikTok: %s", errText)
	}
	for _, p := range f.pushed() {
		if strings.Contains(p.title, "TikTok") {
			t.Fatalf("nada vai ao celular: %+v", p)
		}
	}
}

func TestTiktokRefusesChangedFile(t *testing.T) {
	f := newFixture(t)
	db := newChatDB(t, f)
	f.app.cfg.MotionDir = filepath.Join(f.media, "motion")
	var got []string
	f.app.slideshow = fakeSlideshow(&got)
	ig, slides := approvedCarousel(t, f, db)
	if err := os.WriteFile(slides[1], []byte("trocado"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, errText := callTool[requestOutput](t, f, "pedir_aprovacao", tiktokArgs(ig.Codigo)); !strings.Contains(errText, "mudou depois da aprovação") {
		t.Fatalf("arquivo trocado depois da aprovação não pode ir: %s", errText)
	}
	if len(got) != 0 {
		t.Fatal("não devia montar vídeo")
	}
}

func TestTiktokFallsBackToVideoWhenServerRefusesPhoto(t *testing.T) {
	f := newFixture(t)
	db := newChatDB(t, f)
	f.app.cfg.MotionDir = filepath.Join(f.media, "motion")
	f.refuseFoto = true
	var got []string
	f.app.slideshow = fakeSlideshow(&got)
	ig, slides := approvedCarousel(t, f, db)
	out, errText := callTool[requestOutput](t, f, "pedir_aprovacao", tiktokArgs(ig.Codigo))
	if errText != "" {
		t.Fatal(errText)
	}
	if len(got) != len(slides) {
		t.Fatalf("sem modo foto no painel, devia montar o vídeo com as imagens aprovadas: %v", got)
	}
	approved, _ := f.store.List(publishq.StatusApproved)
	for _, it := range approved {
		if it.ID == out.ID && it.Format != "video" {
			t.Fatalf("devia cair para vídeo: %+v", it)
		}
	}
}

func TestChatOneMessageDecidesEachCode(t *testing.T) {
	f := newFixture(t)
	db := newChatDB(t, f)
	a, _ := callTool[requestOutput](t, f, "pedir_aprovacao", postArgs(f, t, "a.png"))
	b, _ := callTool[requestOutput](t, f, "pedir_aprovacao", postArgs(f, t, "b.png"))
	c, _ := callTool[requestOutput](t, f, "pedir_aprovacao", postArgs(f, t, "c.png"))
	db.ceo(t, fmt.Sprintf("Três peças: %s, %s e %s.", a.Codigo, b.Codigo, c.Codigo))
	db.jairo(t, fmt.Sprintf("aprovo %s, rejeito %s e rejeito %s", a.Codigo, b.Codigo, c.Codigo))
	if _, errText := decide(t, f, a.Codigo, "aprovar", false); errText != "" {
		t.Fatalf("A devia ser aprovada: %s", errText)
	}
	for _, code := range []string{b.Codigo, c.Codigo} {
		if _, errText := decide(t, f, code, "rejeitar", false); errText != "" {
			t.Fatalf("%s devia ser rejeitada: %s", code, errText)
		}
	}
	d, _ := callTool[requestOutput](t, f, "pedir_aprovacao", postArgs(f, t, "d.png"))
	e, _ := callTool[requestOutput](t, f, "pedir_aprovacao", postArgs(f, t, "e.png"))
	db.ceo(t, fmt.Sprintf("Mais duas: %s e %s.", d.Codigo, e.Codigo))
	db.jairo(t, fmt.Sprintf("aprovo %s e %s", d.Codigo, e.Codigo))
	if _, errText := decide(t, f, e.Codigo, "aprovar", true); errText != "" {
		t.Fatalf("código sem verbo herda o anterior: %s", errText)
	}
	g, _ := callTool[requestOutput](t, f, "pedir_aprovacao", postArgs(f, t, "g.png"))
	db.ceo(t, "E a "+g.Codigo+"?")
	db.jairo(t, "não aprovo "+g.Codigo)
	if _, errText := decide(t, f, g.Codigo, "aprovar", true); !strings.Contains(errText, "escreveu rejeitar") {
		t.Fatalf("\"não aprovo\" é rejeição: %s", errText)
	}
}
