package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"opdeals/internal/collecta"
	"opdeals/internal/game"
	"opdeals/internal/logx"
	"opdeals/internal/mercadopago"
	"opdeals/internal/orders"
	"opdeals/internal/shipping"
	"opdeals/internal/trades"
)

type Checkout struct {
	Orders          *orders.Store
	Gateway         mercadopago.Gateway
	Quoter          shipping.Quoter
	Coupons         collecta.Client
	StorefrontToken string
	WebhookSecret   string
	SiteURL         string
	APIURL          string
	FromCEP         string
	FreeShippingMin float64
	Log             *logx.Logger
}

const maxLineQty = 20

var (
	emailRe  = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	nonDigit = regexp.MustCompile(`\D`)
	saoPaulo = mustLocation("America/Sao_Paulo")
)

func mustLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return loc
}

type orderLineInput struct {
	SKU       string `json:"sku"`
	Qty       int    `json:"qty"`
	UnitCents int64  `json:"unitCents"`
}

type orderCreateInput struct {
	Lines    []orderLineInput `json:"lines"`
	Buyer    orders.Buyer     `json:"buyer"`
	Address  orders.Address   `json:"address"`
	Shipping struct {
		ServiceID int     `json:"serviceID"`
		PriceBRL  float64 `json:"priceBRL"`
		JoinOrder int     `json:"joinOrder"`
	} `json:"shipping"`
	BackURLBase    string   `json:"backURLBase"`
	CollectaUserID string   `json:"collectaUserId"`
	CouponIDs      []string `json:"couponIds"`
	PromoCode      string   `json:"promoCode"`
}

type skuEntry struct {
	game   string
	trades []trades.Trade
	lookup trades.PriceLookup
}

type skuIndex map[string]skuEntry

func (s *Server) buildSKUIndex() (skuIndex, error) {
	idx := make(skuIndex)
	add := func(gameID string, all []trades.Trade, lookup trades.PriceLookup) {
		for _, t := range all {
			if t.Status != "holding" || !t.Listed || t.AskBRL <= 0 {
				continue
			}
			key := trades.SKU(gameID, t)
			e := idx[key]
			e.game = gameID
			e.lookup = lookup
			e.trades = append(e.trades, t)
			idx[key] = e
		}
	}
	for _, id := range s.orderedGameIDs() {
		gs := s.games[id]
		if gs.Trades == nil {
			continue
		}
		all, err := gs.Trades.List()
		if err != nil {
			return nil, err
		}
		lookup, _ := s.priceLookup(gs)
		add(gs.Game.ID, all, lookup)
	}
	if s.accessories != nil {
		all, err := s.accessories.List()
		if err != nil {
			return nil, err
		}
		add(accessoryGameID, all, nil)
	}
	return idx, nil
}

const cardImagePath = "/api/card-image?"

// One Piece is registered on the Liga first and imported from there, so each
// holding carries the art of its exact listing. A number-based guess can land on
// another print of the same number (a base Nami shown as the SP), so these games
// show the listing's art or none.
func listingArtOnly(gameID string) bool {
	return gameID == game.OnePiece().ID
}

func lineImage(ts []trades.Trade, gameID string, lookup trades.PriceLookup) string {
	for _, t := range ts {
		if t.ImageURL != "" {
			return t.ImageURL
		}
	}
	if len(ts) == 0 || listingArtOnly(gameID) {
		return ""
	}
	t := ts[0]
	if t.Kind != "" || t.Number == "" {
		return ""
	}
	q := url.Values{}
	q.Set("game", gameID)
	q.Set("set", t.Set)
	q.Set("number", t.Number)
	if lookup != nil {
		if _, u, ok := lookup(t.Number, t.Name, t.Set); ok {
			if id := productIDFromURL(u); id > 0 {
				q.Set("productID", strconv.Itoa(id))
			}
		}
	}
	return cardImagePath + q.Encode()
}

func legacyLineImage(l orders.Line) string {
	if l.ImageURL != "" {
		return l.ImageURL
	}
	if l.Kind != "single" || l.Game == accessoryGameID || listingArtOnly(l.Game) {
		return ""
	}
	parts := strings.Split(l.Meta, " · ")
	if len(parts) < 2 {
		return ""
	}
	q := url.Values{}
	q.Set("game", l.Game)
	q.Set("set", parts[0])
	q.Set("number", parts[1])
	return cardImagePath + q.Encode()
}

