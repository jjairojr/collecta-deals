package main

// TikTok só com peça que o Jairo já aprovou para o Instagram pelo chat.
// Decisão do Jairo de 24/11/2026, no chat do CEO Collecta, à pergunta "TikTok: libera postar lá
// as mesmas peças aprovadas para o Instagram?": "pod ser".
//
// A trava: pedir_aprovacao com canal tiktok exige do_instagram, o código de um post do Instagram
// (post ou carrossel) aprovado pelo chat (aprovar_pelo_chat). A ferramenta confere que os arquivos
// do post não mudaram, monta o vídeo do TikTok só com eles (scripts/tiktok.mjs: as imagens no
// fundo azul da marca), usa a legenda aprovada sem mudar nada e manda uma vez só. Peça nova que
// iria só ao TikTok não tem como passar.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"opdeals/internal/publishq"
)

type slideshowFunc func(ctx context.Context, output string, images []string) error

func nodeSlideshow(node, motionDir string) slideshowFunc {
	return func(ctx context.Context, output string, images []string) error {
		cmd := exec.CommandContext(ctx, node, append([]string{"scripts/tiktok.mjs", output}, images...)...)
		cmd.Dir = motionDir
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("não consegui montar o vídeo do TikTok: %v %s", err, strings.TrimSpace(stderr.String()))
		}
		return nil
	}
}

func (a *app) findPost(ctx context.Context, id string) (publishq.Item, error) {
	for _, status := range []string{publishq.StatusApproved, publishq.StatusPublished} {
		var list struct {
			Items []publishq.Item `json:"items"`
		}
		if err := a.call(ctx, http.MethodGet, "/api/publish?status="+status, nil, &list); err != nil {
			return publishq.Item{}, err
		}
		for _, it := range list.Items {
			if it.ID == id {
				return it, nil
			}
		}
	}
	return publishq.Item{}, fmt.Errorf("o post %s não está aprovado nem publicado no painel", id)
}

func (a *app) tiktokFromInstagram(ctx context.Context, in requestInput) (requestOutput, error) {
	if _, on := a.chatApprovers(); !on {
		return requestOutput{}, errors.New("TikTok só com peça aprovada pelo Jairo no chat, e a aprovação pelo chat está desligada")
	}
	code := strings.ToUpper(strings.TrimSpace(in.DoInstagram))
	if code == "" {
		return requestOutput{}, errors.New("TikTok só com do_instagram: o código do post do Instagram que o Jairo aprovou pelo chat. Peça que só iria ao TikTok não pode")
	}
	tickets, err := a.loadTickets()
	if err != nil {
		return requestOutput{}, err
	}
	var ticket *chatTicket
	for i := range tickets {
		if tickets[i].Kind == kindPost && tickets[i].Code == code {
			ticket = &tickets[i]
		}
	}
	switch {
	case ticket == nil:
		return requestOutput{}, fmt.Errorf("nenhum post com o código %q passou pelo chat", in.DoInstagram)
	case ticket.Decided != "aprovar":
		return requestOutput{}, fmt.Errorf("o post %s não foi aprovado pelo Jairo no chat", code)
	case ticket.TikTok != "":
		return requestOutput{}, fmt.Errorf("o post %s já foi mandado ao TikTok (%s)", code, ticket.TikTok)
	}
	post, err := a.findPost(ctx, ticket.ID)
	if err != nil {
		return requestOutput{}, err
	}
	if post.Channel != "instagram" || (post.Format != "post" && post.Format != "carrossel") {
		return requestOutput{}, fmt.Errorf("só post ou carrossel do Instagram vai ao TikTok; %s é %s %s", code, post.Channel, post.Format)
	}
	images := make([]string, 0, len(post.Files))
	for _, f := range post.Files {
		got, err := hashFile(f.Path)
		if err != nil {
			return requestOutput{}, err
		}
		if got.SHA256 != f.SHA256 {
			return requestOutput{}, fmt.Errorf("%s mudou depois da aprovação do Jairo: não vai ao TikTok", filepath.Base(f.Path))
		}
		images = append(images, f.Path)
	}
	when, err := a.parseWhen(in.PublicarEm)
	if err != nil {
		return requestOutput{}, err
	}
	note := fmt.Sprintf("mesma peça do Instagram %s, aprovada pelo Jairo no chat", code)
	// Modo foto (decisão do Jairo de 25/11/2026): as mesmas imagens, como post de fotos do TikTok.
	// Se o painel ainda não aceitar o formato foto, cai para o vídeo montado com as mesmas imagens.
	var created publishq.Item
	formato := "foto"
	files, err := a.checkFiles("foto", images)
	if err != nil {
		return requestOutput{}, err
	}
	item := publishq.Item{Channel: "tiktok", Format: "foto", Files: files, Caption: post.Caption, PostAt: when, Note: note}
	if err := a.call(ctx, http.MethodPost, "/api/publish", item, &created); err != nil {
		if !strings.Contains(err.Error(), "respondeu 400") {
			return requestOutput{}, err
		}
		formato = "video"
		output := filepath.Join(a.cfg.MotionDir, "out", "tiktok", a.now().In(saoPaulo).Format("2006-01-02")+"-"+post.ID+".mp4")
		if a.slideshow == nil {
			a.slideshow = nodeSlideshow(a.cfg.Node, a.cfg.MotionDir)
		}
		if err := a.slideshow(ctx, output, images); err != nil {
			return requestOutput{}, err
		}
		vfiles, err := a.checkFiles("video", []string{output})
		if err != nil {
			_ = os.Remove(output)
			return requestOutput{}, err
		}
		item = publishq.Item{Channel: "tiktok", Format: "video", Files: vfiles, Caption: post.Caption, PostAt: when, Note: note}
		if err := a.call(ctx, http.MethodPost, "/api/publish", item, &created); err != nil {
			return requestOutput{}, err
		}
	}
	if err := a.selfApprove(ctx, created); err != nil {
		return requestOutput{}, fmt.Errorf("pedido %s criado, mas não foi aprovado sozinho: %v", created.ID, err)
	}
	ticket.TikTok = created.ID
	if err := a.saveTicket(*ticket); err != nil {
		return requestOutput{}, fmt.Errorf("TikTok %s na fila, mas não gravei no registro do post %s: %v", created.ID, code, err)
	}
	a.logLine(fmt.Sprintf("tiktok %s %s da peça do instagram %s (%s), aprovada no chat: %q", formato, created.ID, code, post.ID, ticket.Phrase))
	return requestOutput{
		ID: created.ID, Quando: when.In(saoPaulo).Format("02/01 15:04"), Arquivos: len(item.Files),
		Proximo: "TikTok na fila, já aprovado porque é a mesma peça que o Jairo aprovou para o Instagram: o Publicador posta no horário. Anote o id na seção Peças do seu arquivo do dia e não mande de novo.",
	}, nil
}
