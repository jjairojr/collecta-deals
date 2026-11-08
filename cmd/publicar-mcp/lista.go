package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	listaDefault   = 5
	listaMin       = 3
	releasesMax    = 4
	headlinesMax   = 3
	headlineMaxAge = 7 * 24 * time.Hour
	cardTitleMax   = 32
	chaseDefault   = 4
	chaseFloorUSD  = 5
)

var (
	listaTipos = map[string]bool{"lancamentos": true, "subiram": true, "em_alta": true, "noticias": true, "podem_vir": true}
	listaKinds = map[string]string{"video": "lista", "arte": "lista-arte"}
	chaseHook  = regexp.MustCompile(`(?i)\bpodem?\s+(vir|sair)\b`)
	pulledWord = regexp.MustCompile(`(?i)\b(tiramos|tirei|tirou|abrimos|abri|puxamos|puxei|puxou|saiu pra|saíram pra|nosso pull|pegamos)\b`)
	months     = []string{"JANEIRO", "FEVEREIRO", "MARÇO", "ABRIL", "MAIO", "JUNHO", "JULHO", "AGOSTO", "SETEMBRO", "OUTUBRO", "NOVEMBRO", "DEZEMBRO"}
)

type listaManchete struct {
	Texto string `json:"texto" jsonschema:"a manchete em pt-BR, de 10 a 90 letras, sem travessão, fiel à fonte"`
	Link  string `json:"link" jsonschema:"link https da notícia original (a ferramenta abre o link para conferir)"`
	Data  string `json:"data" jsonschema:"data da notícia, AAAA-MM-DD, no máximo 7 dias atrás"`
	Set   string `json:"set,omitempty" jsonschema:"opcional: sigla ou nome do set no TCGplayer para a foto (ex.: OP18, Delta Reign). Sem ele vale o set mais recente do jogo"`
}

type listaInput struct {
	Tipo      string            `json:"tipo" jsonschema:"lancamentos (sets do mês, data do TCGplayer EUA), subiram (cartas que mais subiram na Liga em 7 dias), em_alta (cartas mais vendidas na Liga em 7 dias), noticias (1 a 3 manchetes com link) ou podem_vir (cartas mais caras que podem vir no set, preço de mercado do TCGplayer EUA, e no fim o produto do site com o nosso preço; para anúncio)"`
	Jogo      string            `json:"jogo" jsonschema:"pokemon, onepiece, riftbound, magic, yugioh ou lorcana. subiram e em_alta: só pokemon, onepiece e riftbound"`
	Formato   string            `json:"formato" jsonschema:"video (Reels e TikTok, 1080x1920) ou arte (post do Instagram, 1080x1350)"`
	Mes       string            `json:"mes,omitempty" jsonschema:"só em lancamentos: AAAA-MM (padrão: o mês de hoje)"`
	Set       string            `json:"set,omitempty" jsonschema:"só em podem_vir: sigla ou nome do set no TCGplayer (ex.: DLR ou Delta Reign, OP18)"`
	Itens     int               `json:"itens,omitempty" jsonschema:"só em subiram, em_alta e podem_vir: quantas cartas, de 3 a 5 (padrão 5; em podem_vir padrão 4)"`
	Etiqueta  string            `json:"etiqueta" jsonschema:"linha pequena no topo, até 32 letras (ex.: One Piece · Liga)"`
	Gancho    string            `json:"gancho" jsonschema:"frase grande dos primeiros segundos, de 4 a 40 letras, sem travessão (ex.: AS CARTAS QUE MAIS SUBIRAM)"`
	Destaque  []string          `json:"destaque,omitempty" jsonschema:"palavras do gancho pintadas de verde, até 4"`
	Manchetes []listaManchete   `json:"manchetes,omitempty" jsonschema:"só em noticias: de 1 a 3 manchetes"`
	Produto   *pecaProdutoInput `json:"produto,omitempty" jsonschema:"lancamentos (opcional) e podem_vir (obrigatório): produto do site; entra no fim com o preço do site no Pix e no cartão"`
	Final     pecaFinalInput    `json:"final"`
}

type listaItem struct {
	Titulo    string       `json:"titulo"`
	Imagem    string       `json:"imagem"`
	Proporcao float64      `json:"proporcao"`
	Rotulo    string       `json:"rotulo"`
	Detalhe   string       `json:"detalhe"`
	Antes     float64      `json:"antes,omitempty"`
	Agora     float64      `json:"agora,omitempty"`
	Alta      int          `json:"alta,omitempty"`
	Unidades  int          `json:"unidades,omitempty"`
	Dolar     float64      `json:"dolar,omitempty"`
	Produto   *pecaProduto `json:"produto,omitempty"`
}

