package api

import (
	"net/http"
	"time"

	"opdeals/internal/conversations"
	"opdeals/internal/orders"
	"opdeals/internal/postsale"
)

type pendingWork struct {
	Quiet    bool      `json:"quiet"`
	Compose  int       `json:"compose"`
	Outbox   int       `json:"outbox"`
	Tracking int       `json:"tracking"`
	Open     int       `json:"open"`
	OldestAt time.Time `json:"oldestAt,omitzero"`
}

func (p pendingWork) Total() int {
	return p.Compose + p.Outbox + p.Tracking
}

func (p *pendingWork) waitingSince(at time.Time) {
	if p.OldestAt.IsZero() || at.Before(p.OldestAt) {
		p.OldestAt = at
	}
}

func pendingWorkOf(all []orders.Order, convs []conversations.Conversation, now time.Time) pendingWork {
	byID := make(map[string]orders.Order, len(all))
	for _, o := range all {
		byID[o.ID] = o
	}
	p := pendingWork{Quiet: postsale.Quiet(now, saoPaulo)}
	for _, c := range convs {
		switch {
		case c.State == conversations.StateCompose && !p.Quiet && unsentStale(c, byID, now) == "":
			p.Compose++
			p.waitingSince(postsale.NextAllowed(c.CreatedAt, saoPaulo))
		case (c.State == conversations.StateWait || c.State == conversations.StateActive) && !c.SentAt.IsZero() && now.Before(c.WindowEndsAt):
			p.Open++
		}
		for _, m := range c.Messages {
			since, ok := outboxPendingSince(c, m, byID, p.Quiet, now)
			if !ok {
				continue
			}
			p.Outbox++
			p.waitingSince(since)
		}
	}
	if !p.Quiet {
		for _, o := range trackingDue(all, now) {
			since := o.ShippedAt
			if !o.TrackingNotifiedAt.IsZero() {
				since = o.TrackingNotifiedAt.Add(trackingRetry)
			}
			p.Tracking++
			p.waitingSince(postsale.NextAllowed(since, saoPaulo))
		}
	}
	return p
}

func outboxPendingSince(c conversations.Conversation, m conversations.Message, byID map[string]orders.Order, quiet bool, now time.Time) (time.Time, bool) {
	if m.Status != conversations.MsgQueued {
		return time.Time{}, false
	}
	if !m.ClaimedAt.IsZero() && now.Sub(m.ClaimedAt) < outboxLease {
		return time.Time{}, false
	}
	since := m.At
	if !m.ClaimedAt.IsZero() {
		since = m.ClaimedAt.Add(outboxLease)
	}
	switch m.Kind {
	case conversations.KindOffer:
		if quiet || c.State != conversations.StateQueued || !orderStillPaid(c, byID, now) {
			return time.Time{}, false
		}
		return postsale.NextAllowed(since, saoPaulo), true
	case conversations.KindReply:
		if now.Sub(m.At) > queuedReplyTTL {
			return time.Time{}, false
		}
	}
	return since, true
}

func (s *Server) pendingWorkNow() (pendingWork, error) {
	all, err := s.Checkout.Orders.List()
	if err != nil {
		return pendingWork{}, err
	}
	convs, err := s.PostSale.Conversations.List()
	if err != nil {
		return pendingWork{}, err
	}
	return pendingWorkOf(all, convs, s.psNow()), nil
}

func (s *Server) handleAgentPending(w http.ResponseWriter, _ *http.Request) {
	p, err := s.pendingWorkNow()
	if err != nil {
		agentError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, p)
}