// listingArt maps each listing-art-only SKU to the art its ledger rows carry,
// sold rows included, so an order keeps showing the art of what was bought.
func (s *Server) listingArt() map[string]string {
	art := make(map[string]string)
	for id, gs := range s.games {
		if !listingArtOnly(id) || gs.Trades == nil {
			continue
		}
		all, err := gs.Trades.List()
		if err != nil {
			continue
		}
		for _, t := range all {
			key := trades.SKU(id, t)
			if _, ok := art[key]; !ok && t.ImageURL != "" {
				art[key] = t.ImageURL
			}
		}
	}
	return art
}

// orderLineImage replaces the number-based guess older orders stored for a
// listing-art-only game with the listing's own art.
func orderLineImage(l orders.Line, art map[string]string) string {
	if !listingArtOnly(l.Game) {
		return legacyLineImage(l)
	}
	if l.ImageURL != "" && !strings.HasPrefix(l.ImageURL, cardImagePath) {
		return l.ImageURL
	}
	return art[l.SKU]
}

func storefrontImage(raw string) string {
	if strings.HasPrefix(raw, cardImagePath) {
		return "/img?" + strings.TrimPrefix(raw, cardImagePath)
	}
	return raw
}

func (s *Server) ledgerForGame(game string) *trades.Store {
	if game == accessoryGameID {
		return s.accessories
	}
	if gs, ok := s.games[game]; ok {
		return gs.Trades
	}
	return nil
}

func lineMeta(t trades.Trade) string {
	parts := make([]string, 0, 4)
	for _, p := range []string{t.Set, t.Number, t.Variant, t.Condition} {
		if p != "" && p != "SEALED" && p != "ACESSORIO" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, " · ")
}

type lineProblem struct {
	SKU       string `json:"sku"`
	UnitCents int64  `json:"unitCents,omitempty"`
	Available int    `json:"available"`
}

type resolveResult struct {
	lines      []orders.Line
	priceDiffs []lineProblem
	shortages  []lineProblem
}

func resolveLines(in []orderLineInput, idx skuIndex, reserved map[string]int) (resolveResult, error) {
	var res resolveResult
	if len(in) == 0 {
		return res, errors.New("carrinho vazio")
	}
	seen := make(map[string]bool, len(in))
	for _, l := range in {
		if l.SKU == "" || seen[l.SKU] {
			return res, errors.New("item repetido ou sem sku")
		}
		seen[l.SKU] = true
		if l.Qty <= 0 || l.Qty > maxLineQty {
			return res, fmt.Errorf("quantidade inválida para %s", l.SKU)
		}
		e, ok := idx[l.SKU]
		if !ok {
			res.shortages = append(res.shortages, lineProblem{SKU: l.SKU})
			continue
		}
		unit, _ := orders.UnitPrice(e.trades)
		unitCents := int64(math.Round(unit * 100))
		if unitCents != l.UnitCents {
			res.priceDiffs = append(res.priceDiffs, lineProblem{SKU: l.SKU, UnitCents: unitCents})
		}
		available := orders.Available(e.trades) - reserved[l.SKU]
		if available < l.Qty {
			if available < 0 {
				available = 0
			}
			res.shortages = append(res.shortages, lineProblem{SKU: l.SKU, Available: available})
		}
		eligible := orders.Eligible(e.trades)
		first := eligible[0]
		kind := first.Kind
		if kind == "" {
			kind = "single"
		}
		res.lines = append(res.lines, orders.Line{
			SKU:      l.SKU,
			Game:     e.game,
			Kind:     kind,
			Name:     first.Name,
			Meta:     lineMeta(first),
			Qty:      l.Qty,
			UnitBRL:  round2(unit),
			ImageURL: lineImage(eligible, e.game, e.lookup),
		})
	}
	return res, nil
}

func itemsTotal(lines []orders.Line) float64 {
	sum := 0.0
	for _, l := range lines {
		sum += float64(l.Qty) * l.UnitBRL
	}
	return round2(sum)
}