type listaProps struct {
	Music    bool        `json:"music"`
	Tipo     string      `json:"tipo"`
	Etiqueta string      `json:"etiqueta"`
	Gancho   string      `json:"gancho"`
	Destaque []string    `json:"destaque"`
	Selo     string      `json:"selo"`
	Fonte    string      `json:"fonte"`
	Itens    []listaItem `json:"itens"`
	Final    pecaFinal   `json:"final"`
}

type listaSaida struct {
	Titulo   string  `json:"titulo"`
	Rotulo   string  `json:"rotulo"`
	Detalhe  string  `json:"detalhe"`
	Antes    float64 `json:"preco_antes,omitempty"`
	Agora    float64 `json:"preco_agora,omitempty"`
	Alta     int     `json:"alta_pct,omitempty"`
	Unidades int     `json:"unidades,omitempty"`
	Dolar    float64 `json:"preco_mercado_usd,omitempty"`
	Link     string  `json:"link,omitempty"`
}

type listaOutput struct {
	Arquivo  string       `json:"arquivo"`
	Previas  []string     `json:"previas,omitempty"`
	Segundos float64      `json:"segundos,omitempty"`
	Fonte    string       `json:"fonte"`
	Itens    []listaSaida `json:"itens"`
	Precos   []pecaPreco  `json:"precos,omitempty"`
	Avisos   []string     `json:"avisos,omitempty"`
	Proximo  string       `json:"proximo"`
}

type listaBuild struct {
	props  listaProps
	out    listaOutput
	photos []string
}

func (a *app) addListaTool(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "criar_lista",
		Description: "Só para o Social Collecta. Gera vídeo ou arte de conteúdo com dados reais que a própria ferramenta busca: lancamentos (sets do mês no TCGplayer), subiram (cartas que mais subiram na Liga em 7 dias), em_alta (mais vendidas na Liga em 7 dias) noticias (manchetes com link) ou podem_vir (anúncio: as cartas mais caras que podem vir no set, com preço de mercado do TCGplayer EUA, e o produto do site com o nosso preço). Você escreve só os textos; datas, preços e vendas vêm da fonte e voltam em itens. Olhe as prévias com Read antes de pedir aprovação. Conta no limite de 30 peças por dia junto com criar_peca.",
	}, a.toolLista)
}

