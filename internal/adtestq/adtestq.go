package adtestq

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

var (
	ErrNotFound     = errors.New("teste de anúncio não encontrado")
	ErrBadToken     = errors.New("código de aprovação inválido")
	ErrNotPending   = errors.New("teste já foi decidido")
	ErrWrongStatus  = errors.New("teste não está na situação certa para isso")
	ErrInvalidInput = errors.New("teste de anúncio inválido")
	ErrOverBudget   = errors.New("passa do teto de verba da semana")
)

const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
	StatusRunning  = "running"
	StatusDone     = "done"
	StatusFailed   = "failed"
)

const (
	WeeklyCapCents = 95000 // vendas, seguidores e remarketing (Jairo, 27/11: dezembro até R$ 900/semana)
	// Mensagens (compra de cartas, clique para WhatsApp) has its own budget,
	// outside the sales one (Jairo, 27/11: "até R$ 1.000/mês"): R$ 250 a week
	// (rolling 7 days) and R$ 1.000 a calendar month in Goiânia.
	MessagesWeeklyCapCents  = 25000
	MessagesMonthlyCapCents = 100000
	MaxBudgetCents          = 30000
	BudgetWindow            = 7 * 24 * time.Hour
	MinBudgetCents          = 3000
	MinDays                 = 3
	MaxDays                 = 7
	MinCreatives            = 2
	MaxCreatives            = 5
)

var (
	objectives = map[string]bool{"seguidores": true, "vendas": true, "mensagens": true}
	audiences  = map[string]bool{"goiania": true, "brasil": true, "remarketing": true, "semelhante": true, "clientes": true, "aberto": true}
	kinds      = map[string]bool{"video": true, "imagem": true, "post": true}
)

// SalesEvents are the site events a vendas test can optimize for.
var SalesEvents = map[string]bool{"compra": true, "checkout": true, "carrinho": true}

type Creative struct {
	Path     string `json:"path"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
	Kind     string `json:"kind"`
	Thumb    string `json:"thumb,omitempty"`
	Text     string `json:"text"`
	Headline string `json:"headline,omitempty"`
	Hook     string `json:"hook"`
	// Link is this creative's own page on the site (vendas only); empty = the
	// test's link. Lets one test send each ad to its own product.
	Link string `json:"link,omitempty"`
	// MediaID is the Instagram media boosted by a "post" creative (an existing
	// post, reel or carousel of the store's account); Permalink is its address.
	// For a post, Path is only the preview image and Text the caption excerpt.
	MediaID   string `json:"mediaId,omitempty"`
	Permalink string `json:"permalink,omitempty"`
}

type MetaAd struct {
	ID       string `json:"id"`
	Creative int    `json:"creative"`
	Paused   bool   `json:"paused,omitempty"`
	Reason   string `json:"reason,omitempty"`
	// Spend and Results are this ad's numbers at the last review, measured in Meta.Metric.
	Spend   float64 `json:"spend,omitempty"`
	Results float64 `json:"results,omitempty"`
}

// Refused is a creative Meta would not run (for a post: licensed music, a
// filter, a deleted post); the test went live without it.
type Refused struct {
	Creative int    `json:"creative"`
	Reason   string `json:"reason"`
}

type Meta struct {
	CampaignID string    `json:"campaignId"`
	AdSetID    string    `json:"adSetId"`
	Ads        []MetaAd  `json:"ads"`
	Metric     string    `json:"metric,omitempty"`
	Refused    []Refused `json:"refused,omitempty"`
}

type Test struct {
	ID        string `json:"id"`
	Objective string `json:"objective"`
	Audience  string `json:"audience"`
	Link      string `json:"link,omitempty"`
	// Event is what a vendas test optimizes for: compra (default), checkout or carrinho.
	Event       string     `json:"event,omitempty"`
	Creatives   []Creative `json:"creatives"`
	BudgetCents int64      `json:"budgetCents"`
	Days        int        `json:"days"`
	Note        string     `json:"note,omitempty"`
	Status      string     `json:"status"`
	Token       string     `json:"token,omitempty"`
	Error       string     `json:"error,omitempty"`
	Meta        *Meta      `json:"meta,omitempty"`
	Summary     string     `json:"summary,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	DecidedAt   time.Time  `json:"decidedAt,omitzero"`
	StartedAt   time.Time  `json:"startedAt,omitzero"`
	EndsAt      time.Time  `json:"endsAt,omitzero"`
	FinishedAt  time.Time  `json:"finishedAt,omitzero"`
}

func (t Test) Public() Test {
	t.Token = ""
	return t
}

func Valid(t Test) bool {
	if !objectives[t.Objective] || !audiences[t.Audience] || t.BudgetCents < MinBudgetCents || t.BudgetCents > MaxBudgetCents || t.Days < MinDays || t.Days > MaxDays {
		return false
	}
	if len(t.Creatives) < MinCreatives || len(t.Creatives) > MaxCreatives {
		return false
	}
	if t.Objective == "vendas" && t.Link == "" {
		return false
	}
	if t.Event != "" && (t.Objective != "vendas" || !SalesEvents[t.Event]) {
		return false
	}
	for _, c := range t.Creatives {
		if c.Kind == "post" {
			if c.MediaID == "" {
				return false
			}
			continue
		}
		if !kinds[c.Kind] || c.Path == "" || c.SHA256 == "" || c.Text == "" || (c.Kind == "video" && c.Thumb == "") {
			return false
		}
	}
	return true
}

