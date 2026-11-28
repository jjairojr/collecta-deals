package main

import (
	"os"
	"strings"
	"testing"

	"opdeals/internal/publishq"
)

func approvedReels(t *testing.T, f *fixture, db *chatDB, video string) requestOutput {
	t.Helper()
	out, errText := callTool[requestOutput](t, f, "pedir_aprovacao", reelsArgs(video))
	if errText != "" {
		t.Fatal(errText)
	}
	db.ceo(t, "Reels "+out.Codigo+" para hoje.")
	db.jairo(t, "aprovo")
	if _, errText := decide(t, f, out.Codigo, "aprovar", false); errText != "" {
		t.Fatal(errText)
	}
	return out
}

// Vídeo gravado pelo Jairo (28/11/2026): o mesmo .mp4 do Reels aprovado vai ao
// TikTok como vídeo, com a mesma legenda, sem montar slideshow.
func TestTiktokTakesTheSameReelsVideo(t *testing.T) {
	f := newFixture(t)
	db := newChatDB(t, f)
	var got []string
	f.app.slideshow = fakeSlideshow(&got)
	video := f.file(t, "gravado.mp4", "video gravado")
	ig := approvedReels(t, f, db, video)
	out, errText := callTool[requestOutput](t, f, "pedir_aprovacao", tiktokArgs(ig.Codigo))
	if errText != "" {
		t.Fatal(errText)
	}
	if len(got) != 0 {
		t.Fatalf("Reels não monta slideshow: %v", got)
	}
	approved, _ := f.store.List(publishq.StatusApproved)
	var tt, igItem publishq.Item
	for _, it := range approved {
		switch it.ID {
		case out.ID:
			tt = it
		case ig.ID:
			igItem = it
		}
	}
	if tt.Channel != "tiktok" || tt.Format != "video" || tt.Caption != igItem.Caption || len(tt.Files) != 1 || tt.Files[0].SHA256 != igItem.Files[0].SHA256 || !strings.Contains(tt.Note, ig.Codigo) {
		t.Fatalf("TikTok devia ser o mesmo vídeo, aprovado, com a mesma legenda: %+v", tt)
	}
	if _, errText := callTool[requestOutput](t, f, "pedir_aprovacao", tiktokArgs(ig.Codigo)); !strings.Contains(errText, "já foi mandado") {
		t.Fatalf("o mesmo Reels não vai duas vezes ao TikTok: %s", errText)
	}
}

func TestTiktokRefusesReelsChangedAfterApproval(t *testing.T) {
	f := newFixture(t)
	db := newChatDB(t, f)
	video := f.file(t, "gravado.mp4", "video gravado")
	ig := approvedReels(t, f, db, video)
	if err := os.WriteFile(video, []byte("outro video"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, errText := callTool[requestOutput](t, f, "pedir_aprovacao", tiktokArgs(ig.Codigo)); !strings.Contains(errText, "mudou depois da aprovação") {
		t.Fatalf("vídeo trocado depois da aprovação não pode ir: %s", errText)
	}
}