func (a *app) toolLista(ctx context.Context, _ *mcp.CallToolRequest, in listaInput) (*mcp.CallToolResult, listaOutput, error) {
	tipo := strings.ToLower(strings.TrimSpace(in.Tipo))
	kind, ok := listaKinds[strings.ToLower(strings.TrimSpace(in.Formato))]
	ext := ".mp4"
	if kind == "lista-arte" {
		ext = ".png"
	}
	if !listaTipos[tipo] || !ok {
		return nil, listaOutput{}, fmt.Errorf("tipo %q ou formato %q não existe: tipo lancamentos, subiram, em_alta, noticias ou podem_vir; formato video ou arte", in.Tipo, in.Formato)
	}
	jogo := strings.ToLower(strings.TrimSpace(in.Jogo))
	game, ok := listaGames[jogo]
	if !ok {
		return nil, listaOutput{}, fmt.Errorf("jogo %q não existe: pokemon, onepiece, riftbound, magic, yugioh ou lorcana", in.Jogo)
	}
	textIn := pecaInput{Etiqueta: in.Etiqueta, Gancho: in.Gancho, Destaque: in.Destaque, Final: in.Final}
	if in.Produto != nil {
		textIn.Produtos = []pecaProdutoInput{*in.Produto}
	}
	texts, err := a.pecaTexts("lista", textIn)
	if err != nil {
		return nil, listaOutput{}, err
	}
	if err := checkListaShape(tipo, in); err != nil {
		return nil, listaOutput{}, err
	}
	if !a.renderMu.TryLock() {
		return nil, listaOutput{}, errors.New("outra peça está sendo gerada agora: tente de novo em 1 minuto")
	}
	defer a.renderMu.Unlock()
	if n := a.rendersToday(); n >= rendersPerDay {
		return nil, listaOutput{}, fmt.Errorf("já foram %d peças hoje, o limite do dia: use as que já existem", n)
	}
	id := randomID()
	a.pruneStaging()
	var b listaBuild
	switch tipo {
	case "lancamentos":
		b, err = a.buildReleases(ctx, id, game, in)
	case "noticias":
		b, err = a.buildNews(ctx, id, game, in.Manchetes)
	case "podem_vir":
		b, err = a.buildChase(ctx, id, game, in)
	default:
		b, err = a.buildLiga(ctx, id, game, tipo, in.Itens)
	}
	if err != nil {
		return nil, listaOutput{}, err
	}
	b.props.Music = true
	b.props.Tipo = tipo
	b.props.Etiqueta = texts.Etiqueta
	b.props.Gancho = texts.Gancho
	b.props.Destaque = texts.Destaque
	b.props.Final = texts.Final
	name := fmt.Sprintf("%s-lista-%s-%s-%s%s", a.now().In(saoPaulo).Format("2006-01-02"), strings.ReplaceAll(tipo, "_", "-"), jogo, id, ext)
	output := filepath.Join(a.cfg.MotionDir, "out", "agente", name)
	propsPath := filepath.Join(a.cfg.DataDir, "pecas", id+".json")
	if err := writeJSON(propsPath, b.props); err != nil {
		return nil, listaOutput{}, err
	}
	rctx, cancel := context.WithTimeout(ctx, renderTimeout)
	defer cancel()
	res, err := a.render(rctx, propsPath, kind, output)
	if err != nil {
		a.pecaLog(fmt.Sprintf("falhou %s lista-%s %s: %v", id, tipo, kind, err))
		return nil, listaOutput{}, fmt.Errorf("a peça não renderizou: %v", err)
	}
	a.pecaLog(fmt.Sprintf("gerada %s lista-%s %s %s", id, tipo, kind, output))
	out := b.out
	out.Arquivo = output
	out.Previas = res.Previas
	out.Segundos = res.Segundos
	out.Fonte = b.props.Fonte
	if kind == "lista-arte" {
		out.Previas = []string{output}
	}
	out.Proximo = "olhe as prévias com Read. Se estiver boa, peça aprovação com pedir_aprovacao usando este arquivo; a legenda usa só os números e datas de itens e cita a fonte. Se não, gere de novo com textos melhores."
	return nil, out, nil
}

func checkListaShape(tipo string, in listaInput) error {
	switch {
	case tipo != "noticias" && len(in.Manchetes) > 0:
		return errors.New("manchetes só valem no tipo noticias")
	case tipo == "noticias" && (len(in.Manchetes) < 1 || len(in.Manchetes) > headlinesMax):
		return fmt.Errorf("noticias leva de 1 a %d manchetes", headlinesMax)
	case tipo != "lancamentos" && strings.TrimSpace(in.Mes) != "":
		return errors.New("mes só vale no tipo lancamentos")
	case tipo != "lancamentos" && tipo != "podem_vir" && in.Produto != nil:
		return errors.New("produto só vale em lancamentos e podem_vir")
	case tipo != "podem_vir" && strings.TrimSpace(in.Set) != "":
		return errors.New("set só vale em podem_vir")
	case tipo == "podem_vir" && (in.Produto == nil || strings.TrimSpace(in.Set) == ""):
		return errors.New("podem_vir precisa de set (sigla do TCGplayer, ex.: DLR) e produto (link do site)")
	case tipo == "podem_vir" && !chaseHook.MatchString(in.Gancho):
		return errors.New(`em podem_vir o gancho tem que dizer "pode vir" ou "podem vir" (ex.: AS MAIS CARAS QUE PODEM VIR): nunca dê a entender que a carta já saiu pra gente`)
	case pulledWord.MatchString(in.Gancho + " " + in.Etiqueta + " " + in.Final.Titulo + " " + in.Final.Oferta):
		return errors.New("o texto dá a entender que a gente abriu ou tirou a carta: diga que ela pode vir")
	case (tipo == "subiram" || tipo == "em_alta" || tipo == "podem_vir") && in.Itens != 0 && (in.Itens < listaMin || in.Itens > listaDefault):
		return fmt.Errorf("itens vai de %d a %d", listaMin, listaDefault)
	case tipo != "subiram" && tipo != "em_alta" && tipo != "podem_vir" && in.Itens != 0:
		return errors.New("itens só vale em subiram, em_alta e podem_vir")
	}
	for _, m := range in.Manchetes {
		if err := (textLimit{"manchetes.texto", cleanLine(m.Texto), 10, 90}).check(); err != nil {
			return err
		}
	}
	return nil
}

