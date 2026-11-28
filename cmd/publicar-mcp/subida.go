package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"opdeals/internal/adtestq"
)

// Subida idempotente (28/11/2026): o teste 0ab7fc2cbdfff8e2 subiu duas vezes porque
// o painel respondeu 502 ao /started depois que a campanha já estava no ar no Meta;
// o teste ficou "aprovado" e a rodada seguinte criou outra campanha. Agora:
//   - antes de criar, o executor procura no Meta a campanha do teste pelo nome
//     ("Collecta teste <id> ..."); se ela está completa, reaproveita; se sobrou uma
//     incompleta de uma tentativa que falhou, pausa a sobra e cria de novo;
//   - o /started é tentado de novo algumas vezes; se o painel continuar fora, a
//     campanha fica pausada até a próxima rodada registrar, sem gastar sem registro.

const startedTries = 3

func campaignName(t adtestq.Test) string {
	return fmt.Sprintf("Collecta teste %s %s", t.ID, t.Objective)
}

type metaObject struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	EffectiveStatus string `json:"effective_status"`
	EndTime         string `json:"end_time"`
}

func (e *executor) children(ctx context.Context, parent, edge, fields string) ([]metaObject, error) {
	var list struct {
		Data []metaObject `json:"data"`
	}
	if err := e.g.get(ctx, parent+"/"+edge, url.Values{"fields": {fields}, "limit": {"50"}}, &list); err != nil {
		return nil, err
	}
	return list.Data, nil
}

// recoverLaunched procura no Meta campanhas deste teste que sobraram de uma
// subida anterior. Devolve a completa (com conjunto e anúncios) para reaproveitar
// e os ids das incompletas, que não devem ficar ativas.
func (e *executor) recoverLaunched(ctx context.Context, t adtestq.Test) (*adtestq.Meta, time.Time, []string, error) {
	var list struct {
		Data []metaObject `json:"data"`
	}
	filter := jsonParam([]map[string]string{{"field": "name", "operator": "CONTAIN", "value": t.ID}})
	if err := e.g.get(ctx, e.g.cfg.Account+"/campaigns", url.Values{"fields": {"id,name,effective_status"}, "filtering": {filter}, "limit": {"50"}}, &list); err != nil {
		return nil, time.Time{}, nil, fmt.Errorf("não consegui conferir se o teste já tem campanha no Meta: %w", err)
	}
	var found *adtestq.Meta
	var endsAt time.Time
	var leftovers []string
	for _, c := range list.Data {
		if c.Name != campaignName(t) || c.EffectiveStatus == "DELETED" || c.EffectiveStatus == "ARCHIVED" {
			continue
		}
		meta, end, ok, err := e.completeCampaign(ctx, t, c.ID)
		if err != nil {
			return nil, time.Time{}, nil, err
		}
		if ok && found == nil {
			found, endsAt = meta, end
			continue
		}
		leftovers = append(leftovers, c.ID)
	}
	return found, endsAt, leftovers, nil
}

// completeCampaign diz se a campanha tem um conjunto com data de fim e ao menos o
// mínimo de anúncios do teste, e remonta os ids como o painel guarda.
func (e *executor) completeCampaign(ctx context.Context, t adtestq.Test, campaignID string) (*adtestq.Meta, time.Time, bool, error) {
	sets, err := e.children(ctx, campaignID, "adsets", "id,effective_status,end_time")
	if err != nil {
		return nil, time.Time{}, false, err
	}
	if len(sets) != 1 {
		return nil, time.Time{}, false, nil
	}
	end, err := time.Parse("2006-01-02T15:04:05-0700", sets[0].EndTime)
	if err != nil {
		return nil, time.Time{}, false, nil
	}
	ads, err := e.children(ctx, sets[0].ID, "ads", "id,name,effective_status")
	if err != nil {
		return nil, time.Time{}, false, err
	}
	meta := &adtestq.Meta{CampaignID: campaignID, AdSetID: sets[0].ID}
	prefix := t.ID + " "
	for _, ad := range ads {
		rest, ok := strings.CutPrefix(ad.Name, prefix)
		if !ok || ad.EffectiveStatus == "DELETED" || ad.EffectiveStatus == "ARCHIVED" {
			continue
		}
		n, err := strconv.Atoi(strings.SplitN(rest, " ", 2)[0])
		if err != nil || n < 1 || n > len(t.Creatives) {
			continue
		}
		meta.Ads = append(meta.Ads, adtestq.MetaAd{ID: ad.ID, Creative: n - 1})
	}
	if len(meta.Ads) < adtestq.MinCreatives {
		return nil, time.Time{}, false, nil
	}
	return meta, end, true, nil
}

func (e *executor) setStatus(ctx context.Context, meta adtestq.Meta, status string) error {
	ids := make([]string, 0, len(meta.Ads)+2)
	for _, ad := range meta.Ads {
		ids = append(ids, ad.ID)
	}
	for _, id := range append(ids, meta.AdSetID, meta.CampaignID) {
		if err := e.g.post(ctx, id, url.Values{"status": {status}}, nil); err != nil {
			return err
		}
	}
	return nil
}

// markStarted registra no painel que o teste está no ar, tentando de novo se o
// painel falhar. Sem registro, a campanha é pausada: a próxima rodada a acha pelo
// nome, reativa e registra, em vez de criar outra.
func (e *executor) markStarted(ctx context.Context, t adtestq.Test, meta adtestq.Meta, endsAt time.Time) error {
	var err error
	for try := 0; try < startedTries; try++ {
		if try > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(e.startedWait * time.Duration(try)):
			}
		}
		if err = e.app.call(ctx, http.MethodPost, "/api/adtests/"+url.PathEscape(t.ID)+"/started", map[string]any{"meta": meta, "endsAt": endsAt}, nil); err == nil {
			return nil
		}
	}
	if perr := e.g.post(ctx, meta.CampaignID, url.Values{"status": {"PAUSED"}}, nil); perr != nil {
		err = errors.Join(err, fmt.Errorf("e a campanha %s não pausou: %v", meta.CampaignID, perr))
	} else {
		e.app.adLog(fmt.Sprintf("painel não registrou %s: campanha %s pausada até a próxima rodada registrar", t.ID, meta.CampaignID))
	}
	return err
}
