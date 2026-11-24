package security

import (
	"testing"
)

func TestBlacklist_Memory(t *testing.T) {
	bl := NewBlacklist(nil)

	if bl.IsBlocked("189.40.12.5") {
		t.Fatalf("expected IP not to be blocked initially")
	}

	bl.Block("189.40.12.5", "Abusive rate limit")

	if !bl.IsBlocked("189.40.12.5") {
		t.Fatalf("expected IP to be blocked after Block()")
	}

	if !bl.IsBlocked("189.40.12.5:54321") {
		t.Fatalf("expected IP with port to be blocked")
	}

	if bl.Count() != 1 {
		t.Fatalf("expected count 1, got %d", bl.Count())
	}

	list := bl.List()
	if len(list) != 1 || list[0].IP != "189.40.12.5" {
		t.Fatalf("expected entry in list, got %+v", list)
	}

	ok := bl.Unblock("189.40.12.5")
	if !ok {
		t.Fatalf("expected unblock to return true")
	}

	if bl.IsBlocked("189.40.12.5") {
		t.Fatalf("expected IP not to be blocked after unblock")
	}

	if bl.Count() != 0 {
		t.Fatalf("expected count 0, got %d", bl.Count())
	}
}