func (a *app) listaPhoto(ctx context.Context, id string, b *listaBuild, raw string) (string, float64, error) {
	rel, ratio, err := a.downloadPhoto(ctx, id, len(b.photos), raw)
	if err != nil {
		return "", 0, err
	}
	b.photos = append(b.photos, rel)
	return rel, ratio, nil
}

func (a *app) listaMonth(raw string) (time.Time, error) {
	today := a.now().In(saoPaulo)
	if raw = strings.TrimSpace(raw); raw == "" {
		return time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, saoPaulo), nil
	}
	month, err := time.ParseInLocation("2006-01", raw, saoPaulo)
	if err != nil {
		return time.Time{}, fmt.Errorf("mes inválido %q: use AAAA-MM", raw)
	}
	return month, nil
}

func (a *app) buildReleases(ctx context.Context, id string, game listaGame, in listaInput) (listaBuild, error) {
	month, err := a.listaMonth(in.Mes)
	if err != nil {
		return listaBuild{}, err
	}
	end := month.AddDate(0, 1, 0)
	all, err := a.releases(ctx, game)
	if err != nil {
		return listaBuild{}, err
	}
	b := listaBuild{props: listaProps{Selo: months[month.Month()-1], Fonte: "Datas de lançamento nos EUA: TCGplayer"}}
	today := a.now().In(saoPaulo)
	type pick struct {
		release tcgRelease
		detail  string
	}
	picks := []pick{}
	for _, r := range all {
		if !r.Date.Before(month) && r.Date.Before(end) && len(picks) < releasesMax {
			detail := "Lançamento EUA"
			if !r.Date.After(today) {
				detail = "Já saiu nos EUA"
			}
			picks = append(picks, pick{r, detail})
		}
	}
	if len(picks) == 0 {
		var last, next *tcgRelease
		for i := range all {
			if all[i].Date.Before(month) {
				last = &all[i]
			}
			if !all[i].Date.Before(end) && next == nil {
				next = &all[i]
			}
		}
		if last != nil {
			picks = append(picks, pick{*last, "Último lançamento"})
		}
		if next != nil {
			picks = append(picks, pick{*next, "Próximo lançamento"})
		}
		if len(picks) == 0 {
			return listaBuild{}, fmt.Errorf("o TCGplayer não tem sets de %s para mostrar", game.name)
		}
		b.out.Avisos = append(b.out.Avisos, fmt.Sprintf("%s não tem set novo em %s no TCGplayer: o vídeo mostra o último e o próximo. O gancho e a legenda precisam dizer isso, sem prometer lançamento no mês", game.name, strings.ToLower(b.props.Selo)))
	}
	for _, p := range picks {
		img, err := a.boxImage(ctx, game, p.release.Group)
		if err != nil {
			b.out.Avisos = append(b.out.Avisos, fmt.Sprintf("%s ficou de fora: %v", p.release.Group.Name, err))
			continue
		}
		rel, ratio, err := a.listaPhoto(ctx, id, &b, img)
		if err != nil {
			b.out.Avisos = append(b.out.Avisos, fmt.Sprintf("%s ficou de fora: %v", p.release.Group.Name, err))
			continue
		}
		item := listaItem{Titulo: cleanLine(p.release.Group.Name), Imagem: rel, Proporcao: ratio, Rotulo: releaseLabel(p.release.Date, today), Detalhe: p.detail}
		b.props.Itens = append(b.props.Itens, item)
		b.out.Itens = append(b.out.Itens, listaSaida{Titulo: item.Titulo, Rotulo: item.Rotulo, Detalhe: item.Detalhe})
	}
	if len(b.props.Itens) == 0 {
		return listaBuild{}, fmt.Errorf("nenhum set de %s tem foto no TCGplayer", game.name)
	}
	if in.Produto != nil {
		prod, price, err := a.resolveProduct(ctx, id, len(b.photos), *in.Produto)
		if err != nil {
			return listaBuild{}, err
		}
		b.props.Itens = append(b.props.Itens, listaItem{Titulo: prod.Nome, Imagem: prod.Imagem, Proporcao: prod.Proporcao, Rotulo: "Na Collecta", Detalhe: "Pix ou cartão", Produto: &prod})
		b.out.Itens = append(b.out.Itens, listaSaida{Titulo: prod.Nome, Rotulo: "Na Collecta", Agora: prod.Preco, Link: price.Link})
		b.out.Precos = append(b.out.Precos, price)
	}
	return b, nil
}