type Store struct {
	path string
	now  func() time.Time
	mu   sync.Mutex
}

func NewStore(path string) *Store {
	return &Store{path: path, now: time.Now}
}

func NewStoreWithClock(path string, now func() time.Time) *Store {
	return &Store{path: path, now: now}
}

func (s *Store) load() ([]Test, error) {
	body, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return []Test{}, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Test
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) persist(all []Test) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// messagesPool reports whether an objective spends the separate buying budget.
func messagesPool(objective string) bool {
	return objective == "mensagens"
}

// committed sums the budget of non-rejected tests created since `since` in the
// same pool as `objective` (mensagens alone; everything else together).
func committed(all []Test, since time.Time, objective string) int64 {
	var sum int64
	for _, t := range all {
		if t.Status != StatusRejected && !t.CreatedAt.Before(since) && messagesPool(t.Objective) == messagesPool(objective) {
			sum += t.BudgetCents
		}
	}
	return sum
}

var saoPaulo = func() *time.Location {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		return time.FixedZone("BRT", -3*60*60)
	}
	return loc
}()

func monthStart(now time.Time) time.Time {
	local := now.In(saoPaulo)
	return time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, saoPaulo)
}

// available is what a new test of `objective` can still spend.
func available(all []Test, now time.Time, objective string) int64 {
	week := committed(all, now.Add(-BudgetWindow), objective)
	if !messagesPool(objective) {
		return max(0, WeeklyCapCents-week)
	}
	month := committed(all, monthStart(now), objective)
	return max(0, min(MessagesWeeklyCapCents-week, MessagesMonthlyCapCents-month))
}

// Available is what is left this week for sales and followers tests.
func (s *Store) Available() (int64, error) {
	return s.AvailableFor("vendas")
}

// AvailableFor is what is left for a test of this objective (mensagens has its
// own weekly and monthly budget).
func (s *Store) AvailableFor(objective string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return 0, err
	}
	return available(all, s.now(), objective), nil
}

func (s *Store) Add(t Test) (Test, error) {
	if !Valid(t) {
		return Test{}, ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Test{}, err
	}
	if t.BudgetCents > available(all, s.now(), t.Objective) {
		return Test{}, ErrOverBudget
	}
	t.ID = randomHex(8)
	t.Token = randomHex(16)
	t.Status = StatusPending
	t.Error, t.Summary = "", ""
	t.Meta = nil
	t.CreatedAt = s.now()
	t.DecidedAt, t.StartedAt, t.EndsAt, t.FinishedAt = time.Time{}, time.Time{}, time.Time{}, time.Time{}
	all = append(all, t)
	if err := s.persist(all); err != nil {
		return Test{}, err
	}
	return t, nil
}

func (s *Store) List(status string) ([]Test, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return nil, err
	}
	out := make([]Test, 0, len(all))
	for _, t := range all {
		if status == "" || t.Status == status {
			out = append(out, t.Public())
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *Store) update(id string, change func(*Test) error) (Test, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Test{}, err
	}
	for i := range all {
		if all[i].ID != id {
			continue
		}
		if err := change(&all[i]); err != nil {
			return Test{}, err
		}
		if err := s.persist(all); err != nil {
			return Test{}, err
		}
		return all[i].Public(), nil
	}
	return Test{}, ErrNotFound
}

func (s *Store) Decide(id, token string, approve bool) (Test, error) {
	return s.update(id, func(t *Test) error {
		if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(t.Token)) != 1 {
			return ErrBadToken
		}
		if t.Status != StatusPending {
			return ErrNotPending
		}
		t.Status = StatusRejected
		if approve {
			t.Status = StatusApproved
		}
		t.DecidedAt = s.now()
		return nil
	})
}

func (s *Store) Started(id string, meta Meta, endsAt time.Time) (Test, error) {
	return s.update(id, func(t *Test) error {
		if t.Status != StatusApproved || meta.CampaignID == "" || meta.AdSetID == "" || len(meta.Ads) == 0 || endsAt.IsZero() {
			return ErrWrongStatus
		}
		t.Status = StatusRunning
		t.Meta = &meta
		t.StartedAt = s.now()
		t.EndsAt = endsAt
		t.Error = ""
		return nil
	})
}

func (s *Store) SetMeta(id string, meta Meta) (Test, error) {
	return s.update(id, func(t *Test) error {
		if t.Status != StatusRunning || t.Meta == nil || meta.CampaignID != t.Meta.CampaignID {
			return ErrWrongStatus
		}
		t.Meta = &meta
		return nil
	})
}

func (s *Store) Finished(id, summary string) (Test, error) {
	return s.update(id, func(t *Test) error {
		if t.Status != StatusRunning {
			return ErrWrongStatus
		}
		t.Status = StatusDone
		t.Summary = summary
		t.FinishedAt = s.now()
		return nil
	})
}

func (s *Store) Failed(id, msg string, meta *Meta) (Test, error) {
	return s.update(id, func(t *Test) error {
		if t.Status != StatusApproved && t.Status != StatusRunning {
			return ErrWrongStatus
		}
		t.Status = StatusFailed
		t.Error = msg
		if meta != nil {
			t.Meta = meta
		}
		t.FinishedAt = s.now()
		return nil
	})
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
