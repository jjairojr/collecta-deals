package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const feedHeader = `"id","title","description","availability","condition","price","link","image_link","brand"`

type pecaSite struct {
	srv     *httptest.Server
	renders []pecaProps
}

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func newPecaFixture(t *testing.T) (*fixture, *pecaSite) {
	t.Helper()
	f := newFixture(t)
	site := &pecaSite{}
	photo := pngBytes(t, 200, 100)
	site.srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/feed/meta.csv":
			img := site.srv.URL + "/foto.png"
			rows := []string{
				feedHeader,
				fmt.Sprintf(`"1","Booster Box Megaevolução 6 - Reinado Delta — Caixa de Booster Pokémon lacrado","Pix ou cartão em até 6x e envio para todo o Brasil.","preorder","new","389.50 BRL","https://www.collectatcg.com.br/selado/reinado-box","%s","Copag"`, img),
				fmt.Sprintf(`"2","Luffy — Carta One Piece","Pix ou cartão em até 12x.","in stock","used","79.90 BRL","https://www.collectatcg.com.br/carta/luffy","%s","Bandai"`, img),
				fmt.Sprintf(`"3","Charizard — Carta Pokémon","Pix ou cartão em até 12x.","out of stock","used","999.00 BRL","https://www.collectatcg.com.br/carta/charizard","%s","Copag"`, img),
			}
			_, _ = w.Write([]byte(strings.Join(rows, "\n")))
		case "/foto.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(photo)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(site.srv.Close)
	motion := filepath.Join(f.media, "motion")
	if err := os.MkdirAll(filepath.Join(motion, "public", "products"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(motion, "public", "products", "box.png"), pngBytes(t, 100, 200), 0o644); err != nil {
		t.Fatal(err)
	}
	f.app.client = site.srv.Client()
	f.app.cfg.MotionDir = motion
	f.app.cfg.FeedURL = site.srv.URL + "/feed/meta.csv"
	f.app.render = func(_ context.Context, propsPath, kind, output string) (renderResult, error) {
		data, err := os.ReadFile(propsPath)
		if err != nil {
			return renderResult{}, err
		}
		var props pecaProps
		if err := json.Unmarshal(data, &props); err != nil {
			return renderResult{}, err
		}
		site.renders = append(site.renders, props)
		if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
			return renderResult{}, err
		}
		if err := os.WriteFile(output, []byte(kind), 0o644); err != nil {
			return renderResult{}, err
		}
		return renderResult{Saida: output, Previas: []string{output + ".png"}, Segundos: 11.2}, nil
	}
	return f, site
}

func pecaArgs(modelo, formato string, produtos ...map[string]any) map[string]any {
	return map[string]any{
		"modelo": modelo, "formato": formato, "etiqueta": "Pokémon · Selado", "gancho": "Reinado Delta chegou!",
		"destaque": []string{"Reinado"}, "selos": []string{"Em português"}, "produtos": produtos,
		"final": map[string]any{"titulo": "Garanta o seu!", "oferta": "Envio pra todo o Brasil", "linhas": []string{"collectatcg.com.br"}, "cta": "Link na bio"},
	}
}

func TestPecaUsesSitePriceAndPhoto(t *testing.T) {
	f, site := newPecaFixture(t)
	out, errText := callTool[pecaOutput](t, f, "criar_peca", pecaArgs("produto", "video",
		map[string]any{"link": "https://www.collectatcg.com.br/selado/reinado-box", "nome": "Booster Box"},
		map[string]any{"link": "https://collectatcg.com.br/carta/luffy?ref=x", "imagem": "products/box.png"},
	))
	if errText != "" {
		t.Fatal(errText)
	}
	if len(site.renders) != 1 {
		t.Fatalf("renders = %d", len(site.renders))
	}
	props := site.renders[0]
	first, second := props.Produtos[0], props.Produtos[1]
	if first.Preco != 389.5 || first.Parcelas != 6 || first.Nome != "Booster Box" || first.Proporcao != 2 {
		t.Fatalf("primeiro produto = %+v", first)
	}
	if !strings.HasPrefix(first.Imagem, "agente/") {
		t.Fatalf("foto do site não foi baixada para public/agente: %s", first.Imagem)
	}
	if _, err := os.Stat(filepath.Join(f.app.cfg.MotionDir, "public", first.Imagem)); err != nil {
		t.Fatal(err)
	}
	if second.Imagem != "products/box.png" || second.Proporcao != 0.5 || second.Nome != "Luffy" || second.Parcelas != 12 {
		t.Fatalf("segundo produto = %+v", second)
	}
	if !props.Music || props.Gancho != "Reinado Delta chegou!" || props.Final.CTA != "Link na bio" {
		t.Fatalf("textos = %+v", props)
	}
	if !strings.HasPrefix(out.Arquivo, filepath.Join(f.app.cfg.MotionDir, "out", "agente", "2026-11-06-produto-reinado-delta-chegou-")) || !strings.HasSuffix(out.Arquivo, ".mp4") {
		t.Fatalf("arquivo = %s", out.Arquivo)
	}
	if len(out.Precos) != 2 || out.Precos[0].Preco != 389.5 || out.Precos[1].Link != "https://www.collectatcg.com.br/carta/luffy" {
		t.Fatalf("preços = %+v", out.Precos)
	}
	if out.Dias != nil || len(out.Previas) != 1 {
		t.Fatalf("saída = %+v", out)
	}
}