func releaseLabel(date, today time.Time) string {
	if date.Year() != today.Year() {
		return date.Format("02/01/06")
	}
	return date.Format("02/01")
}

func (a *app) buildLiga(ctx context.Context, id string, game listaGame, tipo string, want int) (listaBuild, error) {
	if want == 0 {
		want = listaDefault
	}
	report, err := a.ligaReport(game, tipo)
	if err != nil {
		return listaBuild{}, err
	}
	window := fmt.Sprintf("%s a %s", report.From.Format("02/01"), report.To.Format("02/01"))
	b := listaBuild{props: listaProps{Selo: "LIGA · " + window, Fonte: "Menor preço na Liga, " + window}}
	if tipo == "em_alta" {
		b.props.Fonte = "Vendas na Liga pela baixa de estoque, " + window
	}
	for _, c := range report.Cards {
		if len(b.props.Itens) == want {
			break
		}
		if utf8.RuneCountInString(c.Name) > cardTitleMax || c.Name == "" {
			continue
		}
		rel, ratio, err := a.listaPhoto(ctx, id, &b, c.Image)
		if err != nil {
			continue
		}
		item := listaItem{Titulo: c.Name, Imagem: rel, Proporcao: ratio, Rotulo: fmt.Sprintf("#%d", len(b.props.Itens)+1), Detalhe: cardDetail(c)}
		saida := listaSaida{Titulo: c.Name, Rotulo: item.Rotulo, Detalhe: item.Detalhe}
		if tipo == "subiram" {
			item.Antes, item.Agora, item.Alta = c.Before, c.Now, int(c.Change)
			saida.Antes, saida.Agora, saida.Alta = c.Before, c.Now, int(c.Change)
		} else {
			item.Agora, item.Unidades = c.Average, c.Units
			saida.Agora, saida.Unidades = c.Average, c.Units
		}
		b.props.Itens = append(b.props.Itens, item)
		b.out.Itens = append(b.out.Itens, saida)
	}
	if len(b.props.Itens) < listaMin {
		return listaBuild{}, fmt.Errorf("só %d cartas de %s passaram nos filtros da Liga (preço, lojas e foto): escolha outro jogo ou formato", len(b.props.Itens), game.name)
	}
	return b, nil
}

func (a *app) buildChase(ctx context.Context, id string, game listaGame, in listaInput) (listaBuild, error) {
	want := in.Itens
	if want == 0 {
		want = chaseDefault
	}
	today := a.now().In(saoPaulo)
	day := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, saoPaulo)
	all, err := a.releases(ctx, game)
	if err != nil {
		return listaBuild{}, err
	}
	release, ok := pickRelease(all, in.Set, day)
	if !ok {
		return listaBuild{}, fmt.Errorf("o TCGplayer não tem set de %s com %q: sem fonte de preço de mercado, esse produto fica sem podem_vir", game.name, in.Set)
	}
	cards, presale, err := a.chaseCards(ctx, game, release.Group)
	if err != nil {
		return listaBuild{}, err
	}
	stamp := today.Format("02/01")
	b := listaBuild{props: listaProps{Selo: "MERCADO EUA · " + stamp, Fonte: "Preço de mercado nos EUA: TCGplayer, " + stamp}}
	for _, c := range cards {
		if len(b.props.Itens) == want {
			break
		}
		if c.Market < chaseFloorUSD || c.Title == "" || utf8.RuneCountInString(c.Title) > cardTitleMax {
			continue
		}
		rel, ratio, err := a.listaPhoto(ctx, id, &b, c.Image)
		if err != nil {
			continue
		}
		market := math.Round(c.Market*100) / 100
		item := listaItem{Titulo: c.Title, Imagem: rel, Proporcao: ratio, Rotulo: fmt.Sprintf("#%d", len(b.props.Itens)+1), Detalhe: c.Detail, Dolar: market}
		b.props.Itens = append(b.props.Itens, item)
		b.out.Itens = append(b.out.Itens, listaSaida{Titulo: item.Titulo, Rotulo: item.Rotulo, Detalhe: item.Detalhe, Dolar: market})
	}
	if len(b.props.Itens) < listaMin {
		if presale > 0 {
			return listaBuild{}, fmt.Errorf("%s ainda está em pré-venda no TCGplayer (lançamento %s): as cartas ainda não têm preço de mercado. Tente depois do lançamento", cleanLine(release.Group.Name), release.Date.Format("02/01"))
		}
		return listaBuild{}, fmt.Errorf("só %d cartas de %s têm preço de mercado de pelo menos US$ %d e foto no TCGplayer", len(b.props.Itens), cleanLine(release.Group.Name), chaseFloorUSD)
	}
	prod, price, err := a.resolveProduct(ctx, id, len(b.photos), *in.Produto)
	if err != nil {
		return listaBuild{}, err
	}
	b.props.Itens = append(b.props.Itens, listaItem{Titulo: prod.Nome, Imagem: prod.Imagem, Proporcao: prod.Proporcao, Rotulo: "Na Collecta", Detalhe: "Pix ou cartão", Produto: &prod})
	b.out.Itens = append(b.out.Itens, listaSaida{Titulo: prod.Nome, Rotulo: "Na Collecta", Agora: prod.Preco, Link: price.Link})
	b.out.Precos = append(b.out.Precos, price)
	b.out.Avisos = append(b.out.Avisos, fmt.Sprintf("preços em dólar do mercado dos EUA (TCGplayer, %s), não são preço da Collecta: a legenda diz isso e cita a fonte. Diga sempre que as cartas podem vir, nunca que saíram pra gente", stamp))
	return b, nil
}

