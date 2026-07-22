package compare

import (
	"testing"

	"opdeals/internal/game"
)

// Riftbound's three sources each spell the same printing differently: Liga
// "226S", TCGCSV "226*/221", MyP "226P/221". Set-scoped matching joins on the
// number, so all three must collapse to one key or the card is dropped from
// every deal list that mixes sources.
func TestRiftboundSignatureSpellingsAgree(t *testing.T) {
	m := MatcherFor(game.Riftbound())
	want := m.Key("226S", "Virtuoso (Signature) (226S)", "SFD")
	for _, spelling := range []struct{ number, name string }{
		{"226*/221", "Virtuoso (Signature)"},
		{"226P/221", "Virtuoso (Signature)"},
		{"226S/221", "Virtuoso (Signature)"},
	} {
		if got := m.Key(spelling.number, spelling.name, "SFD"); got != want {
			t.Errorf("Key(%q) = %q, want %q", spelling.number, got, want)
		}
	}
	if want != "SFD|226|sig" {
		t.Errorf("signature key = %q, want SFD|226|sig", want)
	}
}

// The alt-art and plain spellings must stay distinct from the signature key.
func TestRiftboundVariantKeysStayDistinct(t *testing.T) {
	m := MatcherFor(game.Riftbound())
	sig := m.Key("226P/221", "Virtuoso (Signature)", "SFD")
	alt := m.Key("226A/221", "Virtuoso (Alternate Art)", "SFD")
	plain := m.Key("226/221", "Virtuoso", "SFD")
	if sig == alt || sig == plain || alt == plain {
		t.Errorf("variant keys collided: sig=%q alt=%q plain=%q", sig, alt, plain)
	}
}