func TestPecaArteReturnsImageAsPreview(t *testing.T) {
	f, _ := newPecaFixture(t)
	out, errText := callTool[pecaOutput](t, f, "criar_peca", pecaArgs("carta", "arte", map[string]any{"link": "https://www.collectatcg.com.br/carta/luffy"}))
	if errText != "" {
		t.Fatal(errText)
	}
	if len(out.Avisos) != 1 || !strings.Contains(out.Avisos[0], "SAMPLE") {
		t.Fatalf("avisos = %v", out.Avisos)
	}
	if !strings.HasSuffix(out.Arquivo, ".png") || len(out.Previas) != 1 || out.Previas[0] != out.Arquivo {
		t.Fatalf("saída = %+v", out)
	}
}

func TestPecaCountdownDays(t *testing.T) {
	f, site := newPecaFixture(t)
	args := pecaArgs("contagem", "video")
	out, errText := callTool[pecaOutput](t, f, "criar_peca", args)
	if errText != "" {
		t.Fatal(errText)
	}
	if out.Dias == nil || *out.Dias != 9 || site.renders[0].Dias != 9 {
		t.Fatalf("dias = %v", out.Dias)
	}
	args["para_o_dia"] = "2026-11-10"
	out, errText = callTool[pecaOutput](t, f, "criar_peca", args)
	if errText != "" {
		t.Fatal(errText)
	}
	if *out.Dias != 5 {
		t.Fatalf("dias = %d", *out.Dias)
	}
	f.now = time.Date(2026, 11, 16, 9, 0, 0, 0, saoPaulo)
	if _, errText = callTool[pecaOutput](t, f, "criar_peca", pecaArgs("contagem", "video")); !strings.Contains(errText, "inauguração já foi") {
		t.Fatalf("depois da inauguração: %q", errText)
	}
}

func TestPecaRefuses(t *testing.T) {
	box := map[string]any{"link": "https://www.collectatcg.com.br/selado/reinado-box"}
	cases := []struct {
		name string
		args func() map[string]any
		want string
	}{
		{"modelo", func() map[string]any { return pecaArgs("story", "video", box) }, "não existe"},
		{"travessão", func() map[string]any {
			a := pecaArgs("produto", "video", box)
			a["gancho"] = "Chegou — corre"
			return a
		}, "travessão"},
		{"gancho longo", func() map[string]any {
			a := pecaArgs("produto", "video", box)
			a["gancho"] = strings.Repeat("a", 41)
			return a
		}, "gancho com 41 letras"},
		{"carta com dois", func() map[string]any { return pecaArgs("carta", "video", box, box) }, "exatamente 1"},
		{"sem produto", func() map[string]any { return pecaArgs("produto", "video") }, "de 1 a 3"},
		{"site de fora", func() map[string]any {
			return pecaArgs("produto", "video", map[string]any{"link": "https://www.ligapokemon.com.br/selado/reinado-box"})
		}, "não é de um produto do site"},
		{"fora do catálogo", func() map[string]any {
			return pecaArgs("produto", "video", map[string]any{"link": "https://www.collectatcg.com.br/selado/nao-existe"})
		}, "não está no catálogo"},
		{"sem estoque", func() map[string]any {
			return pecaArgs("produto", "video", map[string]any{"link": "https://www.collectatcg.com.br/carta/charizard"})
		}, "sem estoque"},
		{"nome do site longo", func() map[string]any { return pecaArgs("produto", "video", box) }, "nome curto"},
		{"imagem fora da pasta", func() map[string]any {
			return pecaArgs("produto", "video", map[string]any{"link": box["link"], "nome": "Box", "imagem": "../../segredo.png"})
		}, "inválida"},
		{"imagem que não existe", func() map[string]any {
			return pecaArgs("produto", "video", map[string]any{"link": box["link"], "nome": "Box", "imagem": "products/nada.png"})
		}, "não existe"},
		{"muitas linhas", func() map[string]any {
			a := pecaArgs("produto", "video", box)
			a["final"] = map[string]any{"titulo": "Oi", "oferta": "x", "linhas": []string{"a", "b", "c", "d"}, "cta": "Link"}
			return a
		}, "de 1 a 3 linhas"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, site := newPecaFixture(t)
			_, errText := callTool[pecaOutput](t, f, "criar_peca", c.args())
			if !strings.Contains(errText, c.want) {
				t.Fatalf("erro = %q, queria %q", errText, c.want)
			}
			if len(site.renders) != 0 {
				t.Fatal("renderizou mesmo recusando")
			}
		})
	}
}

func TestPecaDailyLimit(t *testing.T) {
	f, site := newPecaFixture(t)
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
	_, errText := callTool[pecaOutput](t, f, "criar_peca", pecaArgs("contagem", "arte"))
	if !strings.Contains(errText, "limite do dia") || len(site.renders) != 0 {
		t.Fatalf("erro = %q", errText)
	}
}

func TestPecaOneRenderAtATime(t *testing.T) {
	f, _ := newPecaFixture(t)
	f.app.renderMu.Lock()
	defer f.app.renderMu.Unlock()
	_, errText := callTool[pecaOutput](t, f, "criar_peca", pecaArgs("contagem", "video"))
	if !strings.Contains(errText, "outra peça") {
		t.Fatalf("erro = %q", errText)
	}
}

func TestWebpSize(t *testing.T) {
	head := make([]byte, 30)
	copy(head, "RIFF")
	copy(head[8:], "WEBPVP8X")
	head[24], head[25], head[26] = 0x37, 0x04, 0x00
	head[27], head[28], head[29] = 0x7f, 0x07, 0x00
	w, h, err := webpSize(bytes.NewReader(head))
	if err != nil || w != 1080 || h != 1920 {
		t.Fatalf("webp = %dx%d %v", w, h, err)
	}
}