func (s *Server) quoteShipping(ctx context.Context, cep string, lines []orders.Line) ([]shipping.Service, error) {
	services := make([]shipping.Service, 0, 5)
	if cep != "" && s.Checkout.Quoter != nil && s.Checkout.FromCEP != "" {
		items := make([]shipping.Item, 0, len(lines))
		for _, l := range lines {
			items = append(items, shipping.Item{Kind: l.Kind, Name: l.Name, Qty: l.Qty, UnitBRL: l.UnitBRL})
		}
		quoted, err := s.Checkout.Quoter.Quote(ctx, s.Checkout.FromCEP, cep, shipping.Pack(items))
		if err != nil {
			s.logCheckout("frete %s: %v", cep, err)
		}
		services = append(services, quoted...)
		if min := s.Checkout.FreeShippingMin; min > 0 && len(services) > 0 && itemsTotal(lines) >= min {
			services[0].PriceBRL = 0
		}
	}
	services = append(services, shipping.Pickup(), shipping.Join())
	return services, nil
}

func (s *Server) joinableOrder(number int, buyer orders.Buyer) (orders.Order, bool) {
	all, err := s.Checkout.Orders.List()
	if err != nil {
		return orders.Order{}, false
	}
	now := time.Now()
	for _, o := range all {
		if o.Number != number {
			continue
		}
		if orders.EffectiveStatus(o, now) != orders.StatusPaid {
			return orders.Order{}, false
		}
		if o.Buyer.CPF != buyer.CPF && o.Buyer.Email != buyer.Email {
			return orders.Order{}, false
		}
		return o, true
	}
	return orders.Order{}, false
}

func (s *Server) requireStorefront(w http.ResponseWriter, r *http.Request) bool {
	token := s.Checkout.StorefrontToken
	if token == "" {
		http.Error(w, "checkout desligado", http.StatusServiceUnavailable)
		return false
	}
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Storefront-Token")), []byte(token)) != 1 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if s.adminToken == "" {
		return true
	}
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Admin-Token")), []byte(s.adminToken)) != 1 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

