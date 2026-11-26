package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"opdeals/internal/orders"
)

var recoveryClock = time.Date(2026, 11, 26, 14, 0, 0, 0, saoPaulo)

// recoveryOrderAt monta um pedido cuja reserva venceu (ou vence) em expires. O
// pedido nasce ReservationTTL antes, então os casos não dependem do prazo da
// reserva.
func recoveryOrderAt(id string, expires time.Time, status orders.Status, phone string) orders.Order {
	created := expires.Add(-orders.ReservationTTL)
	return orders.Order{
		ID: id, Number: len(id), Status: status,
		Lines:         []orders.Line{{SKU: "x", Name: "Booster", Qty: 1, UnitBRL: 30}},
		Buyer:         orders.Buyer{Name: "Cliente Teste", Phone: phone, Email: id + "@exemplo.test"},
		PaymentMethod: orders.PaymentPix,
		CreatedAt:     created, UpdatedAt: created, ExpiresAt: expires,
	}
}

func candidateIDs(list []orders.Order) []string {
	ids := []string{}
	for _, o := range list {
		ids = append(ids, o.ID)
	}
	return ids
}

func TestRecoveryCandidatesRules(t *testing.T) {
	now := recoveryClock
	expired := recoveryOrderAt("expirado", now.Add(-90*time.Minute), orders.StatusPending, "62 99999-0001")
	tooSoon := recoveryOrderAt("cedo", now.Add(-10*time.Minute), orders.StatusPending, "62 99999-0002") // expirou há 10 min
	tooOld := recoveryOrderAt("velho", now.Add(-(19*time.Hour + 30*time.Minute)), orders.StatusPending, "62 99999-0003")
	stillPending := recoveryOrderAt("pendente", now.Add(20*time.Minute), orders.StatusPending, "62 99999-0004")
	failed := recoveryOrderAt("recusado", now.Add(-150*time.Minute), orders.StatusFailed, "62 99999-0005")
	failed.PaymentMethod = orders.PaymentCard
	failed.UpdatedAt = now.Add(-2 * time.Hour)
	amountMismatch := recoveryOrderAt("divergente", now.Add(-150*time.Minute), orders.StatusFailed, "62 99999-0006")
	amountMismatch.MP.Status = "approved"
	amountMismatch.UpdatedAt = now.Add(-2 * time.Hour)
	retried := recoveryOrderAt("tentou", now.Add(-150*time.Minute), orders.StatusPending, "62 99999-0007")
	retriedPaid := recoveryOrderAt("pagou", now.Add(-90*time.Minute), orders.StatusPaid, "(62) 99999-0007")
	notifiedBefore := recoveryOrderAt("avisado", now.Add(-(5*24*time.Hour - 30*time.Minute)), orders.StatusPending, "62 99999-0008")
	notifiedBefore.RecoveryNotifiedAt = now.Add(-5 * 24 * time.Hour)
	again := recoveryOrderAt("denovo", now.Add(-90*time.Minute), orders.StatusPending, "62 99999-0008")
	noPhone := recoveryOrderAt("semtelefone", now.Add(-90*time.Minute), orders.StatusPending, "")

	got := candidateIDs(recoveryCandidates([]orders.Order{
		expired, tooSoon, tooOld, stillPending, failed, amountMismatch, retried, retriedPaid, notifiedBefore, again, noPhone,
	}, now))
	want := map[string]bool{"expirado": true, "recusado": true}
	if len(got) != len(want) {
		t.Fatalf("candidatos = %v, quero só expirado e recusado", got)
	}
	for _, id := range got {
		if !want[id] {
			t.Fatalf("candidatos = %v, quero só expirado e recusado", got)
		}
	}
}

func TestRecoveryCandidatesOnePerBuyer(t *testing.T) {
	now := recoveryClock
	older := recoveryOrderAt("a", now.Add(-210*time.Minute), orders.StatusPending, "62 99999-0001")
	newer := recoveryOrderAt("b", now.Add(-90*time.Minute), orders.StatusPending, "62 99999-0001")
	if got := candidateIDs(recoveryCandidates([]orders.Order{older, newer}, now)); len(got) != 1 || got[0] != "b" {
		t.Fatalf("candidatos = %v, quero só o mais novo", got)
	}
}

func TestRecoveryReasons(t *testing.T) {
	now := recoveryClock
	pix := recoveryOrderAt("p", now.Add(-90*time.Minute), orders.StatusPending, "62 99999-0001")
	if _, reason, ok := failedAt(pix, now); !ok || reason != recoveryReasonPix {
		t.Fatalf("pix = %q %v", reason, ok)
	}
	card := recoveryOrderAt("c", now.Add(-90*time.Minute), orders.StatusFailed, "62 99999-0001")
	if _, reason, ok := failedAt(card, now); !ok || reason != recoveryReasonFail {
		t.Fatalf("cartão = %q %v", reason, ok)
	}
	paid := recoveryOrderAt("pg", now.Add(-90*time.Minute), orders.StatusPaid, "62 99999-0001")
	if _, _, ok := failedAt(paid, now); ok {
		t.Fatal("pedido pago não é candidato")
	}
}

