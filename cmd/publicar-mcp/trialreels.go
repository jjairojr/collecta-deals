package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"opdeals/internal/publishq"
)

const trialGraduation = "SS_PERFORMANCE"

func (e *executor) publishTrials(ctx context.Context) []error {
	var list struct {
		Items []publishq.Item `json:"items"`
	}
	if err := e.app.call(ctx, http.MethodGet, "/api/publish?status="+publishq.StatusApproved, nil, &list); err != nil {
		return []error{err}
	}
	var errs []error
	now := e.app.now()
	for _, it := range list.Items {
		if !publishq.ViaAPI(it.Channel, it.Format) || it.PostAt.After(now) {
			continue
		}
		if err := e.publishTrial(ctx, it); err != nil {
			errs = append(errs, fmt.Errorf("reels de teste %s: %w", it.ID, err))
		}
	}
	return errs
}

func (e *executor) publishTrial(ctx context.Context, it publishq.Item) error {
	link, err := e.trialReel(ctx, it)
	if err != nil {
		e.app.adLog(fmt.Sprintf("reels de teste %s falhou: %v", it.ID, err))
		failed, ferr := e.app.fail(ctx, it.ID, err.Error(), false)
		if ferr != nil {
			return errors.Join(err, ferr)
		}
		if failed.Status == publishq.StatusFailed {
			e.notify(ctx, "Reels de teste não saiu", "high", "warning", fmt.Sprintf("%s: %v", filepath.Base(it.Files[0].Path), err))
		}
		return err
	}
	if err := e.app.call(ctx, http.MethodPost, "/api/publish/"+url.PathEscape(it.ID)+"/published", map[string]string{"link": link}, nil); err != nil {
		return fmt.Errorf("saiu em %s, mas não registrou: %w", link, err)
	}
	e.app.logLine(fmt.Sprintf("publicado %s instagram teste %s", it.ID, link))
	e.notify(ctx, "Reels de teste no ar", "default", "clapper", fmt.Sprintf("Só para quem não segue a conta; entra no perfil sozinho se for bem.\n%s", link))
	return nil
}

func (e *executor) trialReel(ctx context.Context, it publishq.Item) (string, error) {
	if len(it.Files) != 1 {
		return "", errors.New("reels de teste leva exatamente 1 vídeo")
	}
	file := it.Files[0]
	got, err := hashFile(file.Path)
	if err != nil {
		return "", err
	}
	if got.SHA256 != file.SHA256 {
		return "", fmt.Errorf("%s mudou depois do pedido", filepath.Base(file.Path))
	}
	var container struct {
		ID string `json:"id"`
	}
	if err := e.g.post(ctx, e.g.cfg.IGUserID+"/media", url.Values{
		"media_type": {"REELS"}, "upload_type": {"resumable"}, "caption": {it.Caption},
		"trial_params": {jsonParam(map[string]string{"graduation_strategy": trialGraduation})},
	}, &container); err != nil {
		return "", err
	}
	if container.ID == "" {
		return "", errors.New("Meta não devolveu o container do vídeo")
	}
	if err := e.rupload(ctx, container.ID, file.Path); err != nil {
		return "", err
	}
	if err := e.waitContainer(ctx, container.ID); err != nil {
		return "", err
	}
	var media struct {
		ID string `json:"id"`
	}
	if err := e.g.post(ctx, e.g.cfg.IGUserID+"/media_publish", url.Values{"creation_id": {container.ID}}, &media); err != nil {
		return "", err
	}
	var info struct {
		Permalink string `json:"permalink"`
	}
	if err := e.g.get(ctx, media.ID, url.Values{"fields": {"permalink"}}, &info); err != nil || info.Permalink == "" {
		return "https://www.instagram.com/" + e.g.cfg.IGUsername + "/", nil
	}
	return info.Permalink, nil
}

func (e *executor) rupload(ctx context.Context, containerID, path string) error {
	fh, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fh.Close()
	info, err := fh.Stat()
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.g.cfg.RuploadURL+"/"+url.PathEscape(containerID), fh)
	if err != nil {
		return err
	}
	req.ContentLength = info.Size()
	req.Header.Set("Authorization", "OAuth "+e.g.cfg.Token)
	req.Header.Set("offset", "0")
	req.Header.Set("file_size", strconv.FormatInt(info.Size(), 10))
	var out struct {
		Success bool `json:"success"`
	}
	if err := e.g.do(req, &out); err != nil {
		return fmt.Errorf("envio do vídeo: %w", err)
	}
	if !out.Success {
		return errors.New("envio do vídeo não confirmado pela Meta")
	}
	return nil
}

func (e *executor) waitContainer(ctx context.Context, id string) error {
	deadline := e.app.now().Add(e.pollMax)
	for {
		var st struct {
			StatusCode string `json:"status_code"`
			Status     string `json:"status"`
		}
		if err := e.g.get(ctx, id, url.Values{"fields": {"status_code,status"}}, &st); err != nil {
			return err
		}
		switch st.StatusCode {
		case "FINISHED":
			return nil
		case "ERROR", "EXPIRED":
			return fmt.Errorf("Instagram não processou o vídeo (%s): %s", st.StatusCode, st.Status)
		}
		if e.app.now().After(deadline) {
			return fmt.Errorf("vídeo ainda processando depois de %s", e.pollMax)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(e.pollEvery):
		}
	}
}