func cardDetail(c ligaCard) string {
	if strings.Contains(c.Number, "-") {
		return c.Number
	}
	return strings.ToUpper(c.Set) + " · " + c.Number
}

func (a *app) buildNews(ctx context.Context, id string, game listaGame, headlines []listaManchete) (listaBuild, error) {
	today := a.now().In(saoPaulo)
	day := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, saoPaulo)
	b := listaBuild{props: listaProps{Selo: "NOTÍCIAS", Fonte: "Fontes na legenda"}}
	all, err := a.releases(ctx, game)
	if err != nil {
		return listaBuild{}, err
	}
	for _, h := range headlines {
		date, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(h.Data), saoPaulo)
		switch {
		case err != nil:
			return listaBuild{}, fmt.Errorf("data inválida %q: use AAAA-MM-DD", h.Data)
		case date.After(day):
			return listaBuild{}, fmt.Errorf("a data %s ainda não chegou", h.Data)
		case day.Sub(date) > headlineMaxAge:
			return listaBuild{}, fmt.Errorf("a notícia de %s tem mais de 7 dias: notícia velha não entra", date.Format("02/01"))
		}
		host, err := a.checkNewsLink(ctx, h.Link)
		if err != nil {
			return listaBuild{}, err
		}
		release, ok := pickRelease(all, h.Set, day)
		if !ok {
			return listaBuild{}, fmt.Errorf("nenhum set de %s com %q no TCGplayer para a foto", game.name, h.Set)
		}
		img, err := a.boxImage(ctx, game, release.Group)
		if err != nil {
			return listaBuild{}, fmt.Errorf("foto de %s: %v", release.Group.Name, err)
		}
		rel, ratio, err := a.listaPhoto(ctx, id, &b, img)
		if err != nil {
			return listaBuild{}, err
		}
		item := listaItem{Titulo: cleanLine(h.Texto), Imagem: rel, Proporcao: ratio, Rotulo: date.Format("02/01"), Detalhe: "fonte: " + host}
		b.props.Itens = append(b.props.Itens, item)
		b.out.Itens = append(b.out.Itens, listaSaida{Titulo: item.Titulo, Rotulo: item.Rotulo, Detalhe: item.Detalhe, Link: strings.TrimSpace(h.Link)})
	}
	return b, nil
}

func pickRelease(all []tcgRelease, set string, day time.Time) (tcgRelease, bool) {
	set = strings.ToLower(cleanLine(set))
	var latest tcgRelease
	found := false
	for _, r := range all {
		if set != "" {
			if strings.EqualFold(r.Group.Abbreviation, set) || strings.Contains(strings.ToLower(r.Group.Name), set) {
				latest, found = r, true
			}
			continue
		}
		if !r.Date.After(day) {
			latest, found = r, true
		}
	}
	return latest, found
}

func (a *app) checkNewsLink(ctx context.Context, raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return "", fmt.Errorf("link %q inválido: use o https da notícia", raw)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Collecta/1.0")
	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("o link %s não abriu: %v", u.Host, err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone || resp.StatusCode >= 500 {
		return "", fmt.Errorf("o link %s respondeu %d: confira a notícia", u.String(), resp.StatusCode)
	}
	return strings.TrimPrefix(u.Hostname(), "www."), nil
}
