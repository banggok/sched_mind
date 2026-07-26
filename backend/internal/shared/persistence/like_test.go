package persistence

import "testing"

func TestEscapeLike(t *testing.T) {
	if got := EscapeLike(`a\b%c_d`); got != `a\\b\%c\_d` {
		t.Fatalf("unexpected escaped pattern: %q", got)
	}
}