func (f *checkoutFixture) seedExpired(t *testing.T, now time.Time) orders.Order {
	t.Helper()
	o, err := f.s.Checkout.Orders.Create(func(map[string]int, []orders.Order) (orders.Order, error) {
		o := recoveryOrderAt("", now.Add(-90*time.Minute), orders.StatusPending, "62 99999-0001")
		return o, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func TestRecoveryEndpointsOffByDefault(t *testing.T) {
	f := checkoutServer(t)
	f.s.postSale.now = func() time.Time { return recoveryClock }
	f.seedExpired(t, recoveryClock)
	admin := map[string]string{"X-Admin-Token": "admin"}

	t.Setenv(recoveryEnvFlag, "")
	t.Setenv(recoveryEnvText, "Oi {nome}")
	rec := f.do(t, http.MethodGet, "/api/recovery/orders", nil, admin)
	var out recoveryList
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || rec.Code != http.StatusOK || out.On || len(out.Orders) != 0 {
		t.Fatalf("desligado: %d %s", rec.Code, rec.Body)
	}
	if rec := f.do(t, http.MethodPost, "/api/recovery/orders/x/sent", nil, admin); rec.Code != http.StatusForbidden {
		t.Fatalf("marcar desligado = %d", rec.Code)
	}

	t.Setenv(recoveryEnvFlag, "1")
	t.Setenv(recoveryEnvText, "")
	rec = f.do(t, http.MethodGet, "/api/recovery/orders", nil, admin)
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || out.On || len(out.Orders) != 0 {
		t.Fatalf("sem texto aprovado devia continuar desligado: %s", rec.Body)
	}
	if rec := f.do(t, http.MethodGet, "/api/recovery/orders", nil, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("sem token = %d", rec.Code)
	}
}

func TestRecoveryEndpointsOn(t *testing.T) {
	f := checkoutServer(t)
	now := recoveryClock
	f.s.postSale.now = func() time.Time { return now }
	o := f.seedExpired(t, now)
	admin := map[string]string{"X-Admin-Token": "admin"}
	t.Setenv(recoveryEnvFlag, "1")
	t.Setenv(recoveryEnvText, "Oi, {nome}! Seu pedido #{numero} não foi pago: {link}")

	rec := f.do(t, http.MethodGet, "/api/recovery/orders", nil, admin)
	var out recoveryList
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || !out.On || len(out.Orders) != 1 {
		t.Fatalf("ligado: %d %s", rec.Code, rec.Body)
	}
	got := out.Orders[0]
	wantLink := "https://site.test/pedido/" + o.ID
	if got.Reason != recoveryReasonPix || got.Phone != "5562999990001" || got.Link != wantLink || got.FirstName != "Cliente" ||
		got.Message != "Oi, Cliente! Seu pedido #"+itoaTest(o.Number)+" não foi pago: "+wantLink {
		t.Fatalf("pedido = %+v", got)
	}

	if rec := f.do(t, http.MethodPost, "/api/recovery/orders/"+o.ID+"/sent", nil, admin); rec.Code != http.StatusOK {
		t.Fatalf("marcar = %d %s", rec.Code, rec.Body)
	}
	if rec := f.do(t, http.MethodPost, "/api/recovery/orders/"+o.ID+"/sent", nil, admin); rec.Code != http.StatusConflict {
		t.Fatalf("marcar de novo = %d", rec.Code)
	}
	rec = f.do(t, http.MethodGet, "/api/recovery/orders", nil, admin)
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || len(out.Orders) != 0 {
		t.Fatalf("depois de enviado devia sair da lista: %s", rec.Body)
	}

	night := time.Date(2026, 11, 26, 22, 30, 0, 0, saoPaulo)
	f.s.postSale.now = func() time.Time { return night }
	rec = f.do(t, http.MethodGet, "/api/recovery/orders", nil, admin)
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || !out.OutOfHours || len(out.Orders) != 0 {
		t.Fatalf("fora do horário: %s", rec.Body)
	}
}

func itoaTest(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func TestRecoveryAgentRoutes(t *testing.T) {
	f := checkoutServer(t)
	now := recoveryClock
	f.s.postSale.now = func() time.Time { return now }
	o := f.seedExpired(t, now)
	list := "/api/postsale/agent/recovery/orders"
	sent := list + "/" + o.ID + "/sent"
	t.Setenv(recoveryEnvFlag, "1")
	t.Setenv(recoveryEnvText, "Oi, {nome}! Pedido #{numero}: {link}")

	if rec := f.do(t, http.MethodGet, list, nil, agentAuth()); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("sem token de agente configurado = %d", rec.Code)
	}
	f.s.PostSale.AgentToken = testAgentToken
	if rec := f.do(t, http.MethodGet, list, nil, map[string]string{"X-Admin-Token": "admin"}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("token de admin na rota de agente = %d", rec.Code)
	}

	rec := f.do(t, http.MethodGet, list, nil, agentAuth())
	var out recoveryList
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || rec.Code != http.StatusOK || !out.On || len(out.Orders) != 1 || out.Orders[0].ID != o.ID || out.Orders[0].Message == "" {
		t.Fatalf("lista do agente: %d %s", rec.Code, rec.Body)
	}
	rec = f.do(t, http.MethodPost, sent, nil, agentAuth())
	var ok struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &ok); err != nil || rec.Code != http.StatusOK || !ok.OK {
		t.Fatalf("marcar pelo agente = %d %s", rec.Code, rec.Body)
	}
	if rec := f.do(t, http.MethodPost, sent, nil, agentAuth()); rec.Code != http.StatusConflict {
		t.Fatalf("marcar de novo pelo agente = %d", rec.Code)
	}

	t.Setenv(recoveryEnvFlag, "")
	rec = f.do(t, http.MethodGet, list, nil, agentAuth())
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || rec.Code != http.StatusOK || out.On || len(out.Orders) != 0 {
		t.Fatalf("desligado pelo agente: %d %s", rec.Code, rec.Body)
	}
}