func writeError(w http.ResponseWriter, status int, body map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleShippingQuote(w http.ResponseWriter, r *http.Request) {
	if !s.requireStorefront(w, r) {
		return
	}
	var in struct {
		CEP   string           `json:"cep"`
		Lines []orderLineInput `json:"lines"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	cep := digits(in.CEP)
	if cep != "" && len(cep) != 8 {
		writeError(w, http.StatusBadRequest, map[string]any{"error": "cep inválido"})
		return
	}
	idx, err := s.buildSKUIndex()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	res, err := resolveLines(in.Lines, idx, map[string]int{})
	if err != nil {
		writeError(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if len(res.lines) == 0 {
		writeError(w, http.StatusConflict, map[string]any{"error": "out_of_stock", "lines": res.shortages})
		return
	}
	services, err := s.quoteShipping(r.Context(), cep, res.lines)
	if err != nil {
		writeError(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"services": services})
}

func (s *Server) handleOrderCreate(w http.ResponseWriter, r *http.Request) {
	if !s.requireStorefront(w, r) {
		return
	}
	if s.Checkout.Gateway == nil {
		http.Error(w, "checkout desligado", http.StatusServiceUnavailable)
		return
	}
	var in orderCreateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if msg := validateBuyer(&in.Buyer); msg != "" {
		writeError(w, http.StatusBadRequest, map[string]any{"error": msg})
		return
	}
	local := in.Shipping.ServiceID < 0
	if local {
		in.Address = orders.Address{}
	} else if msg := validateAddress(&in.Address); msg != "" {
		writeError(w, http.StatusBadRequest, map[string]any{"error": msg})
		return
	}
	couponIDs, err := normalizeCouponIDs(in.CouponIDs)
	if err != nil {
		writeError(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	in.CollectaUserID = strings.TrimSpace(in.CollectaUserID)
	in.PromoCode = normalizePromoCode(in.PromoCode)
	var promoRule promo
	if in.PromoCode != "" {
		if in.CollectaUserID == "" {
			writeError(w, http.StatusBadRequest, map[string]any{"error": "cupom exige login"})
			return
		}
		if len(couponIDs) > 0 {
			writeError(w, http.StatusConflict, map[string]any{"error": "coupon_invalid", "message": promoNoStacking})
			return
		}
		rule, ok := lookupPromo(in.PromoCode)
		if !ok {
			writeError(w, http.StatusConflict, map[string]any{"error": "coupon_invalid", "message": promoUnknown})
			return
		}
		promoRule = rule
	}
	if len(couponIDs) > 0 {
		if in.CollectaUserID == "" {
			writeError(w, http.StatusBadRequest, map[string]any{"error": "cupom exige login"})
			return
		}
		if s.Checkout.Coupons == nil {
			writeError(w, http.StatusConflict, map[string]any{"error": "coupon_invalid", "message": "cupons indisponíveis no momento"})
			return
		}
	}
	idx, err := s.buildSKUIndex()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	preview, err := resolveLines(in.Lines, idx, map[string]int{})
	if err != nil {
		writeError(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if len(preview.priceDiffs) > 0 {
		writeError(w, http.StatusConflict, map[string]any{"error": "price_changed", "lines": preview.priceDiffs})
		return
	}
	if len(preview.shortages) > 0 {
		writeError(w, http.StatusConflict, map[string]any{"error": "out_of_stock", "lines": preview.shortages})
		return
	}
	services, err := s.quoteShipping(r.Context(), in.Address.CEP, preview.lines)
	if err != nil {
		writeError(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	var chosen *shipping.Service
	for i := range services {
		if services[i].ID == in.Shipping.ServiceID {
			chosen = &services[i]
			break
		}
	}
	if chosen == nil || math.Abs(chosen.PriceBRL-in.Shipping.PriceBRL) > 0.5 {
		writeError(w, http.StatusConflict, map[string]any{"error": "shipping_changed", "services": services})
		return
	}
	ship := orders.Shipping{
		ServiceID:    chosen.ID,
		ServiceName:  chosen.Name,
		Company:      chosen.Company,
		PriceBRL:     round2(chosen.PriceBRL),
		DeliveryDays: chosen.DeliveryDays,
	}
	if chosen.ID == shipping.JoinServiceID {
		base, ok := s.joinableOrder(in.Shipping.JoinOrder, in.Buyer)
		if !ok {
			writeError(w, http.StatusConflict, map[string]interface{}{"error": "join_not_found"})
			return
		}
		ship.JoinOrder = base.Number
		ship.Company = fmt.Sprintf("pedido #%d", base.Number)
		ship.DeliveryDays = base.Shipping.DeliveryDays
		in.Address = base.Address
	}

	superseded, err := s.Checkout.Orders.CancelPending(func(o orders.Order) bool {
		return o.Buyer.CPF == in.Buyer.CPF || o.Buyer.Email == in.Buyer.Email
	}, "substituído por uma nova tentativa do mesmo comprador")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, o := range superseded {
		s.releaseLater(r.Context(), o)
	}

	var conflict map[string]any
	order, err := s.Checkout.Orders.Create(func(reserved map[string]int, existing []orders.Order) (orders.Order, error) {
		idx, err := s.buildSKUIndex()
		if err != nil {
			return orders.Order{}, err
		}
		res, err := resolveLines(in.Lines, idx, reserved)
		if err != nil {
			return orders.Order{}, err
		}
		if len(res.priceDiffs) > 0 {
			conflict = map[string]any{"error": "price_changed", "lines": res.priceDiffs}
			return orders.Order{}, errors.New("price_changed")
		}
		if len(res.shortages) > 0 {
			conflict = map[string]any{"error": "out_of_stock", "lines": res.shortages}
			return orders.Order{}, errors.New("out_of_stock")
		}
		items := itemsTotal(res.lines)
		o := orders.Order{
			Lines:          res.lines,
			Buyer:          in.Buyer,
			Address:        in.Address,
			Shipping:       ship,
			Totals:         orders.Totals{ItemsBRL: items, ShippingBRL: ship.PriceBRL, TotalBRL: round2(items + ship.PriceBRL)},
			CollectaUserID: in.CollectaUserID,
		}
		if in.PromoCode == "" {
			return o, nil
		}
		if msg := promoRejection(promoRule, existing, in.CollectaUserID, in.Buyer.CPF, time.Now()); msg != "" {
			conflict = map[string]any{"error": "coupon_invalid", "message": msg}
			return orders.Order{}, errors.New("coupon_invalid")
		}
		itemCents := itemsCents(res.lines)
		discount := promoDiscountCents(itemCents, promoRule)
		o.Promo = &orders.Promo{Code: in.PromoCode, PercentBps: promoRule.PercentBps, ValueBRL: brl(discount)}
		o.Totals.DiscountBRL = brl(discount)
		o.Totals.TotalBRL = brl(itemCents + cents(ship.PriceBRL) - discount)
		return o, nil
	})
	if err != nil {
		if conflict != nil {
			writeError(w, http.StatusConflict, conflict)
			return
		}
		writeError(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if len(couponIDs) > 0 {
		var status int
		var body map[string]any
		order, status, body = s.reserveCoupons(r.Context(), order, couponIDs)
		if status != 0 {
			writeError(w, status, body)
			return
		}
	}

	site := strings.TrimRight(in.BackURLBase, "/")
	if site == "" {
		site = strings.TrimRight(s.Checkout.SiteURL, "/")
	}
	pref, err := s.Checkout.Gateway.CreatePreference(r.Context(), mercadopago.PreferenceInput{
		OrderID:         order.ID,
		Number:          order.Number,
		Items:           preferenceItems(order),
		Payer:           mercadopago.Payer{Name: order.Buyer.Name, Email: order.Buyer.Email, Phone: order.Buyer.Phone, CPF: order.Buyer.CPF},
		ReturnURL:       site + "/pedido/" + order.ID,
		NotificationURL: strings.TrimRight(s.Checkout.APIURL, "/") + "/api/orders/mp-webhook",
		ExpiresAt:       order.ExpiresAt,
	})
	if err != nil {
		cancelled, _ := s.Checkout.Orders.Update(order.ID, func(o *orders.Order) {
			o.Status = orders.StatusCancelled
			o.Notes = appendNote(o.Notes, "preferência MP falhou: "+err.Error())
		})
		s.releaseLater(r.Context(), cancelled)
		writeError(w, http.StatusBadGateway, map[string]any{"error": "pagamento indisponível no momento"})
		return
	}
	order, err = s.Checkout.Orders.Update(order.ID, func(o *orders.Order) {
		o.MP.PreferenceID = pref.ID
		o.MP.InitPoint = pref.InitPoint
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{
		"id":        order.ID,
		"number":    order.Number,
		"initPoint": order.MP.InitPoint,
		"expiresAt": order.ExpiresAt,
	})
}

func preferenceItems(o orders.Order) []mercadopago.Item {
	items := discountedItems(o.Lines, cents(o.Totals.DiscountBRL))
	if o.Shipping.PriceBRL > 0 {
		items = append(items, mercadopago.Item{
			ID:      "frete",
			Title:   "Frete — " + strings.TrimSpace(o.Shipping.Company+" "+o.Shipping.ServiceName),
			Qty:     1,
			UnitBRL: o.Shipping.PriceBRL,
		})
	}
	return items
}

func validateBuyer(b *orders.Buyer) string {
	b.Name = strings.Join(strings.Fields(b.Name), " ")
	b.Email = strings.TrimSpace(strings.ToLower(b.Email))
	b.Phone = digits(b.Phone)
	b.CPF = digits(b.CPF)
	if len(b.Name) < 3 {
		return "nome inválido"
	}
	if !emailRe.MatchString(b.Email) {
		return "e-mail inválido"
	}
	if len(b.Phone) < 10 || len(b.Phone) > 11 {
		return "telefone inválido"
	}
	if !validCPF(b.CPF) {
		return "cpf inválido"
	}
	return ""
}

func validateAddress(a *orders.Address) string {
	a.CEP = digits(a.CEP)
	a.Street = strings.TrimSpace(a.Street)
	a.Number = strings.TrimSpace(a.Number)
	a.Complement = strings.TrimSpace(a.Complement)
	a.District = strings.TrimSpace(a.District)
	a.City = strings.TrimSpace(a.City)
	a.State = strings.ToUpper(strings.TrimSpace(a.State))
	if len(a.CEP) != 8 {
		return "cep inválido"
	}
	if a.Street == "" || a.Number == "" || a.City == "" {
		return "endereço incompleto"
	}
	if len(a.State) != 2 {
		return "uf inválida"
	}
	return ""
}

func digits(s string) string {
	return nonDigit.ReplaceAllString(s, "")
}

func validCPF(cpf string) bool {
	if len(cpf) != 11 {
		return false
	}
	same := true
	for i := 1; i < 11; i++ {
		if cpf[i] != cpf[0] {
			same = false
			break
		}
	}
	if same {
		return false
	}
	check := func(n int) int {
		sum := 0
		for i := 0; i < n; i++ {
			sum += int(cpf[i]-'0') * (n + 1 - i)
		}
		d := (sum * 10) % 11
		if d == 10 {
			d = 0
		}
		return d
	}
	return check(9) == int(cpf[9]-'0') && check(10) == int(cpf[10]-'0')
}

func appendNote(notes, add string) string {
	if notes == "" {
		return add
	}
	return notes + "; " + add
}

type publicLine struct {
	Name     string  `json:"name"`
	Meta     string  `json:"meta,omitempty"`
	Kind     string  `json:"kind"`
	Qty      int     `json:"qty"`
	UnitBRL  float64 `json:"unitBRL"`
	ImageURL string  `json:"imageURL,omitempty"`
}

type publicCoupon struct {
	Code     string  `json:"code"`
	ValueBRL float64 `json:"valueBRL"`
}

type publicOrder struct {
	ID           string          `json:"id"`
	Number       int             `json:"number"`
	Status       orders.Status   `json:"status"`
	Lines        []publicLine    `json:"lines"`
	Totals       orders.Totals   `json:"totals"`
	Shipping     orders.Shipping `json:"shipping"`
	Coupons      []publicCoupon  `json:"coupons,omitempty"`
	TrackingCode string          `json:"trackingCode,omitempty"`
	InitPoint    string          `json:"initPoint,omitempty"`
	ExpiresAt    time.Time       `json:"expiresAt"`
	FirstName    string          `json:"firstName"`
	CreatedAt    time.Time       `json:"createdAt"`
}

func publicView(o orders.Order, now time.Time, art map[string]string) publicOrder {
	status := orders.EffectiveStatus(o, now)
	out := publicOrder{
		ID:           o.ID,
		Number:       o.Number,
		Status:       status,
		Lines:        make([]publicLine, 0, len(o.Lines)),
		Totals:       o.Totals,
		Shipping:     o.Shipping,
		TrackingCode: o.TrackingCode,
		ExpiresAt:    o.ExpiresAt,
		CreatedAt:    o.CreatedAt,
	}
	if parts := strings.Fields(o.Buyer.Name); len(parts) > 0 {
		out.FirstName = parts[0]
	}
	for _, c := range o.Coupons {
		out.Coupons = append(out.Coupons, publicCoupon{Code: c.Code, ValueBRL: c.ValueBRL})
	}
	if o.Promo != nil {
		out.Coupons = append(out.Coupons, publicCoupon{Code: o.Promo.Code, ValueBRL: o.Promo.ValueBRL})
	}
	for _, l := range o.Lines {
		out.Lines = append(out.Lines, publicLine{
			Name:     l.Name,
			Meta:     l.Meta,
			Kind:     l.Kind,
			Qty:      l.Qty,
			UnitBRL:  l.UnitBRL,
			ImageURL: storefrontImage(orderLineImage(l, art)),
		})
	}
	if status == orders.StatusPending || status == orders.StatusFailed {
		out.InitPoint = o.MP.InitPoint
	}
	return out
}

func (s *Server) handleOrderGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if pid := r.URL.Query().Get("payment_id"); pid != "" && s.Checkout.Gateway != nil {
		if n, err := strconv.ParseInt(pid, 10, 64); err == nil {
			if pay, err := s.Checkout.Gateway.GetPayment(r.Context(), n); err == nil && pay.ExternalReference == id {
				if err := s.applyPayment(r.Context(), pay); err != nil {
					s.logCheckout("sync payment %d: %v", n, err)
				}
			}
		}
	}
	o, err := s.Checkout.Orders.Get(id)
	if errors.Is(err, orders.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, publicView(o, time.Now(), s.listingArt()))
}

func (s *Server) logCheckout(format string, args ...any) {
	fmt.Printf("[CHECKOUT] "+format+"\n", args...)
}

type webhookBody struct {
	Type   string `json:"type"`
	Action string `json:"action"`
	Data   struct {
		ID string `json:"id"`
	} `json:"data"`
}

func (s *Server) handleMPWebhook(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var body webhookBody
	_ = json.NewDecoder(r.Body).Decode(&body)
	topic := q.Get("type")
	if topic == "" {
		topic = q.Get("topic")
	}
	if topic == "" {
		topic = body.Type
	}
	dataID := q.Get("data.id")
	if dataID == "" {
		dataID = q.Get("id")
	}
	if dataID == "" {
		dataID = body.Data.ID
	}
	if topic != "payment" {
		writeJSON(w, map[string]bool{"ok": true})
		return
	}
	if s.Checkout.WebhookSecret == "" {
		http.Error(w, "webhook secret not configured", http.StatusUnauthorized)
		return
	}
	if err := mercadopago.ValidateSignature(r.Header.Get("x-signature"), r.Header.Get("x-request-id"), dataID, s.Checkout.WebhookSecret); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, err := strconv.ParseInt(dataID, 10, 64)
	if err != nil {
		http.Error(w, "bad payment id", http.StatusBadRequest)
		return
	}
	if s.Checkout.Gateway == nil {
		http.Error(w, "checkout desligado", http.StatusServiceUnavailable)
		return
	}
	pay, err := s.Checkout.Gateway.GetPayment(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.applyPayment(r.Context(), pay); err != nil {
		s.logCheckout("apply payment %d: %v", id, err)
	} else {
		s.logCheckout("webhook payment %d %s → order %s", pay.ID, pay.Status, pay.ExternalReference)
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) applyPayment(ctx context.Context, pay mercadopago.Payment) error {
	if pay.ExternalReference == "" {
		return nil
	}
	o, err := s.Checkout.Orders.Transact(pay.ExternalReference, func(o *orders.Order) error {
		now := time.Now()
		switch orders.EffectiveStatus(*o, now) {
		case orders.StatusPaid, orders.StatusShipped, orders.StatusDelivered:
			if pay.ID != o.MP.PaymentID && pay.Status == "approved" {
				o.Notes = appendNote(o.Notes, fmt.Sprintf("pagamento duplicado %d", pay.ID))
			}
			if pay.ID == o.MP.PaymentID && (pay.Status == "refunded" || pay.Status == "charged_back") {
				o.Status = orders.StatusRefunded
				o.MP.Status = pay.Status
			}
			return nil
		case orders.StatusOversold, orders.StatusRefunded:
			if o.MP.PaymentID == 0 {
				o.MP.PaymentID = pay.ID
				o.MP.Status = pay.Status
				o.MP.Method = pay.Method
			}
			return nil
		case orders.StatusCancelled:
			if pay.Status != "approved" {
				return nil
			}
		}
		o.MP.PaymentID = pay.ID
		o.MP.Status = pay.Status
		o.MP.Method = pay.Method
		switch pay.Status {
		case "approved":
			return s.settle(o, pay, now)
		case "rejected", "cancelled":
			o.Status = orders.StatusFailed
		}
		return nil
	})
	if errors.Is(err, orders.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	s.releaseLater(ctx, o)
	return nil
}

func (s *Server) settle(o *orders.Order, pay mercadopago.Payment, now time.Time) error {
	if math.Abs(pay.AmountBRL-o.Totals.TotalBRL) >= 0.01 {
		o.Status = orders.StatusFailed
		o.Notes = appendNote(o.Notes, fmt.Sprintf("valor pago %.2f difere do total %.2f", pay.AmountBRL, o.Totals.TotalBRL))
		return nil
	}
	plan := make([][]orders.Pick, len(o.Lines))
	for i, l := range o.Lines {
		st := s.ledgerForGame(l.Game)
		if st == nil {
			o.Status = orders.StatusOversold
			o.Notes = appendNote(o.Notes, "ledger desconhecido para "+l.Game)
			return nil
		}
		all, err := st.List()
		if err != nil {
			return err
		}
		picks, ok := orders.Allocate(bySKU(all, l.Game, l.SKU), l.Qty)
		if !ok {
			o.Status = orders.StatusOversold
			o.Notes = appendNote(o.Notes, fmt.Sprintf("sem estoque para %s x%d", l.Name, l.Qty))
			return nil
		}
		plan[i] = picks
	}
	buyer := fmt.Sprintf("%s — Site #%d", o.Buyer.Name, o.Number)
	date := now.In(saoPaulo).Format("2006-01-02")
	for i, picks := range plan {
		l := &o.Lines[i]
		st := s.ledgerForGame(l.Game)
		for _, p := range picks {
			sold, err := st.Sell(p.TradeID, trades.Sale{
				Qty:          p.Qty,
				SellPrice:    round2(float64(p.Qty) * l.UnitBRL),
				SellCurrency: "BRL",
				SellDate:     date,
				Buyer:        buyer,
				Channel:      trades.ChannelSite,
			})
			if err != nil {
				o.Status = orders.StatusOversold
				o.Notes = appendNote(o.Notes, fmt.Sprintf("baixa falhou em %s: %v", p.TradeID, err))
				return nil
			}
			l.TradeIDs = append(l.TradeIDs, sold.ID)
			s.queueStockExit(l.Game, sold)
		}
	}
	if !o.CouponsReleasedAt.IsZero() {
		o.Notes = appendNote(o.Notes, "cupons liberados antes do pagamento")
		s.logCheckout("pedido #%d pago com cupons já liberados: %s", o.Number, strings.Join(codesOf(o.Coupons), ", "))
	}
	o.Status = orders.StatusPaid
	o.PaidAt = now
	return nil
}

func bySKU(all []trades.Trade, game, sku string) []trades.Trade {
	out := make([]trades.Trade, 0, 2)
	for _, t := range all {
		if trades.SKU(game, t) == sku {
			out = append(out, t)
		}
	}
	return out
}

func (s *Server) handleOrdersList(w http.ResponseWriter, r *http.Request) {
	if uid := strings.TrimSpace(r.URL.Query().Get("collectaUserId")); uid != "" {
		s.handleUserOrders(w, r, uid)
		return
	}
	if !s.requireAdmin(w, r) {
		return
	}
	all, err := s.Checkout.Orders.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	now := time.Now()
	art := s.listingArt()
	for i := range all {
		all[i].Status = orders.EffectiveStatus(all[i], now)
		for j := range all[i].Lines {
			all[i].Lines[j].ImageURL = orderLineImage(all[i].Lines[j], art)
		}
	}
	writeJSON(w, all)
}

func (s *Server) handleUserOrders(w http.ResponseWriter, r *http.Request, uid string) {
	if !s.requireStorefront(w, r) {
		return
	}
	all, err := s.Checkout.Orders.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	now := time.Now()
	art := s.listingArt()
	out := make([]publicOrder, 0)
	for _, o := range all {
		if o.CollectaUserID == uid {
			out = append(out, publicView(o, now, art))
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, out)
}

func (s *Server) handleOrderPatch(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	var in struct {
		Status       *string `json:"status"`
		TrackingCode *string `json:"trackingCode"`
		Notes        *string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var bad string
	o, err := s.Checkout.Orders.Transact(r.PathValue("id"), func(o *orders.Order) error {
		if in.TrackingCode != nil {
			o.TrackingCode = strings.TrimSpace(*in.TrackingCode)
		}
		if in.Notes != nil {
			o.Notes = strings.TrimSpace(*in.Notes)
		}
		if in.Status == nil {
			return nil
		}
		now := time.Now()
		current := orders.EffectiveStatus(*o, now)
		if err := orders.SetStatus(o, orders.Status(*in.Status), now); err != nil {
			bad = fmt.Sprintf("não dá para mudar de %s para %s", current, *in.Status)
			return err
		}
		return nil
	})
	if errors.Is(err, orders.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		if bad != "" {
			http.Error(w, bad, http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if in.Status != nil {
		s.releaseLater(r.Context(), o)
	}
	writeJSON(w, o)
}
