package orders

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"opdeals/internal/logx"
)

var ErrNotFound = errors.New("order not found")

type Status string

const (
	StatusPending   Status = "pending"
	StatusPaid      Status = "paid"
	StatusFailed    Status = "failed"
	StatusExpired   Status = "expired"
	StatusCancelled Status = "cancelled"
	StatusOversold  Status = "oversold"
	StatusShipped   Status = "shipped"
	StatusRefunded  Status = "refunded"
)

const ReservationTTL = 30 * time.Minute

type Line struct {
	SKU      string   `json:"sku"`
	Game     string   `json:"game"`
	Kind     string   `json:"kind,omitempty"`
	Name     string   `json:"name"`
	Meta     string   `json:"meta,omitempty"`
	Qty      int      `json:"qty"`
	UnitBRL  float64  `json:"unitBRL"`
	ImageURL string   `json:"imageURL,omitempty"`
	TradeIDs []string `json:"tradeIDs,omitempty"`
}

type Buyer struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
	CPF   string `json:"cpf"`
}

type Address struct {
	CEP        string `json:"cep"`
	Street     string `json:"street"`
	Number     string `json:"number"`
	Complement string `json:"complement,omitempty"`
	District   string `json:"district"`
	City       string `json:"city"`
	State      string `json:"state"`
}

type Shipping struct {
	ServiceID    int     `json:"serviceID"`
	ServiceName  string  `json:"serviceName"`
	Company      string  `json:"company"`
	PriceBRL     float64 `json:"priceBRL"`
	DeliveryDays int     `json:"deliveryDays"`
	JoinOrder    int     `json:"joinOrder,omitempty"`
}

type Totals struct {
	ItemsBRL    float64 `json:"itemsBRL"`
	ShippingBRL float64 `json:"shippingBRL"`
	DiscountBRL float64 `json:"discountBRL"`
	TotalBRL    float64 `json:"totalBRL"`
}

type Coupon struct {
	ID       string  `json:"id"`
	Code     string  `json:"code"`
	ValueBRL float64 `json:"valueBRL"`
}

type MP struct {
	PreferenceID string `json:"preferenceID,omitempty"`
	InitPoint    string `json:"initPoint,omitempty"`
	PaymentID    int64  `json:"paymentID,omitempty"`
	Status       string `json:"status,omitempty"`
	Method       string `json:"method,omitempty"`
}

type Order struct {
	ID     string `json:"id"`
	Number int    `json:"number"`
	Status Status `json:"status"`

	Lines    []Line   `json:"lines"`
	Buyer    Buyer    `json:"buyer"`
	Address  Address  `json:"address"`
	Shipping Shipping `json:"shipping"`
	Totals   Totals   `json:"totals"`
	MP       MP       `json:"mp"`

	CollectaUserID    string    `json:"collectaUserId,omitempty"`
	Coupons           []Coupon  `json:"coupons,omitempty"`
	CouponsReleasedAt time.Time `json:"couponsReleasedAt,omitempty"`

	TrackingCode string `json:"trackingCode,omitempty"`
	Notes        string `json:"notes,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	PaidAt    time.Time `json:"paidAt,omitempty"`
	ShippedAt time.Time `json:"shippedAt,omitempty"`
}

func EffectiveStatus(o Order, now time.Time) Status {
	if o.Status == StatusPending && !o.ExpiresAt.IsZero() && now.After(o.ExpiresAt) {
		return StatusExpired
	}
	return o.Status
}

func (o Order) Reserves(now time.Time) bool {
	return EffectiveStatus(o, now) == StatusPending
}

type Store struct {
	path string
	mu   sync.RWMutex
	log  *logx.Logger
}

func NewStore(path string, log *logx.Logger) *Store {
	return &Store{path: path, log: log}
}

func (s *Store) load() ([]Order, error) {
	body, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Order{}, nil
		}
		return nil, err
	}
	var out []Order
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) persist(all []Order) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) List() ([]Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all, err := s.load()
	if err != nil {
		return nil, err
	}
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})
	return all, nil
}

func (s *Store) Get(id string) (Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all, err := s.load()
	if err != nil {
		return Order{}, err
	}
	for _, o := range all {
		if o.ID == id {
			return o, nil
		}
	}
	return Order{}, ErrNotFound
}

func (s *Store) Reserved(now time.Time) (map[string]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all, err := s.load()
	if err != nil {
		return nil, err
	}
	return reserved(all, now), nil
}

func reserved(all []Order, now time.Time) map[string]int {
	out := make(map[string]int)
	for _, o := range all {
		if !o.Reserves(now) {
			continue
		}
		for _, l := range o.Lines {
			out[l.SKU] += l.Qty
		}
	}
	return out
}

func (s *Store) Create(build func(reserved map[string]int) (Order, error)) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Order{}, err
	}
	now := time.Now()
	o, err := build(reserved(all, now))
	if err != nil {
		return Order{}, err
	}
	number := 0
	for _, e := range all {
		if e.Number > number {
			number = e.Number
		}
	}
	o.ID = newID()
	o.Number = number + 1
	o.Status = StatusPending
	o.CreatedAt = now
	o.UpdatedAt = now
	if o.ExpiresAt.IsZero() {
		o.ExpiresAt = now.Add(ReservationTTL)
	}
	all = append(all, o)
	if err := s.persist(all); err != nil {
		return Order{}, err
	}
	return o, nil
}

func (s *Store) CancelPending(match func(Order) bool, note string) ([]Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	cancelled := make([]Order, 0)
	for i := range all {
		status := EffectiveStatus(all[i], now)
		if (status != StatusPending && status != StatusFailed) || !match(all[i]) {
			continue
		}
		all[i].Status = StatusCancelled
		all[i].Notes = note
		all[i].UpdatedAt = now
		cancelled = append(cancelled, all[i])
	}
	if len(cancelled) == 0 {
		return nil, nil
	}
	if err := s.persist(all); err != nil {
		return nil, err
	}
	return cancelled, nil
}

func (s *Store) Update(id string, mut func(*Order)) (Order, error) {
	return s.Transact(id, func(o *Order) error {
		mut(o)
		return nil
	})
}

func (s *Store) Transact(id string, fn func(*Order) error) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Order{}, err
	}
	for i := range all {
		if all[i].ID != id {
			continue
		}
		if err := fn(&all[i]); err != nil {
			return Order{}, err
		}
		all[i].ID = id
		all[i].UpdatedAt = time.Now()
		if err := s.persist(all); err != nil {
			return Order{}, err
		}
		return all[i], nil
	}
	return Order{}, ErrNotFound
}

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("150405.000000")))
	}
	return hex.EncodeToString(b[:])
}
