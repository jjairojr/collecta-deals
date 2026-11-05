package postsale

import (
	"sort"
	"strings"
	"time"
	"unicode"

	"opdeals/internal/conversations"
	"opdeals/internal/orders"
	"opdeals/internal/shipping"
	"opdeals/internal/whatsapp"
)

const (
	OfferDelay      = 15 * time.Minute
	Window          = 12 * time.Hour
	Cooldown        = 7 * 24 * time.Hour
	DraftTTL        = 24 * time.Hour
	EligibleFor     = 24 * time.Hour
	ServiceWindow   = 24 * time.Hour
	QuietStart      = 22
	QuietEnd        = 8
	MinOfferBRL     = 10.0
	MinShare        = 0.3
	MaxShare        = 1.0
	MaxCandidates   = 4
	MaxPitchRunes   = 300
	MaxOpeningRunes = 600
)

type Item struct {
	SKU       string  `json:"sku"`
	Game      string  `json:"game"`
	Kind      string  `json:"kind"`
	Name      string  `json:"name"`
	Set       string  `json:"set"`
	Number    string  `json:"number"`
	Meta      string  `json:"meta"`
	ImageURL  string  `json:"imageURL,omitempty"`
	UnitBRL   float64 `json:"unitBRL"`
	Available int     `json:"available"`
	Featured  bool    `json:"featured,omitempty"`
}

func (it Item) FitsEnvelope() bool {
	return it.Kind == "" || it.Kind == "single"
}

type Candidate struct {
	Item
	Related  int      `json:"related"`
	Velocity float64  `json:"velocity"`
	Reasons  []string `json:"reasons"`
}

func Candidates(bought []orders.Line, itemsBRL float64, stock []Item, velocity map[string]float64) []Candidate {
	floor := max(MinOfferBRL, itemsBRL*MinShare)
	ceil := itemsBRL * MaxShare
	owned := make(map[string]bool, len(bought))
	for _, l := range bought {
		owned[l.SKU] = true
	}
	out := make([]Candidate, 0)
	for _, it := range stock {
		if owned[it.SKU] || !it.FitsEnvelope() || it.Available <= 0 {
			continue
		}
		if it.UnitBRL < floor || it.UnitBRL > ceil {
			continue
		}
		c := Candidate{Item: it, Velocity: velocity[it.SKU]}
		c.Related, c.Reasons = relatedness(it, bought)
		if c.Velocity > 0 {
			c.Reasons = append(c.Reasons, "tá saindo muito")
		}
		if it.Featured {
			c.Reasons = append(c.Reasons, "em destaque na loja")
		}
		out = append(out, c)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Related != b.Related {
			return a.Related > b.Related
		}
		if a.Velocity != b.Velocity {
			return a.Velocity > b.Velocity
		}
		if a.Featured != b.Featured {
			return a.Featured
		}
		return a.UnitBRL > b.UnitBRL
	})
	if len(out) > MaxCandidates {
		out = out[:MaxCandidates]
	}
	return out
}

func relatedness(it Item, bought []orders.Line) (int, []string) {
	score := 0
	var reasons []string
	itWords := Words(it.Name)
	for _, l := range bought {
		if l.Game != it.Game {
			continue
		}
		s := 1
		if set := lineSet(l); set != "" && strings.EqualFold(set, it.Set) {
			s += 2
		}
		if shared := sharedWord(itWords, Words(l.Name)); shared != "" {
			s += 3
		}
		if s > score {
			score = s
			reasons = []string{"combina com " + l.Name}
		}
	}
	return score, reasons
}

func lineSet(l orders.Line) string {
	set, _, _ := strings.Cut(l.Meta, " · ")
	return strings.TrimSpace(set)
}

var stopWords = map[string]bool{
	"the": true, "and": true, "of": true, "de": true, "da": true, "do": true, "dos": true, "das": true,
	"ex": true, "card": true, "carta": true, "parallel": true, "alt": true, "art": true, "full": true,
}

func Words(s string) []string {
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := fields[:0]
	for _, f := range fields {
		if len([]rune(f)) >= 3 && !stopWords[f] {
			out = append(out, f)
		}
	}
	return out
}

func sharedWord(a, b []string) string {
	set := make(map[string]bool, len(b))
	for _, w := range b {
		set[w] = true
	}
	for _, w := range a {
		if set[w] {
			return w
		}
	}
	return ""
}

func Quiet(t time.Time, loc *time.Location) bool {
	h := t.In(loc).Hour()
	return h >= QuietStart || h < QuietEnd
}

func NextAllowed(t time.Time, loc *time.Location) time.Time {
	if !Quiet(t, loc) {
		return t
	}
	local := t.In(loc)
	next := time.Date(local.Year(), local.Month(), local.Day(), QuietEnd, 0, 0, 0, loc)
	if local.Hour() >= QuietStart {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

func EligibleForOffer(o orders.Order, existing []conversations.Conversation, now time.Time) bool {
	if o.Shipping.ServiceID == shipping.JoinServiceID {
		return false
	}
	if orders.EffectiveStatus(o, now) != orders.StatusPaid || o.PaidAt.IsZero() {
		return false
	}
	if now.Sub(o.PaidAt) < OfferDelay || now.Sub(o.PaidAt) > EligibleFor {
		return false
	}
	return eligibleContact(o.ID, o.Buyer.Phone, existing, now)
}

func EligibleLigaOrder(orderID, phone string, at time.Time, existing []conversations.Conversation, now time.Time) bool {
	if at.IsZero() || now.Sub(at) > EligibleFor || at.After(now.Add(time.Hour)) {
		return false
	}
	if len(whatsapp.BrazilE164(phone)) < 12 {
		return false
	}
	return eligibleContact(orderID, phone, existing, now)
}

func eligibleContact(orderID, phone string, existing []conversations.Conversation, now time.Time) bool {
	key := whatsapp.PhoneKey(phone)
	for _, c := range existing {
		if c.OrderID == orderID {
			return false
		}
		if whatsapp.PhoneKey(c.Phone) == key && now.Sub(c.CreatedAt) < Cooldown && c.Outcome != conversations.OutcomeSkipped {
			return false
		}
	}
	return true
}

func FirstName(full string) string {
	if parts := strings.Fields(full); len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func CleanPitch(s string) string {
	s = whatsapp.TemplateSafe(s)
	if r := []rune(s); len(r) > MaxPitchRunes {
		s = strings.TrimSpace(string(r[:MaxPitchRunes]))
	}
	return s
}

func CleanOpening(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.Join(strings.Fields(l), " ")
	}
	s = strings.TrimSpace(strings.Join(lines, "\n"))
	if r := []rune(s); len(r) > MaxOpeningRunes {
		s = strings.TrimSpace(string(r[:MaxOpeningRunes]))
	}
	return s
}
