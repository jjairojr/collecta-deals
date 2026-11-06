package publishq

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	now := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)
	return NewStoreWithClock(filepath.Join(t.TempDir(), "publishq.json"), func() time.Time { return now })
}

func sample() Item {
	return Item{
		Channel: "instagram",
		Format:  "reels",
		Files:   []File{{Path: "/x/video.mp4", SHA256: "abc", Size: 10}},
		Caption: "Faltam 10 dias",
		PostAt:  time.Date(2026, 11, 6, 15, 0, 0, 0, time.UTC),
	}
}

func TestAddHidesTokenFromList(t *testing.T) {
	s := newTestStore(t)
	it, err := s.Add(sample())
	if err != nil {
		t.Fatal(err)
	}
	if it.ID == "" || len(it.Token) != 32 || it.Status != StatusPending {
		t.Fatalf("item errado: %+v", it)
	}
	list, err := s.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Token != "" {
		t.Fatalf("lista não devia expor o código: %+v", list)
	}
	if _, err := s.Add(Item{Channel: "instagram"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("pedido incompleto devia falhar: %v", err)
	}
	bad := sample()
	bad.Channel = "tiktok"
	if _, err := s.Add(bad); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("reels no tiktok devia falhar: %v", err)
	}
}

func TestDecideNeedsTokenAndOnlyOnce(t *testing.T) {
	s := newTestStore(t)
	it, _ := s.Add(sample())
	if _, err := s.Decide(it.ID, "errado", true); !errors.Is(err, ErrBadToken) {
		t.Fatalf("código errado devia falhar: %v", err)
	}
	if _, err := s.Decide(it.ID, "", true); !errors.Is(err, ErrBadToken) {
		t.Fatalf("código vazio devia falhar: %v", err)
	}
	got, err := s.Decide(it.ID, it.Token, true)
	if err != nil || got.Status != StatusApproved || got.DecidedAt.IsZero() || got.Token != "" {
		t.Fatalf("aprovação falhou: %+v %v", got, err)
	}
	if _, err := s.Decide(it.ID, it.Token, false); !errors.Is(err, ErrNotPending) {
		t.Fatalf("segunda decisão devia falhar: %v", err)
	}
	if _, err := s.Decide("nao-existe", it.Token, true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("id inexistente devia falhar: %v", err)
	}
}

func TestPublishOnlyApproved(t *testing.T) {
	s := newTestStore(t)
	it, _ := s.Add(sample())
	if _, err := s.Published(it.ID, "https://www.instagram.com/reel/x"); !errors.Is(err, ErrNotApproved) {
		t.Fatalf("pendente não pode ser publicado: %v", err)
	}
	s.Decide(it.ID, it.Token, true)
	got, err := s.Published(it.ID, "https://www.instagram.com/reel/x")
	if err != nil || got.Status != StatusPublished || got.Link == "" || got.PublishedAt.IsZero() {
		t.Fatalf("publicado errado: %+v %v", got, err)
	}
	rejected, _ := s.Add(sample())
	s.Decide(rejected.ID, rejected.Token, false)
	if _, err := s.Published(rejected.ID, "x"); !errors.Is(err, ErrNotApproved) {
		t.Fatalf("rejeitado não pode ser publicado: %v", err)
	}
}

func TestFailedRetriesUntilLimit(t *testing.T) {
	s := newTestStore(t)
	it, _ := s.Add(sample())
	s.Decide(it.ID, it.Token, true)
	for i := 1; i < MaxAttempts; i++ {
		got, err := s.Failed(it.ID, "tela não carregou", false)
		if err != nil || got.Status != StatusApproved || got.Attempts != i {
			t.Fatalf("tentativa %d: %+v %v", i, got, err)
		}
	}
	got, _ := s.Failed(it.ID, "tela não carregou", false)
	if got.Status != StatusFailed {
		t.Fatalf("depois de %d tentativas devia falhar de vez: %+v", MaxAttempts, got)
	}
	other, _ := s.Add(sample())
	s.Decide(other.ID, other.Token, true)
	if got, _ := s.Failed(other.ID, "arquivo mudou", true); got.Status != StatusFailed {
		t.Fatalf("falha permanente devia encerrar: %+v", got)
	}
}
