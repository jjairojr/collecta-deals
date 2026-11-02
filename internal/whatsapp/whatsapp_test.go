package whatsapp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPhoneKeyMatchesWithAndWithoutNinthDigit(t *testing.T) {
	cases := map[string]string{
		"11987654321":     "1187654321",
		"5511987654321":   "1187654321",
		"551187654321":    "1187654321",
		"(11) 98765-4321": "1187654321",
		"1132654321":      "1132654321",
	}
	for in, want := range cases {
		if got := PhoneKey(in); got != want {
			t.Errorf("PhoneKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBrazilE164(t *testing.T) {
	if got := BrazilE164("(11) 98765-4321"); got != "5511987654321" {
		t.Fatalf("got %q", got)
	}
	if got := BrazilE164("5511987654321"); got != "5511987654321" {
		t.Fatalf("got %q", got)
	}
}

func TestTemplateSafeStripsNewlinesAndRuns(t *testing.T) {
	if got := TemplateSafe("a\nb\t c     d"); got != "a b c d" {
		t.Fatalf("got %q", got)
	}
}

func TestParseWebhook(t *testing.T) {
	body := []byte(`{"object":"whatsapp_business_account","entry":[{"changes":[{"field":"messages","value":{
		"contacts":[{"wa_id":"551187654321","profile":{"name":"João"}}],
		"messages":[
			{"id":"w1","from":"551187654321","timestamp":"1700000000","type":"text","text":{"body":"tem a zoro?"}},
			{"id":"w2","from":"551187654321","timestamp":"1700000001","type":"button","button":{"text":"Quero"}},
			{"id":"w3","from":"551187654321","timestamp":"1700000002","type":"image"}
		]}},{"field":"messages","value":{"statuses":[{"id":"x"}]}}]}]}`)
	got, err := ParseWebhook(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].Text != "tem a zoro?" || got[0].Name != "João" || got[0].At.Unix() != 1700000000 {
		t.Fatalf("first = %+v", got[0])
	}
	if got[1].Text != "Quero" || got[2].Text != "[image]" {
		t.Fatalf("got %+v", got)
	}
}

func TestValidateSignature(t *testing.T) {
	body := []byte(`{"a":1}`)
	mac := hmac.New(sha256.New, []byte("s3cret"))
	mac.Write(body)
	header := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if err := ValidateSignature(body, header, "s3cret"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSignature(body, header, "other"); err == nil {
		t.Fatal("wrong secret accepted")
	}
	if err := ValidateSignature(body, "", "s3cret"); err == nil {
		t.Fatal("missing header accepted")
	}
}

func TestSendTemplate(t *testing.T) {
	var got outbound
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v23.0/PNID/messages" || r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("path %s auth %s", r.URL.Path, r.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Error(err)
		}
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.1"}]}`))
	}))
	defer srv.Close()
	c := New(srv.URL, "v23.0", "PNID", "tok")
	id, err := c.SendTemplate(context.Background(), "5511987654321", "collecta_pos_venda", "pt_BR", []string{"João", "linha\nquebrada"})
	if err != nil {
		t.Fatal(err)
	}
	if id != "wamid.1" {
		t.Fatalf("id %q", id)
	}
	if got.Template == nil || got.Template.Name != "collecta_pos_venda" || got.Template.Components[0].Parameters[1].Text != "linha quebrada" {
		t.Fatalf("payload %+v", got.Template)
	}
}

func TestSendErrorSurfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"Re-engagement message","code":131047}}`))
	}))
	defer srv.Close()
	_, err := New(srv.URL, "v23.0", "P", "t").SendText(context.Background(), "55", "oi")
	if err == nil || err.Error() != "whatsapp 131047: Re-engagement message" {
		t.Fatalf("err = %v", err)
	}
}
